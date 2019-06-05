package fpga

import (
	"os"
	"syscall"
	"unsafe"
)

// Definitions for the redpitaya FPGA (digdar build)
// ChA (Vid) & ChB (Trig) data - 14 lowest bits valid; starts from 0x10000 and
// 0x20000 and are each 16k samples long
// XChA (ACP) & XChB (ARP) data - 12 lowest bits valid; starts from 0x30000 and
// 0x40000 and are each 16k samples long

const (
	FAST_ADC_CLOCK          = 125E6       // Fast ADC sampling rate, Hz (Vid, Trig)
	FAST_ADC_SAMPLE_PERIOD  = 1.0 / FAST_ADC_CLOCK // Fast ADC sample period
	SLOW_ADC_CLOCK          = 1E5         // Slow ADC sampling rate, Hz (ACP, ARP)
	SLOW_ADC_SAMPLE_PERIOD  = 1.0 / SLOW_ADC_CLOCK // Slow ADC sample period
	SAMPLES_PER_BUFF        = 16 * 1024   // Number of samples in a signal buffer
	BUFF_SIZE_BYTES         = 4 * SAMPLES_PER_BUFF // Samples in buff are uint32, so 4 bytes big
	OSC_FPGA_BASE_ADDR      = 0x40100000  // Starting address of FPGA registers handling Oscilloscope module
	OSC_FPGA_BASE_SIZE      = 0x50000     // The size of FPGA registers handling Oscilloscope module
	OSC_FPGA_SIG_LEN        = SAMPLES_PER_BUFF // Size of data buffer into which input signal is captured , must be 2^n!
	OSC_FPGA_CONF_ARM_BIT   = 1           // Bit index in FPGA configuration register for arming the trigger
	OSC_FPGA_CONF_RST_BIT   = 2           // Bit index in FPGA configuration register for reseting write state machine
	OSC_FPGA_POST_TRIG_ONLY = 4           // Bit index in FPGA configuration register for only writing after a trigger is detected
	OSC_FPGA_TRIG_SRC_MASK  = 0x0000000f  // Bit mask in the trigger_source register for depicting the trigger source type.
	OSC_FPGA_CHA_THR_MASK   = 0x00003fff  // Bit mask in the cha_thr register for depicting trigger threshold on channel A.
	OSC_FPGA_CHB_THR_MASK   = 0x00003fff  // Bit mask in the cha_thr register for depicting trigger threshold on channel B.
	OSC_FPGA_TRIG_DLY_MASK  = 0xffffffff  // Bit mask in the trigger_delay register for depicting trigger delay.
	OSC_FPGA_CHA_OFFSET     = 0x10000     // Offset to the memory buffer where signal on channel A is captured.
	OSC_FPGA_CHB_OFFSET     = 0x20000     // Offset to the memory buffer where signal on channel B is captured.
	OSC_FPGA_XCHA_OFFSET    = 0x30000     // Offset to the memory buffer where signal on slow channel A is captured.
	OSC_FPGA_XCHB_OFFSET    = 0x40000     // Offset to the memory buffer where signal on slow channel B is captured.
	DIGDAR_FPGA_BASE_ADDR   = 0x40600000  // Starting address of FPGA registers handling the Digdar module.
	DIGDAR_FPGA_BASE_SIZE   = 0x0000B8    // The size of FPGA register block handling the Digdar module.
	BPS_VID         = 14 // bits per sample, video channel sample (fast ADC A)
	BPS_TRIG        = 14 // bits per sample, trigger channel sample (fast ADC B))
	BPS_ARP         = 12 // bits per sample, ARP channel sample (slow ADC A)
	BPS_ACP         = 12 // bits per sample, ACP channel sample (slow ADC B)
)

