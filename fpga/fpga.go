// Interface to the redpitaya FPGA (digdar build).
//
// FPGA registers and BRAM are accessed via mmap()ing segments of
// /dev/mem and coercing the returned []byte into pointers to structs,
// using unsafe.Pointer()
//
// The redpitaya FPGA reads four radar channels:
//
// - video: strength of received (reflected) radar signal; 14 bit
// samples from ADC channel A sampling at 125 MHz, for a base range
// resolution of ~ 1.2 metres
//
// - trigger: strength of radar trigger line; 14 bit samples from ADC
// channel B sampling at 125 MHz.  In normal operation, when a pulse
// is detected on this channel, the FPGA begins filling a BRAM buffer
// with samples from the video channel, until the specified number of
// samples have been acquired.  Acquisition can be delayed by a
// specified amount after trigger pulse detection to allow for the
// delay between its detection by the digitizer and the actual release
// of a microwave pulse from the magnetron (e.g. due to cable length).
//
// - ACP: strength of radar azimuth count pulse; 12 bit samples from
// XADC channel A, sampling at 100 kHz.  Pulses from this line are
// counted, and the count is recorded with each digitized pulse to
// provide the relative antenna azimuth.  On the Furuno FR- series
// (1955, 1965, 8252) there are 450 ACP pulses per antenna rotation.
//
// - ARP: strength of radar azimuth return pulse; 12 bit samples from
// XADC channel B, sampling at 100 kHz.  A pulse should be detected on
// this line once per antenna rotation, and the antenna's true azimuth
// at that instant is constant across restarts of the radar.  This
// allows conversion of the (relative) ACP count to an absolute
// (compass) azimuth.
//
// To help with calibration of thresholds for trigger, ACP and ARP
// pulses, acquisition can alternatively be trigged by ADC or ARP
// pulses, or immediately, and raw values from all four channels can
// be recorded in separate buffers simultaenously.  These can be
// displayed via the web scope interface.
package fpga

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	FAST_ADC_CLOCK          = 125E6                // Fast ADC sampling rate, Hz (Vid, Trig)
	FAST_ADC_SAMPLE_PERIOD  = 1.0 / FAST_ADC_CLOCK // Fast ADC sample period
	SLOW_ADC_CLOCK          = 1E5                  // Slow ADC sampling rate, Hz (ACP, ARP)
	SLOW_ADC_SAMPLE_PERIOD  = 1.0 / SLOW_ADC_CLOCK // Slow ADC sample period
	SAMPLES_PER_BUFF        = 16 * 1024            // Number of samples in a signal buffer
	BUFF_SIZE_BYTES         = 4 * SAMPLES_PER_BUFF // Samples in buff are uint32, so 4 bytes big
	OSC_FPGA_BASE_ADDR      = 0x40100000           // Starting address of FPGA registers handling Oscilloscope module
	OSC_FPGA_BASE_SIZE      = 0x50000              // The size of FPGA registers handling Oscilloscope module
	OSC_FPGA_SIG_LEN        = SAMPLES_PER_BUFF     // Size of data buffer into which input signal is captured , must be 2^n!
	OSC_FPGA_CONF_ARM_BIT   = 1                    // Bit index in FPGA configuration register for arming the trigger
	OSC_FPGA_CONF_RST_BIT   = 2                    // Bit index in FPGA configuration register for reseting write state machine
	OSC_FPGA_TRIG_SRC_MASK  = 0x0000000f           // Bit mask in the trigger_source register for depicting the trigger source type.
	OSC_FPGA_CHA_OFFSET     = 0x10000              // Offset to the memory buffer where signal on channel A is captured.
	OSC_FPGA_CHB_OFFSET     = 0x20000              // Offset to the memory buffer where signal on channel B is captured.
	OSC_FPGA_XCHA_OFFSET    = 0x30000              // Offset to the memory buffer where signal on slow channel A is captured.
	OSC_FPGA_XCHB_OFFSET    = 0x40000              // Offset to the memory buffer where signal on slow channel B is captured.
	DIGDAR_FPGA_BASE_ADDR   = 0x40600000           // Starting address of FPGA registers handling the Digdar module.
	DIGDAR_FPGA_BASE_SIZE   = 0x0000B8             // The size of FPGA register block handling the Digdar module.
	BPS_VID                 = 14                   // bits per sample, video channel sample (fast ADC A)
	BPS_TRIG                = 14                   // bits per sample, trigger channel sample (fast ADC B))
	BPS_ARP                 = 12                   // bits per sample, ARP channel sample (slow ADC A)
	BPS_ACP                 = 12                   // bits per sample, ACP channel sample (slow ADC B)
)

