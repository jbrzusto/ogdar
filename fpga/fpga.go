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
	// DEBUG:	"fmt"
	"os"
	"reflect"
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
	BASE_ADDR              = 0x40100000           // Starting address of FPGA registers handling the Digdar module
	BASE_SIZE              = 0x1000               // The size of FPGA registers handling the Digdar module
	SIG_LEN                = SAMPLES_PER_BUFF     // Size of data buffer into which input signal is captured , must be 2^n!
	CMD_ARM_BIT            = 1                    // Bit index in FPGA Command register for arming the trigger
	CMD_RST_BIT            = 2                    // Bit index in FPGA Command register for resetting write state machine
	TRIG_SRC_TRIG_MASK     = 0xff                 // Bit mask in the trigger_source register for depicting the trigger source type.
	CHA_OFFSET             = 0x10000              // Offset to the memory buffer where signal on channel A is captured.
	CHB_OFFSET             = 0x20000              // Offset to the memory buffer where signal on channel B is captured.
	XCHA_OFFSET            = 0x30000              // Offset to the memory buffer where signal on slow channel A is captured.
	XCHB_OFFSET            = 0x40000              // Offset to the memory buffer where signal on slow channel B is captured.
	BPS_VID                = 14                   // bits per sample, video channel sample (fast ADC A)
	BPS_TRIG               = 14                   // bits per sample, trigger channel sample (fast ADC B))
	BPS_ARP                = 12                   // bits per sample, ARP channel sample (slow ADC A)
	BPS_ACP                = 12                   // bits per sample, ACP channel sample (slow ADC B)
)

// TrigType enumerates sources for a trigger, and flags for Armed (bit 8) and Fired (bit 9)
type TrigType uint32

const (
	TRG_NONE      TrigType = iota
	TRG_IMMEDIATE          // start acquisition immediately upon arming
	TRG_TRIG               // pulse on radar trigger channel
	TRG_ACP                // pulse on radar ACP channel
	TRG_ARP                // pulse on radar ARP channel
)

// Status are flags for FPGA status
type Status uint32

const (
	STATUS_ARMED     Status = 1 << iota // FPGA is ready to detect a trigger and begin capturing
	STATUS_CAPTURING                    // FPGA detected a trigger and is capturing
	STATUS_FIRED                        // FPGA detected a trigger and has finished capturing
)

// DigdarOption is a set of bit flags for the field OscFPGARegMem.Options
type DigdarOption uint32

const (
	DDOPT_AVERAGING    DigdarOption = 1 << iota // average samples when decimating
	DDOPT_USE_SUM                               // return sample sum, not average, for decimation rates <= 4
	DDOPT_NEGATE_VIDEO                          // invert video sample values
	DDOPT_COUNT_MODE                            // return ADC clock count instead of real video samples; for testing
)

// -------------<Types for FPGA Registers and BRAM>-------------------------------
//
// The following types are used only for singleton items stored in the
// FPGA BRAM, memory which is *not* managed by Go.  So we declare them
// `notinheap` by using the //go:notinheap pragma to prevent Go from
// inserting gc memory barrier code, which breaks things.
//
// -------------------------------------------------------------------------------

// Regs holds all the FPGA registers.
//
//  ** WARNING ** WARNING ** WARNING **
//
// This struct must match the FPGA code exactly!  In particular, the
// order and tags of fields in this struct is **CRITICAL**.  If you
// change them, you **MUST** re-run gen_verilog, then copy the
// generated_*.v files to proj/digdar/FPGA/release_1/fpga/code/rtl,
// and regenerate the FPGA bitstream and boot.bin using Vivado, then
// install that on the redpitaya/digdar image SD card.
//
// As an exception, it *is* safe to change just the 'desc:' component
// of the field tags.  These descriptions will appear in ogdar's web
// interface.

