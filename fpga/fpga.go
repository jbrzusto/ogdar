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
	FAST_ADC_CLOCK         = 125E6                // Fast ADC sampling rate, Hz (Vid, Trig)
	FAST_ADC_SAMPLE_PERIOD = 1.0 / FAST_ADC_CLOCK // Fast ADC sample period
	SLOW_ADC_CLOCK         = 1E5                  // Slow ADC sampling rate, Hz (ACP, ARP)
	SLOW_ADC_SAMPLE_PERIOD = 1.0 / SLOW_ADC_CLOCK // Slow ADC sample period
	SAMPLES_PER_BUFF       = 16 * 1024            // Number of samples in a signal buffer
	BUFF_SIZE_BYTES        = 4 * SAMPLES_PER_BUFF // Samples in buff are uint32, so 4 bytes big
	BASE_ADDR              = 0x40100000           // Starting address of FPGA registers handling Oscilloscope module
	BASE_SIZE              = 0x50000              // The size of FPGA registers handling Oscilloscope module
	SIG_LEN                = SAMPLES_PER_BUFF     // Size of data buffer into which input signal is captured , must be 2^n!
	CONF_ARM_BIT           = 1                    // Bit index in FPGA configuration register for arming the trigger
	CONF_RST_BIT           = 2                    // Bit index in FPGA configuration register for reseting write state machine
	TRIG_SRC_MASK          = 0x0000000f           // Bit mask in the trigger_source register for depicting the trigger source type.
	CHA_OFFSET             = 0x10000              // Offset to the memory buffer where signal on channel A is captured.
	CHB_OFFSET             = 0x20000              // Offset to the memory buffer where signal on channel B is captured.
	XCHA_OFFSET            = 0x30000              // Offset to the memory buffer where signal on slow channel A is captured.
	XCHB_OFFSET            = 0x40000              // Offset to the memory buffer where signal on slow channel B is captured.
	BPS_VID                = 14                   // bits per sample, video channel sample (fast ADC A)
	BPS_TRIG               = 14                   // bits per sample, trigger channel sample (fast ADC B))
	BPS_ARP                = 12                   // bits per sample, ARP channel sample (slow ADC A)
	BPS_ACP                = 12                   // bits per sample, ACP channel sample (slow ADC B)
)

// TrigType enumerates sources for a trigger
type TrigType uint32

const (
	TRG_NONE      TrigType = iota
	TRG_IMMEDIATE          // start acquisition immediately upon arming
	TRG_TRIG               // pulse on radar trigger channel
	TRG_ACP                // pulse on radar ACP channel
	TRG_ARP                // pulse on radar ARP channel
)

// DigdarOption is a set of bit flags for the field OscFPGARegMem.Options
type DigdarOption uint32

const (
	DDOPT_AVERAGING    DigdarOption = 1 << iota // average samples when decimating
	DDOPT_USE_SUM                               // return sample sum, not average, for decimation rates <= 4
	DDOPT_NEGATE_VIDEO                          // invert video sample values
	DDOPT_COUNT_MODE                            // return ADC clock count instead of real video samples; for testing
)