// TrigType enumerates sources for a trigger
type TrigType uint32

const (
	_             TrigType = iota
	TRG_IMMEDIATE          // start acquisition immediately upon arming
	TRG_TRIG               // pulse on radar trigger channel
	TRG_ACP                // pulse on radar ACP channel
	TRG_ARP                // pulse on radar ARP channel
)

// DigdarOption is a set of bit flags for the field OscFPGARegMem.DigdarOptions
type DigdarOption uint32

const (
	DDOPT_NEGATE_VIDEO = 1 << iota // invert video sample values
	DDOPT_COUNT_MODE               // return ADC clock count instead of real video samples; for testing
	DDOPT_USE_SUM                  // return sample sum, not average, for decimation rates <= 4
)

// OscRegs is a direct image of physical FPGA memory. It provides direct read/write access to FPGA registers when it is mmapped to
// OSC_FPGA_BASE_ADDR through /dev/mem.
type OscRegs struct {

	Command uint32 // Command register (offset 0x0000)
	// bit     [0] - arm_trigger
	// bit     [1] - rst_wr_state_machine
	// bits [31:2] - reserved

	TrigSource uint32 // Trigger source (offset 0x0004)
	// bits [ 3 : 0] - trigger source:
	//   0 - don't trigger
	//   1 - trigger immediately upon arming
	//   2 - digdar trigger pulse
	//   3 - digdar acp pulse
	//   4 - digdar arp pulse
	// bits [31 : 4] -reserved

	NumSamp uint32 //  Number of samples to write after being triggered (offset 0x0008)
	// bits [31: 0] - trigger delay
	// 32 bit number - how many decimated samples should be stored into a buffer.
	// (max 16k samples)

	DecRate uint32 //  Data decimation (offset 0x000C)
	// bits [16: 0] - decimation rate
	// For rates 1, 2, 3, 4, samples can be summed instead of decimated
	// For rates 1, 2, 4, 8, 64, 1024, 8192 65536, samples can be averaged instead of decimated
	// bits [31:17] - reserved

	Averaging uint32 // signal averaging (offset 0x0010)
	// bits [0] - enable signal average at decimation
	// bits [31:1] - reserved
	// Defaults to 0x1 in FPGA

	DigdarOptions DigdarOption // digdar-specific options; see type DigdarOption (offset 0x0014)

	ADCCounter uint32 // 14-bit ADC counter used in counting mode (offset 0x18)
}