type OscFPGARegMem struct { // FPGA register structure for Oscilloscope core module.

	// This struct is a direct image of physical FPGA memory. It
	// provides direct read/write access to FPGA registers when it
	// is mmapped to OSC_FPGA_BASE_ADDR through /dev/mem.

	Conf uint32 //  Configuration:
	// bit     [0] - arm_trigger
	// bit     [1] - rst_wr_state_machine
	// bits [31:2] - reserved

	TrigSource uint32 //  Trigger source:
	// bits [ 3 : 0] - trigger source:
	//   1 - trig immediately
	//   2 - ChA positive edge
	//   3 - ChA negative edge
	//   4 - ChB positive edge
	//   5 - ChB negative edge
	//   6 - External trigger 0
	//   7 - External trigger 1
	//   8 - ASG positive edge
	//   9 - ASG negative edge
	//  10 - digdar counted trigger pulse
	//  11 - digdar acp pulse
	//  12 - digdar arp pulse
	// bits [31 : 4] -reserved

	ChaThr uint32 //  ChA threshold:
	// bits [13: 0] - ChA threshold
	// bits [31:14] - reserved

	ChbThr uint32 //  ChB threshold:
	// bits [13: 0] - ChB threshold
	// bits [31:14] - reserved

	NumSamp uint32 //  Number of samples to write after being triggered
	// bits [31: 0] - trigger delay
	// 32 bit number - how many decimated samples should be stored into a buffer.
	// (max 16k samples)

	DataDec uint32 //  Data decimation
	// bits [16: 0] - decimation factor, legal values:
	//   1, 2, 8, 64, 1024, 8192 65536
	//   If other values are written data is undefined
	// bits [31:17] - reserved

	WrPtrCur uint32 // where machine stopped writing after trigger
	// bits [13: 0] - index into
	// bits [31:14] - reserved

	WrPtrTrigger uint32 // where trigger was detected
	// bits [13: 0] - pointer
	// bits [31:14] - reserved

	ChaHystersis uint32 //  ChA & ChB hysteresis - both of the format:
	// bits [13: 0] - hysteresis threshold
	// bits [31:14] - reserved

	ChbHystersis uint32

	Other uint32 // @brief
	// bits [0] - enable signal average at decimation
	// bits [31:1] - reserved

	Reserved uint32

	ChaFiltAa uint32 // ChA Equalization filter
	// bits [17:0] - AA coefficient (pole)
	// bits [31:18] - reserved

	ChaFiltBb uint32 // ChA Equalization filter
	// bits [24:0] - BB coefficient (zero)
	// bits [31:25] - reserved

	ChaFiltKk uint32 // ChA Equalization filter
	// bits [24:0] - KK coefficient (gain)
	// bits [31:25] - reserved

	ChaFiltPp uint32 // ChA Equalization filter
	// bits [24:0] - PP coefficient (pole)
	// bits [31:25] - reserved

	ChbFiltAa uint32 // ChB Equalization filter
	// bits [17:0] - AA coefficient (pole)
	// bits [31:18] - reserved

	ChbFiltBb uint32 // ChB Equalization filter
	// bits [24:0] - BB coefficient (zero)
	// bits [31:25] - reserved

	ChbFiltKk uint32 // ChB Equalization filter
	// bits [24:0] - KK coefficient (gain)
	// bits [31:25] - reserved

	ChbFiltPp uint32 // ChB Equalization filter
	// bits [24:0] - PP coefficient (pole)
	// bits [31:25] - reserved

	DigdarExtraOptions uint32 // Extra options:
	// bit [0] - if 1, only record samples after trigger detected
	//            this serves to protect a digitized pulse, so that
	//            we can be reading it from BRAM into DRAM while the FPGA
	//            waits for and digitizes another pulse. (Provided the number
	//            of samples to be digitized is <= 1/2 the buffer size of 16 k samples)
	// bit [1] - if 1, ADC A negates values and returns in 2s complement; otherwise,
	//           values are returned as-is.
	// bit [2] - use 32-bit reads from buffers
	// bit [3] - use counting mode, not real ADC samples; for debugging only
	// bit [4] - return sample sum, not average, for decimation rates <= 4 (returns as 16-bit)
	// bits [31:2] - reserved

}