//go:notinheap
type regs struct {
	Command uint32 `reg:"command" mode:"p" desc:"Command Register: bit[0]: arm trigger; bit[1]: reset"`

	TrigSource uint32 `reg:"trig_source" mode:"rw" desc:"Trigger source: 0: don't trigger; 1: trigger immediately upon arming; 2: radar trigger pulse; 3: ACP pulse; 4: ARP pulse"`

	NumSamp uint32 `reg:"num_samp" mode:"rw" desc:"Number of Samples: number of samples to write after being triggered.  Must be even and in the range 2...16384."`

	DecRate uint32 `reg:"dec_rate" mode:"rw" desc:"Decimation Rate: number of input samples to consume for one output sample. 0...65536.  For rates 1, 2, 3 and 4, samples can be summed instead of decimated.  For rates 1, 2, 4, 8, 64, 1024, 8192 and 65536, samples can be averaged instead of decimated bits [31:17] - reserved"`

	Options uint32 `reg:"options" mode:"rw" desc:"Options: digdar-specific options; see type DigdarOption bit[0]: Average samples; bit[1]: Sum samples; bit[2]: Negate video; bit[3]: Counting mode"`

	TrigThreshExcite uint32 `reg:"trig_thresh_excite" mode:"rw" desc:"Trigger Excite Threshold: Trigger pulse is detected after trigger channel ADC value meets or exceeds this value (in direction away from the Trigger Relax Threshold).  -8192...8191"`

	TrigThreshRelax uint32 `reg:"trig_thresh_relax" mode:"rw" desc:"Trigger Relax Threshold: After a trigger pulse has been detected, the trigger channel ADC value must meet or exceed this value (in direction away from the Trigger Excite Threshold) before a trigger will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -8192...8191"`

	TrigDelay uint32 `reg:"trig_delay" mode:"rw" desc:"Trigger Delay: How long to wait after trigger is detected before starting to capture samples from the video channel.  The delay is in units of ADC clocks; i.e. the value is multiplied by 8 nanoseconds."`
	// Note: this usage of 'delay' is traditional for radar
	// digitizing but differs from the red pitaya scope usage,
	// which means "number of decimated ADC samples to acquire
	// after trigger is raised"

	TrigLatency uint32 `reg:"trig_latency" mode:"rw" desc:"Trigger Latency: how long to wait after trigger relaxation before allowing next excitation.  To further debounce the trigger signal, we can specify a minimum wait time between relaxation and excitation.  0...65535 (which gets multiplied by 8 nanoseconds)"`

	TrigCount uint32 `reg:"trig_count" mode:"r" is_wire:"y" desc:"Trigger Count: number of trigger pulses detected since last reset"`

	ACPThreshExcite uint32 `reg:"acp_thresh_excite" mode:"rw" desc:"ACP Excite Threshold: the AC Pulse is detected when the ACP channel value meets or exceeds this value (in direction away from the ACP Relax Threshold).  -2048...2047"`

	ACPThreshRelax uint32 `reg:"acp_thresh_relax" mode:"rw" desc:"ACP Relax Threshold: After an ACP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from acp_thresh_excite) before an ACP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048...2047"`

	ACPLatency uint32 `reg:"acp_latency" mode:"rw" desc:"ACP Latency: how long to wait after ACP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)"`

	ARPThreshExcite uint32 `reg:"arp_thresh_excite" mode:"rw" desc:"ARP Excite Threshold: the AR Pulse is detected when the ARP channel value meets or exceeds this value (in direction away from the ARP Relax Threshold).  -2048..2047"`

	ARPThreshRelax uint32 `reg:"arp_thresh_relax" mode:"rw" desc:"ARP Relax Threshold: After an ARP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from arp_thresh_excite) before an ARP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048..2047"`

	ARPLatency uint32 `reg:"arp_latency" mode:"rw" desc:"ARP Latency: how long to wait after ARP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)"`

	TrigClock uint64 `reg:"trig_clock" mode:"r" desc:"Trigger Clock: ADC clock count at last trigger pulse"`

	TrigPrevClock uint64 `reg:"trig_prev_clock" mode:"r" desc:"Previous Trigger Clock: ADC clock count at previous trigger pulse"`

	ACPClock uint64 `reg:"acp_clock" mode:"r" desc:"ACP Clock: ADC clock count at last ACP"`

	ACPPrevClock uint64 `reg:"acp_prev_clock" mode:"r" desc:"Previous ACP Clock: ADC clock count at previous ACP"`

	ARPClock uint64 `reg:"arp_clock" mode:"r" desc:"ARP Clock: ADC clock count at last ARP"`

	ARPPrevClock uint64 `reg:"arp_prev_clock" mode:"r" desc:"Previous ARP Clock: ADC clock count at previous ARP"`

	ACPCount uint32 `reg:"acp_count" mode:"r" is_wire:"y" desc:"ACP Count: number of Azimuth Count Pulses detected since last reset"`

	ARPCount uint32 `reg:"arp_count" mode:"r" is_wire:"y" desc:"ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset"`

	ACPPerARP uint32 `reg:"acp_per_arp" mode:"r" desc:"count of ACP between two most recent ARP"`

	ACPAtARP uint32 `reg:"acp_at_arp" mode:"r" desc:"ACP at ARP: ACP count at most recent ARP"`

	ClockSinceACPAtARP uint32 `reg:"clock_since_acp_at_arp" mode:"r" desc:"ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP"`

	TrigAtARP uint32 `reg:"trig_at_arp" mode:"r" desc:"Trig at ARP: Trigger count at most recent ARP"`

	Clocks uint64 `reg:"clocks" mode:"r" desc:"clocks: 64-bit count of ADC clock ticks since reset"`

	SavedTrigClock uint64 `reg:"saved_trig_clock" mode:"r" desc:"Trigger Clock: ADC clock count at last trigger pulse"`

	SavedTrigPrevClock uint64 `reg:"saved_trig_prev_clock" mode:"r" desc:"Previous Trigger Clock: ADC clock count at previous trigger pulse"`

	SavedACPClock uint64 `reg:"saved_acp_clock" mode:"r" desc:"ACP Clock: ADC clock count at last ACP"`

	SavedACPPrevClock uint64 `reg:"saved_acp_prev_clock" mode:"r" desc:"Previous ACP Clock: ADC clock count at previous ACP"`

	SavedARPClock uint64 `reg:"saved_arp_clock" mode:"r" desc:"ARP Clock: ADC clock count at last ARP"`

	SavedARPPrevClock uint64 `reg:"saved_arp_prev_clock" mode:"r" desc:"Previous ARP Clock: ADC clock count at previous ARP"`

	SavedTrigCount uint32 `reg:"saved_trig_count" mode:"r" desc:"Trigger Count: number of trigger pulses detected since last reset"`

	SavedACPCount uint32 `reg:"saved_acp_count" mode:"r" desc:"ACP Count: number of Azimuth Count Pulses detected since last reset"`

	SavedARPCount uint32 `reg:"saved_arp_count" mode:"r" desc:"ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset"`

	SavedACPPerARP uint32 `reg:"saved_acp_per_arp" mode:"r" desc:"count of ACP between two most recent ARP"`

	SavedACPAtARP uint32 `reg:"saved_acp_at_arp" mode:"r" desc:"ACP at ARP: ACP count at most recent ARP"`

	SavedClockSinceACPAtARP uint32 `reg:"saved_clock_since_acp_at_arp" mode:"r" desc:"ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP"`

	SavedTrigAtARP uint32 `reg:"saved_trig_at_arp" mode:"r" desc:"Trig at ARP: Trigger count at most recent ARP"`

	ADCCounter uint32 `reg:"adc_counter" mode:"r" desc:"ADC Counter: 14-bit ADC counter used in counting mode; starts at 0 upon triggering, and increments at each ADC clock"`

	ACPRaw uint32 `reg:"acp_raw" mode:"r" desc:"most recent slow ADC value from ACP"`

	ARPRaw uint32 `reg:"arp_raw" mode:"r" desc:"most recent slow ADC value from ARP"`

	Status uint32 `reg:"status" mode:"r" desc:"Status: bit[0]: armed; bit[1]: capturing; bit[2]: fired"`
}