// OgdarRegs is a direct image of physical FPGA memory. It provides direct read/write access to FPGA registers when it is mmapped
// to DIGDAR_FPGA_BASE_ADDR through /dev/mem.
type OgdarRegs struct {

	// TRIGGER
	//
	TrigThreshExcite uint32 //  trig_thresh_excite: trigger excitation threshold Trigger is raised for one FPGA clock after trigger
	//  channel ADC value meets or exceeds this value (in direction away from trig_thresh_relax).  bits [13:
	//  0] - threshold, signed bit [31:14] - reserved
	TrigThreshRelax uint32 //  trig_thresh_relax: trigger relaxation threshold After a trigger has been raised, the trigger channel
	//  ADC value must meet or exceeds this value (in direction away from trig_thresh_excite) before a
	//  trigger will be raised again.  (Serves to debounce signal in Schmitt trigger style).  bits [13: 0] -
	//  threshold, signed bit [31:14] - reserved
	TrigDelay uint32 //  trig_delay: (traditional) trigger delay.  How long to wait after trigger is raised before starting to
	//  capture samples from Video channel.  Note: this usage of 'delay' is traditional for radar digitizing but
	//  differs from the red pitaya scope usage, which means "number of decimated ADC samples to acquire after
	//  trigger is raised" bits [31: 0] - unsigned wait time, in ADC clocks.
	TrigLatency uint32 //  trig_latency: how long to wait after trigger relaxation before allowing next excitation.  To further
	//  debounce the trigger signal, we can specify a minimum wait time between relaxation and excitation.  bits
	//  [31: 0] - unsigned latency time, in ADC clocks.
	TrigCount uint32 //  trig_count: number of trigger pulses detected since last reset
	// bits [31: 0] - unsigned count of trigger pulses detected
	TrigClock_low uint32 //  trig_clock_low: ADC clock count at last trigger pulse
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count
	TrigClockHigh uint32 //  trig_clock_high: ADC clock count at last trigger pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count
	TrigPrevClockLow uint32 //  trig_prev_clock_low: ADC clock count at previous trigger pulse,
	//          so we can calculate trigger rate, regardless of capture rate
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count
	TrigPrevClockHigh uint32 //  trig_prev_clock_high: ADC clock count at previous trigger pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	// ACP
	//
	//  acp_thresh_excite: acp excitation threshold
	//          the acp pulse is detected and counted when the ACP slow ADC
	//          channel meets or exceeds this value in the direction away
	//          from acp_thresh_relax
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved
	ACPThreshExcite uint32
	ACPThreshRelax  uint32 //  acp_thresh_relax: acp relaxation threshold After an acp has been detected, the acp channel ADC value
	//  must meet or exceeds this value (in direction away from acp_thresh_excite) before a acp will be
	//  detected again.  (Serves to debounce signal in Schmitt trigger style).  bits [11: 0] - threshold, signed
	//  bit [31:14] - reserved
	ACPLatency uint32 //  acp_latency: how long to wait after acp relaxation before
	//          allowing next excitation.
	//          To further debounce the acp signal, we can specify a minimum
	//          wait time between relaxation and excitation.
	// bits [31: 0] - unsigned latency time, in ADC clocks.
	ACPCount uint32 //  acp_count: number of acp pulses detected since last reset
	// bits [31: 0] - unsigned count of acp pulses detected
	ACPClockLow uint32 //  acp_clock_low: ADC clock count at last acp pulse
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count
	ACPClockHigh uint32 //  acp_clock_high: ADC clock count at last acp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count
	ACPPrevClockLow uint32 //  acp_prev_clock_low: ADC clock count at previous acp pulse,
	//          so we can calculate acp rate, regardless of capture rate
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count
	ACPPrevClockHigh uint32 //  acp_prev_clock_high: ADC clock count at previous acp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	// ARP
	//  arp_thresh_excite: arp excitation threshold
	//          the arp pulse is detected and counted when the ARP slow ADC
	//          channel meets or exceeds this value in the direction away
	//          from arp_thresh_relax
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved
	ARPThreshExcite uint32
	ARPThreshRelax  uint32 //  arp_thresh_relax: arp relaxation threshold
	//          After an arp has been detected, the arp channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from arp_thresh_excite) before a arp will be detected again.
	//          (Serves to debounce signal in Schmitt trigger style).
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved
	ARPLatency uint32 //  arp_latency: how long to wait after arp relaxation before
	//          allowing next excitation.
	//          To further debounce the arp signal, we can specify a minimum
	//          wait time between relaxation and excitation.
	// bits [31: 0] - unsigned latency time, in ADC clocks.
	ARPCount uint32 //  arp_count: number of arp pulses detected since last reset
	// bits [31: 0] - unsigned count of arp pulses detected
	ARPClockLow uint32 //  arp_clock_low: ADC clock count at last arp pulse
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count
	ARPClockHigh uint32 //  arp_clock_high: ADC clock count at last arp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count
	ARPPrevClockLow uint32 //  arp_prev_clock_low: ADC clock count at previous arp pulse,
	//          so we can calculate arp rate, regardless of capture rate
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count
	ARPPrevClockHigh uint32 //  arp_prev_clock_high: ADC clock count at previous arp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count
	ACPPerARP uint32 //  acp_per_arp: count of ACP pulses between two most recent ARP pulses
	// bits [31: 0] - unsigned count of ACP pulses

	// Saved Copies
	//
	// For these metadata, we want to record the values at the time of the
	// most recently *captured* pulse.  So if the capture thread is not keeping up
	// with the radar, we still have correct values of these metadata for each
	// captured pulse (e.g. the value of the ACP count at each captured radar pulse).
	// The FPGA knows at trigger detection time whether or not
	// the pulse will be captured, and if so, copies the live metadata values to
	// these saved locations.
	SavedTrigCount         uint32 //  saved_trig_count:  value at start of most recently captured pulse
	SavedTrigClockLow      uint32 //  saved_trig_clock_low:  value at start of most recently captured pulse
	SavedTrigClockHigh     uint32 //  saved_trig_clock_high:  value at start of most recently captured pulse
	SavedTrigPrevClockLow  uint32 //  saved_trig_prev_clock_low:  value at start of most recently captured pulse
	SavedTrigPrevClockHigh uint32 //  saved_trig_prev_clock_high:  value at start of most recently captured pulse
	SavedACPCount          uint32 //  saved_acp_count:  value at start of most recently captured pulse
	SavedACPClockLow       uint32 //  saved_acp_clock_low:  value at start of most recently captured pulse
	SavedACPClockHigh      uint32 //  saved_acp_clock_high:  value at start of most recently captured pulse
	SavedACPPrevClockLow   uint32 //  saved_acp_prev_clock_low:  value at start of most recently captured pulse
	SavedACPPrevClockHigh  uint32 //  saved_acp_prev_clock_high:  value at start of most recently captured pulse
	SavedARPCount          uint32 //  saved_arp_count:  value at start of most recently captured pulse
	SavedARPClockLow       uint32 //  saved_arp_clock_low:  value at start of most recently captured pulse
	SavedARPClockHigh      uint32 //  saved_arp_clock_high:  value at start of most recently captured pulse
	SavedARPPrevClockLow   uint32 //  saved_arp_prev_clock_low:  value at start of most recently captured pulse
	SavedARPPrevClockHigh  uint32 //  saved_arp_prev_clock_high:  value at start of most recently captured pulse
	SavedACPPerARP         uint32 //  saved_acp_per_arp:  value at start of most recently captured pulse

	// Time and Azimuth Counters
	Clocks         uint64 //  clocks: 64-bit count of ADC clock ticks since reset
	ACPRaw         uint32 //  most recent slow ADC value from ACP
	ARPRaw         uint32 //  most recent slow ADC value from ARP
	ACPAtARP       uint32 //  acp_at_arp:  value of acp count at most recent arp pulse
	SavedACPAtARP  uint32 //  saved_acp_at_arp:  value at start of most recently captured pulse
	TrigAtARP      uint32 //  trig_at_arp:  value of trig count at most recent arp pulse
	SavedTrigAtARP uint32 //  saved_trig_at_arp:  value at start of most recently captured pulse
}