type OgdarFPGARegMem struct {

	// This struct is a direct image of physical FPGA memory. It
	// provides direct read/write access to FPGA registers when it
	// is mmapped to DIGDAR_FPGA_BASE_ADDR through /dev/mem.

	// --------------- TRIG -----------------

	TrigThreshExcite uint32 //  trig_thresh_excite: trigger excitation threshold
	//          Trigger is raised for one FPGA clock after trigger channel
	//          ADC value meets or exceeds this value (in direction away
	//          from trig_thresh_relax).
	// bits [13: 0] - threshold, signed
	// bit  [31:14] - reserved

	TrigThreshRelax uint32 //  trig_thresh_relax: trigger relaxation threshold
	//          After a trigger has been raised, the trigger channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from trig_thresh_excite) before a trigger will be raised again.
	//          (Serves to debounce signal in schmitt-trigger style).
	// bits [13: 0] - threshold, signed
	// bit  [31:14] - reserved

	TrigDelay uint32 //  trig_delay: (traditional) trigger delay.
	//          How long to wait after trigger is raised
	//          before starting to capture samples from Video channel.
	//          Note: this usage of 'delay' is traditional for radar digitizing
	//          but differs from the red pitaya scope usage, which means
	//          "number of decimated ADC samples to acquire after trigger is raised"
	// bits [31: 0] - unsigned wait time, in ADC clocks.

	TrigLatency uint32 //  trig_latency: how long to wait after trigger relaxation before
	//          allowing next excitation.
	//          To further debounce the trigger signal, we can specify a minimum
	//          wait time between relaxation and excitation.
	// bits [31: 0] - unsigned latency time, in ADC clocks.

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

	// --------------- ACP -----------------

	//  acp_thresh_excite: acp excitation threshold
	//          the acp pulse is detected and counted when the ACP slow ADC
	//          channel meets or exceeds this value in the direction away
	//          from acp_thresh_relax
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

	ACPThreshExcite uint32

	ACPThreshRelax uint32 //  acp_thresh_relax: acp relaxation threshold
	//          After an acp has been detected, the acp channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from acp_thresh_excite) before a acp will be detected again.
	//          (Serves to debounce signal in schmitt-acp style).
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

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

	// --------------- ARP -----------------

	//  arp_thresh_excite: arp excitation threshold
	//          the arp pulse is detected and counted when the ARP slow ADC
	//          channel meets or exceeds this value in the direction away
	//          from arp_thresh_relax
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

	ARPThreshExcite uint32

	ARPThreshRelax uint32 //  arp_thresh_relax: arp relaxation threshold
	//          After an arp has been detected, the arp channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from arp_thresh_excite) before a arp will be detected again.
	//          (Serves to debounce signal in schmitt-arp style).
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

	// --------------------- SAVED COPIES ----------------------------------------
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

	Clocks         uint64 //  clocks: 64-bit count of ADC clock ticks since reset
	ACPRaw         uint32 //  most recent slow ADC value from ACP
	ARPRaw         uint32 //  most recent slow ADC value from ARP
	ACPAtARP       uint32 //  acp_at_arp:  value of acp count at most recent arp pulse
	SavedACPAtARP  uint32 //  saved_acp_at_arp:  value at start of most recently captured pulse
	TrigAtARP      uint32 //  trig_at_arp:  value of trig count at most recent arp pulse
	SavedTrigAtARP uint32 //  saved_trig_at_arp:  value at start of most recently captured pulse
}

type OgdarFPGA struct {
	Osc *OscFPGARegMem  // Oscilloscope FPGA registers
	Ogd *OgdarFPGARegMem // Ogdar FPGA registers
	vidSlice [] byte // video buffer as a byte slice; VidBuf points to vidSlice[0]
	trigSlice [] byte // trigger buffer as a byte slice
	acpSlice [] byte // ACP buffer as a byte slice
	arpSlice [] byte // ARP buffer as a byte slice
	VidBuf *[SAMPLES_PER_BUFF] uint32 // video sample buffer; these are the radar "data"
	TrigBuf *[SAMPLES_PER_BUFF] uint32 // trigger sample buffer; used when configuring digitizer
	ARPBuf *[SAMPLES_PER_BUFF] uint32 // ARP sample buffer; used when configuring digitizer
	ACPBuf *[SAMPLES_PER_BUFF] uint32 // ACP sample buffer; used when configuring digitizer
	memfile *os.File // pointer to open file /dev/mem for mmaping registers

}