// RegsU32 allows access to the registers as an array of uint32
//go:notinheap
type regsU32 uint32

// VidBuf holds the video (Channel A) samples in the FPGA's BRAM buffer
//go:notinheap
type vidBuf [SAMPLES_PER_BUFF]uint32

// TrigBuf holds the trigger (Channel B) samples in the FPGA's BRAM buffer
//go:notinheap
type trigBuf [SAMPLES_PER_BUFF]uint32

// ACPBuf holds the ACP (slow Channel A) samples in the FPGA's BRAM buffer
//go:notinheap
type acpBuf [SAMPLES_PER_BUFF]uint32

// ARPBuf holds the ARP (slow Channel B) samples in the FPGA's BRAM buffer
//go:notinheap
type arpBuf [SAMPLES_PER_BUFF]uint32

// regPtr holds a pointer to a (non-heap) FPGA register
//go:notinheap
type regPtr *uint32

// -------------</ Types for FPGA Registers and BRAM>-----------------------------

// FPGA holds the redpitaya FPGA object.
var (
	Regs      *regs          // pointer to reg structure; will be filled in from mmap()
	RegsU32   *regsU32       // regs as an array of uint32 (pointer to first element, actually)
	regSlice  []byte         // registers as a byte slice
	vidSlice  []byte         // video buffer as a byte slice; VidBuf points to vidSlice[0]
	trigSlice []byte         // trigger buffer as a byte slice
	acpSlice  []byte         // ACP buffer as a byte slice
	arpSlice  []byte         // ARP buffer as a byte slice
	VidBuf    *vidBuf        // video sample buffer; these are the radar "data"
	TrigBuf   *trigBuf       // trigger sample buffer; used when configuring digitizer
	ARPBuf    *arpBuf        // ARP sample buffer; used when configuring digitizer
	ACPBuf    *acpBuf        // ACP sample buffer; used when configuring digitizer
	memfile   *os.File       // pointer to open file /dev/mem for mmaping registers
	RegMap    map[string]int // RegMap translates from the name of a parameter to its index in storage order (i.e. index in RegKeys)
	RegKeys   []string       // RegKeys is a slice of names of registers (keys to RegMap), sorted in storage order
	RegIndex  []uintptr      // RegIndex is a slice of indexes (into RegsU32) of the FPGA registers in storage order
)