// Control is a block of uint32 read/write FPGA registers that control operation of the digitizer.
type Control struct {
	Command uint32 `reg:"command" mode:"p" desc:"Command Register: bit[0]: arm trigger; bit[1]: reset"`

	TrigSource uint32 `reg:"trig_source" mode:"rw" desc:"Trigger source: 0: don't trigger; 1: trigger immediately upon arming; 2: radar trigger pulse; 3: ACP pulse; 4: ARP pulse"`

	NumSamp uint32 `reg:"num_samp" mode:"rw" desc:"Number of Samples: number of samples to write after being triggered.  Must be even and in the range 2...16384."`

	DecRate uint32 `reg:"dec_rate" mode:"rw" desc:"Decimation Rate: number of input samples to consume for one output sample. 0...65536.  For rates 1, 2, 3 and 4, samples can be summed instead of decimated.  For rates 1, 2, 4, 8, 64, 1024, 8192 and 65536, samples can be averaged instead of decimated bits [31:17] - reserved"`

	Options DigdarOption `reg:"options" mode:"rw" desc:"Options: digdar-specific options; see type DigdarOption bit[0]: Average samples; bit[1]: Sum samples; bit[2]: Negate video; bit[3]: Counting mode"`

	TrigThreshExcite uint32 `reg:"trig_thresh_excite" mode:"rw" desc:"Trigger Excite Threshold: Trigger pulse is detected after trigger channel ADC value meets or exceeds this value (in direction away from the Trigger Relax Threshold).  -8192...8191"`

	TrigThreshRelax uint32 `reg:"trig_thresh_relax" mode:"rw" desc:"Trigger Relax Threshold: After a trigger pulse has been detected, the trigger channel ADC value must meet or exceed this value (in direction away from the Trigger Excite Threshold) before a trigger will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -8192...8191"`

	TrigDelay uint32 `reg:"trig_delay" mode:"rw" desc:"Trigger Delay: How long to wait after trigger is detected before starting to capture samples from the video channel.  The delay is in units of ADC clocks; i.e. the value is multiplied by 8 nanoseconds."`
	// Note: this usage of 'delay' is traditional for radar
	// digitizing but differs from the red pitaya scope usage,
	// which means "number of decimated ADC samples to acquire
	// after trigger is raised"

	TrigLatency uint32 `reg:"trig_latency" mode:"rw" desc:"Trigger Latency: how long to wait after trigger relaxation before allowing next excitation.  To further debounce the trigger signal, we can specify a minimum wait time between relaxation and excitation.  0...65535 (which gets multiplied by 8 nanoseconds)"`

	ACPThreshExcite uint32 `reg:"acp_thresh_excite" mode:"rw" desc:"ACP Excite Threshold: the AC Pulse is detected when the ACP channel value meets or exceeds this value (in direction away from the ACP Relax Threshold).  -2048...2047"`

	ACPThreshRelax uint32 `reg:"acp_thresh_relax" mode:"rw" desc:"ACP Relax Threshold: After an ACP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from acp_thresh_excite) before an ACP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048...2047"`

	ACPLatency uint32 `reg:"acp_latency" mode:"rw" desc:"ACP Latency: how long to wait after ACP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)"`

	ARPThreshExcite uint32 `reg:"arp_thresh_excite" mode:"rw" desc:"ARP Excite Threshold: the AR Pulse is detected when the ARP channel value meets or exceeds this value (in direction away from the ARP Relax Threshold).  -2048..2047"`

	ARPThreshRelax uint32 `reg:"arp_thresh_relax" mode:"rw" desc:"ARP Relax Threshold: After an ARP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from arp_thresh_excite) before an ARP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048..2047"`

	ARPLatency uint32 `reg:"arp_thresh_latency" mode:"rw" desc:"ARP Latency: how long to wait after ARP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)"`
}

// Metadata is a block of uint32 read-only FPGA registers that
// provide metadata derived from radar signals and clocks.  Note:
// ordered to maintain a packed structure; the first member must be
// 64-bits wide and any 64-bit members must be separated by an even
// number (possibly zero) of 32-bit members.
type Metadata struct {
	TrigClock uint64 `reg:"trig_clock" mode:"r" desc:"Trigger Clock: ADC clock count at last trigger pulse"`

	TrigPrevClock uint64 `reg:"trig_prev_clock" mode:"r" desc:"Previous Trigger Clock: ADC clock count at previous trigger pulse"`

	ACPClock uint64 `reg:"acp_clock" mode:"r" desc:"ACP Clock: ADC clock count at last ACP"`

	ACPPrevClock uint64 `reg:"acp_prev_clock" mode:"r" desc:"Previous ACP Clock: ADC clock count at previous ACP"`

	ARPClock uint64 `reg:"arp_clock" mode:"r" desc:"ARP Clock: ADC clock count at last ARP"`

	ARPPrevClock uint64 `reg:"arp_prev_clock" mode:"r" desc:"Previous ARP Clock: ADC clock count at previous ARP"`

	TrigCount uint32 `reg:"trig_count" mode:"r" is_wire:"y" desc:"Trigger Count: number of trigger pulses detected since last reset"`

	ACPCount uint32 `reg:"acp_count" mode:"r" is_wire:"y" desc:"ACP Count: number of Azimuth Count Pulses detected since last reset"`

	ARPCount uint32 `reg:"arp_count" mode:"r" is_wire:"y" desc:"ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset"`

	ACPPerARP uint32 `reg:"acp_per_arp" mode:"r" desc:"count of ACP between two most recent ARP"`

	ADCCounter uint32 `reg:"adc_counter" mode:"r" desc:"ADC Counter: 14-bit ADC counter used in counting mode"`

	ACPAtARP uint32 `reg:"acp_at_arp" mode:"r" desc:"ACP at ARP: ACP count at most recent ARP"`

	ClockSinceACPAtARP uint32 `reg:"clock_since_acp_at_arp" mode:"r" desc:"ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP"`

	TrigAtARP uint32 `reg:"trig_at_arp" mode:"r" desc:"Trig at ARP: Trigger count at most recent ARP"`
}