// /**
//  * GENERAL DESCRIPTION:
//  *
//  * This module initializes and provides for other SW modules the access to the
//  * FPGA OSC module. The oscilloscope memory space is divided to three parts:
//  *   - registers (control and status)
//  *   - input signal buffer (Channel A)
//  *   - input signal buffer (Channel B)
//  *
//  * This module maps physical address of the oscilloscope core to the logical
//  * address, which can be used in the GNU/Linux user-space. To achieve this,
//  * OSC_FPGA_BASE_ADDR from CPU memory space is translated automatically to
//  * logical address with the function mmap(). After all the initialization is done,
//  * other modules can use this module to controll oscilloscope FPGA core. Before
//  * any functions or functionality from this module can be used it needs to be
//  * initialized with osc_fpga_init() function. When this module is no longer
//  * needed osc_fpga_exit() must be called.
//  *
//  * FPGA oscilloscope state machine in various modes. Basic principle is that
//  * SW sets the machine, 'arm' the writting machine (FPGA writes from ADC to
//  * input buffer memory) and then set the triggers. FPGA machine is continue
//  * writting to the buffers until the trigger is detected plus the amount set
//  * in trigger delay register. For more detauled description see the FPGA OSC
//  * registers description.
//  *
//  * Nice example how to use this module can be seen in worker.c module.
//  */


func Open() (fpga *OgdarFPGA) {
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
	fpga.Ogd = (*OgdarFPGARegMem)(unsafe.Pointer(&mmap[0]))
	mmap, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR, OSC_FPGA_BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.Osc = (*OscFPGARegMem)(unsafe.Pointer(&mmap[0]))
	fpga.vidSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR + OSC_FPGA_CHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.VidBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.vidSlice[0]))
	fpga.trigSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR + OSC_FPGA_CHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.TrigBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.trigSlice[0]))
	fpga.acpSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR + OSC_FPGA_XCHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.ACPBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.acpSlice[0]))
	fpga.arpSlice, err = syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR + OSC_FPGA_XCHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.ARPBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&fpga.arpSlice[0]))
	return fpga
cleanup:
	fpga.Close()
	return nil
}

// free FPGA resources
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

// arm FPGA so it can start digitizing at next trigger
func (fpga *OgdarFPGA) Arm() {
	fpga.Osc.DigdarExtraOptions = 21 // 1: only buffer samples *after* being triggered; (no: 2: negate range of sample values); 4: double-width reads; 16: return sum if decim <= 4
	fpga.Osc.Conf |=  OSC_FPGA_CONF_ARM_BIT;
}

// Select FPGA trigger source
func (fpga *OgdarFPGA) SelectTrig(src uint32) {
	fpga.Osc.TrigSource = src
}

// Set FPGA ADC decimation rate
// decim must be a valid value for the FPGA build:
//  1, 2, 3, 4, 8, 64, 1024, 8192, 65536
func (fpga *OgdarFPGA) SetDecim(decim uint32) {
	fpga.Osc.DataDec = decim
}

// Set the number of samples to acquire after a trigger.
// Must be in the range 1...SAMPLES_PER_BUFF
func (fpga *OgdarFPGA) SetNumSamp(n uint32) {
	if n <= SAMPLES_PER_BUFF && n > 0 {
		fpga.Osc.NumSamp = n
	}
}

// Check whether FPGA has triggered (i.e. has digitized a pulse)
func (fpga *OgdarFPGA) HasTriggered() bool {
	return (fpga.Osc.TrigSource & OSC_FPGA_TRIG_SRC_MASK) == 0
}

// Return positions in sample buf of current write and last trigger.
func (fpga *OgdarFPGA) Pos() (curr uint32, trig uint32) {
	curr = fpga.Osc.WrPtrCur
	trig = fpga.Osc.WrPtrTrigger
	return
}