// GetRegByIndex returns the uint32 value of a register, given its index
// second return value is true on success
func GetRegByIndex(i int) (uint32, bool) {
	if i < 0 || i >= len(RegMap) {
		return 0, false
	}
	return *((*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(RegsU32)) + RegIndex[i]))), true
}

// GetRegByName returns the uint32 value of a register, given its name
// second return value is true on success
func GetRegByName(x string) (uint32, bool) {
	i, ok := RegMap[x]
	if !ok {
		return 0, false
	}
	return GetRegByIndex(i)
}

// SetRegByIndex sets the value of a register, given its index
// second return value is true on success
func SetRegByIndex(i int, v uint32) bool {
	if i < 0 || i >= len(RegMap) {
		return false
	}
	*((*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(RegsU32)) + RegIndex[i]))) = v
	return true
}

// SetRegByName sets the value of a register, given its name
// second return value is true on success
func SetRegByName(x string, v uint32) bool {
	i, ok := RegMap[x]
	if !ok {
		return false
	}
	return SetRegByIndex(i, v)
}

// NumRegs returns the number of defined FPGA registers; not all are readable and/or writable
func NumRegs() int {
	return len(RegMap)
}

// RegName returns the name of the ith register.
func RegName(i int) string {
	if i < 0 || i >= len(RegKeys) {
		return ""
	}
	return RegKeys[i]
}