// OgdarFPGA represents the redpitaya FPGA object.
type OgdarFPGA struct {
	*OscRegs                             // Oscilloscope FPGA registers
	*OgdarRegs                           // Ogdar FPGA registers
	vidSlice   []byte                    // video buffer as a byte slice; VidBuf points to vidSlice[0]
	trigSlice  []byte                    // trigger buffer as a byte slice
	acpSlice   []byte                    // ACP buffer as a byte slice
	arpSlice   []byte                    // ARP buffer as a byte slice
	VidBuf     *[SAMPLES_PER_BUFF]uint32 // video sample buffer; these are the radar "data"
	TrigBuf    *[SAMPLES_PER_BUFF]uint32 // trigger sample buffer; used when configuring digitizer
	ARPBuf     *[SAMPLES_PER_BUFF]uint32 // ARP sample buffer; used when configuring digitizer
	ACPBuf     *[SAMPLES_PER_BUFF]uint32 // ACP sample buffer; used when configuring digitizer
	memfile    *os.File                  // pointer to open file /dev/mem for mmaping registers

}

// Open sets up pointers to FPGA memory-mapped registers and allocates buffers.
func New() (fpga *OgdarFPGA) {
	var err error
	fpga = new(OgdarFPGA)
	fpga.memfile, err = os.OpenFile("/dev/mem", os.O_RDWR, 0744)
	if err != nil {
		return nil
	}
	mmap, err := syscall.Mmap(int(fpga.memfile.Fd()), DIGDAR_FPGA_BASE_ADDR, DIGDAR_FPGA_BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.OgdarRegs = (*OgdarRegs)(unsafe.Pointer(&mmap[0]))
	mmap, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR, OSC_FPGA_BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.OscRegs = (*OscRegs)(unsafe.Pointer(&mmap[0]))
	fpga.vidSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR+OSC_FPGA_CHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.VidBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.vidSlice[0]))
	fpga.trigSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR+OSC_FPGA_CHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.TrigBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.trigSlice[0]))
	fpga.acpSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR+OSC_FPGA_XCHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.ACPBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.acpSlice[0]))
	fpga.arpSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR+OSC_FPGA_XCHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.ARPBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.arpSlice[0]))
	return fpga