// Misc are unbuffered miscellaneous registers
type Misc struct {
	Clocks uint64 `reg:"clocks" mode:"r" desc:"clocks: 64-bit count of ADC clock ticks since reset"`

	ACPRaw uint32 `reg:"acp_raw" mode:"r" desc:"most recent slow ADC value from ACP"`

	ARPRaw uint32 `reg:"arp_raw" mode:"r" desc:"most recent slow ADC value from ARP"`
}

// Regs holds all the FPGA registers
type Regs struct {
	Control           // Control Registers
	Metadata          // Metadata Regisers
	Misc              // Misc Registers
	AtTrig   Metadata `reg_prefix:"saved_"` // Metadata saved at last captured trigger pulse
}

// FPGA holds the redpitaya FPGA object.
type FPGA struct {
	*Regs                               // pointer to reg structure; will be filled in from mmap()
	regSlice  []byte                    // registers as a byte slice
	vidSlice  []byte                    // video buffer as a byte slice; VidBuf points to vidSlice[0]
	trigSlice []byte                    // trigger buffer as a byte slice
	acpSlice  []byte                    // ACP buffer as a byte slice
	arpSlice  []byte                    // ARP buffer as a byte slice
	VidBuf    *[SAMPLES_PER_BUFF]uint32 // video sample buffer; these are the radar "data"
	TrigBuf   *[SAMPLES_PER_BUFF]uint32 // trigger sample buffer; used when configuring digitizer
	ARPBuf    *[SAMPLES_PER_BUFF]uint32 // ARP sample buffer; used when configuring digitizer
	ACPBuf    *[SAMPLES_PER_BUFF]uint32 // ACP sample buffer; used when configuring digitizer
	memfile   *os.File                  // pointer to open file /dev/mem for mmaping registers
}

// Open sets up pointers to FPGA memory-mapped registers and allocates buffers.
func New() (fpga *FPGA) {
	var err error
	fpga = new(FPGA)
	fpga.memfile, err = os.OpenFile("/dev/mem", os.O_RDWR, 0744)
	if err != nil {
		return nil
	}
	fpga.regSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), BASE_ADDR, BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.Regs = (*Regs)(unsafe.Pointer(&fpga.regSlice))
	fpga.vidSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), BASE_ADDR+CHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.VidBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.vidSlice[0]))
	fpga.trigSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), BASE_ADDR+CHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.TrigBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.trigSlice[0]))
	fpga.acpSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), BASE_ADDR+XCHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.ACPBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.acpSlice[0]))
	fpga.arpSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), BASE_ADDR+XCHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
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
func (fpga *FPGA) Close() {
	if fpga.memfile == nil {
		return
	}
	_ = syscall.Munmap(fpga.arpSlice)
	_ = syscall.Munmap(fpga.acpSlice)
	_ = syscall.Munmap(fpga.trigSlice)
	_ = syscall.Munmap(fpga.vidSlice)
	_ = syscall.Munmap(fpga.regSlice)
	fpga.ARPBuf = nil
	fpga.ACPBuf = nil
	fpga.TrigBuf = nil
	fpga.VidBuf = nil
	fpga.Regs = nil
	fpga.memfile.Close()
	fpga.memfile = nil
}

// Arm tells the FPGA to start digitizing at the next trigger detection.
func (fpga *FPGA) Arm() {
	fpga.Command |= CONF_ARM_BIT
}

// SelectTrig chooses the source used to trigger data acquisition.
func (fpga *FPGA) SelectTrig(t TrigType) {
	fpga.TrigSource = uint32(t)
}

// SetDecim selects the FPGA ADC decimation rate.
// Valid decimation rates are 1..65536.
func (fpga *FPGA) SetDecim(decim uint32) bool {
	if decim < 1 || decim > 65536 {
		return false
	}
	fpga.DecRate = decim
	return true
}

// SetNumSamp sets the number of samples to acquire after a trigger.
// Must be in the range 1...SAMPLES_PER_BUFF
// returns true on success; false otherwise
func (fpga *FPGA) SetNumSamp(n uint32) bool {
	if n > SAMPLES_PER_BUFF || n < 1 {
		return false
	}
	fpga.NumSamp = n
	return true
}

// HasTriggered checks whether the FPGA has received a trigger and completed sample acquisition
// since the last call to Arm().
//
// Note: if called before the first call to Arm(), returns true
// even though there are no data available in the buffer.
func (fpga *FPGA) HasTriggered() bool {
	return (fpga.TrigSource & TRIG_SRC_MASK) == 0
}