// Open sets up pointers to Fpga memory-mapped registers and allocates buffers.
func Init() {
	var err error
	var t reflect.Type
	memfile, err = os.OpenFile("/dev/mem", os.O_RDWR, 0744)
	if err != nil {
		goto cleanup
	}
	regSlice, err = syscall.Mmap(int(memfile.Fd()), BASE_ADDR, BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	// DEBUG:	fmt.Printf("Got RegSlice=%v\n", unsafe.Pointer(&regSlice[0]))
	Regs = (*regs)(unsafe.Pointer(&regSlice[0]))
	RegsU32 = (*regsU32)(unsafe.Pointer(&regSlice[0]))
	vidSlice, err = syscall.Mmap(int(memfile.Fd()), BASE_ADDR+CHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	VidBuf = (*vidBuf)(unsafe.Pointer(&vidSlice[0]))
	trigSlice, err = syscall.Mmap(int(memfile.Fd()), BASE_ADDR+CHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	TrigBuf = (*trigBuf)(unsafe.Pointer(&trigSlice[0]))
	acpSlice, err = syscall.Mmap(int(memfile.Fd()), BASE_ADDR+XCHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	ACPBuf = (*acpBuf)(unsafe.Pointer(&acpSlice[0]))
	arpSlice, err = syscall.Mmap(int(memfile.Fd()), BASE_ADDR+XCHB_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	ARPBuf = (*arpBuf)(unsafe.Pointer(&arpSlice[0]))
	// names of Control registers in a standard order
	t = reflect.TypeOf(Regs).Elem()
	// DEBUG:	fmt.Println("Got typeof *regs")
	RegKeys = make([]string, 0, t.NumField())
	RegMap = make(map[string]int, t.NumField())
	RegIndex = make([]uintptr, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		switch f.Type.Size() {
		case 4:
			RegKeys = append(RegKeys, f.Name)
			RegMap[f.Name] = len(RegIndex)
			RegIndex = append(RegIndex, f.Offset)
		case 8:
			RegKeys = append(RegKeys, f.Name+"_lo", f.Name+"_hi")
			RegMap[f.Name+"_lo"] = len(RegIndex)
			RegIndex = append(RegIndex, f.Offset)
			RegMap[f.Name+"_hi"] = len(RegIndex)
			RegIndex = append(RegIndex, f.Offset+4)
		default:
			panic("unhandled field size in fpga.regs")
		}
	}
	// DEBUG:	fmt.Println("Got past making RegKeys/RegMap")
	return
cleanup:
	panic("Unable to set up fpga")
}

// // Close frees Fpga resources.  NB: when would this ever be needed??
// func (fpga *FPGA) Close() {
// 	if fpga.memfile == nil {
// 		return
// 	}
// 	_ = syscall.Munmap(fpga.arpSlice)
// 	_ = syscall.Munmap(fpga.acpSlice)
// 	_ = syscall.Munmap(fpga.trigSlice)
// 	_ = syscall.Munmap(fpga.vidSlice)
// 	_ = syscall.Munmap(fpga.RegSlice)
// 	fpga.ARPBuf = nil
// 	fpga.ACPBuf = nil
// 	fpga.TrigBuf = nil
// 	fpga.VidBuf = nil
// 	fpga.Regs = nil
// 	fpga.memfile.Close()
// 	fpga.memfile = nil
// }

// Reset tells the Fpga to restart digitizing
func Reset() {
	Regs.Command |= CMD_RST_BIT
}

// Arm tells the Fpga to start digitizing at the next trigger detection.
func Arm() {
	Regs.Command |= CMD_ARM_BIT
}

// SelectTrig chooses the source used to trigger data acquisition.
func SelectTrig(t TrigType) {
	Regs.TrigSource = uint32(t)
}

// SetDecim selects the Fpga ADC decimation rate.
// Valid decimation rates are 1..65536.
func SetDecim(decim uint32) bool {
	if decim < 1 || decim > 65536 {
		return false
	}
	Regs.DecRate = decim
	return true
}

// SetNumSamp sets the number of samples to acquire after a trigger.
// Must be in the range 1...SAMPLES_PER_BUFF
// returns true on success; false otherwise
func SetNumSamp(n uint32) bool {
	if n > SAMPLES_PER_BUFF || n < 1 {
		return false
	}
	Regs.NumSamp = n
	return true
}

// HasFired checks whether the Fpga has received a trigger and completed sample acquisition
// since the last call to Arm().
func HasFired() bool {
	return (Regs.Status & uint32(STATUS_FIRED)) == 0
}

// GetRegsPointerType returns a reflection object for the non-exported type `regs`
func GetRegsPointerType() reflect.Type {
	return reflect.TypeOf(new(*regs))
}