cleanup:
	fpga.Close()
	return nil
}

// Close frees FPGA resources.
func (fpga *OgdarFPGA) Close() {
	if fpga.memfile == nil {
		return
	}
	_ = syscall.Munmap(fpga.arpSlice)
	_ = syscall.Munmap(fpga.acpSlice)
	_ = syscall.Munmap(fpga.trigSlice)
	_ = syscall.Munmap(fpga.vidSlice)
	fpga.ARPBuf = nil
	fpga.ACPBuf = nil
	fpga.TrigBuf = nil
	fpga.VidBuf = nil
	fpga.memfile.Close()
	fpga.memfile = nil
}

// Arm tells the FPGA to start digitizing at the next trigger detection.
func (fpga *OgdarFPGA) Arm() {
	fpga.DigdarOptions = DDOPT_NEGATE_VIDEO | DDOPT_USE_SUM
	fpga.Command |= OSC_FPGA_CONF_ARM_BIT
}

// SelectTrig chooses the source used to trigger data acquisition.
func (fpga *OgdarFPGA) SelectTrig(t TrigType) {
	fpga.TrigSource = uint32(t)
}

// SetDecim selects the FPGA ADC decimation rate.
// decim must be a valid value for the FPGA build:
//  1, 2, 3, 4, 8, 64, 1024, 8192, 65536
// returns true on success; false otherwise
func (fpga *OgdarFPGA) SetDecim(decim uint32) bool {
	switch decim {
	case 1, 2, 3, 4, 8, 64, 1024, 8192, 65536:
		fpga.DecRate = decim
		return true
	default:
		return false
	}
}

// SetNumSamp sets the number of samples to acquire after a trigger.
// Must be in the range 1...SAMPLES_PER_BUFF
// returns true on success; false otherwise
func (fpga *OgdarFPGA) SetNumSamp(n uint32) bool {
	if n <= SAMPLES_PER_BUFF && n > 0 {
		fpga.NumSamp = n
		return true
	}
	return false
}

// HasTriggered checks whether the FPGA has received a trigger and completed sample acquisition
// since the last call to Arm().
func (fpga *OgdarFPGA) HasTriggered() bool {
	return (fpga.TrigSource & OSC_FPGA_TRIG_SRC_MASK) == 0
}
