package ogdar

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Definitions for the redpitaya FPGA (digdar build)
//  ChA & ChB data - 14 lowest bits valid; starts from 0x10000 and
// 0x20000 and are each 16k samples long
// XChA & XChB data - 12 lowest bits valid; starts from 0x30000 and
// 0x40000 and are each 16k samples long

const (
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

	TriggerDelay uint32 //  After trigger delay:
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

type struct OgdarFPGA {
	OscFPGARegMem osc  // Oscilloscope FPGA registers
	OgdarFPGARegMem ogd // Ogdar FPGA registers
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

// /* internal structures */
// /** The FPGA register structure (defined in fpga_osc.h) */
// osc_fpga_reg_mem_t *g_osc_fpga_reg_mem = NULL;

// /* @brief Pointer to FPGA digdar control registers. */
// digdar_fpga_reg_mem_t *g_digdar_fpga_reg_mem = NULL;

// /** The FPGA input signal buffer pointer for channel A */
// uint32_t           *g_osc_fpga_cha_mem = NULL;
// /** The FPGA input signal buffer pointer for channel B */
// uint32_t           *g_osc_fpga_chb_mem = NULL;

// /** The FPGA input signal buffer pointer for slow channel A */
// uint32_t           *g_osc_fpga_xcha_mem = NULL;
// /** The FPGA input signal buffer pointer for slow channel B */
// uint32_t           *g_osc_fpga_xchb_mem = NULL;

// /** The memory file descriptor used to mmap() the FPGA space */
// int             g_osc_fpga_mem_fd = -1;

// /* Constants */
// /** ADC number of bits */
// const int c_osc_fpga_adc_bits = 14;

// /** Slow ADC number of bits */
// const int c_osc_fpga_xadc_bits = 12;

// /** @brief Max and min voltage on ADCs.
//  * Symetrical - Max Voltage = +14, Min voltage = -1 * c_osc_fpga_max_v
//  */
// const float c_osc_fpga_adc_max_v  = +14;
// /** Sampling frequency = 125Mspmpls (non-decimated) */
// const float c_osc_fpga_smpl_freq = 125e6;
// /** Sampling period (non-decimated) - 8 [ns] */
// const float c_osc_fpga_smpl_period = (1. / 125e6);

// /**
//  * @brief Internal function used to clean up memory.
//  *
//  * This function un-maps FPGA register and signal buffers, closes memory file
//  * descriptor and cleans all memory allocated by this module.
//  *
//  * @retval 0 Success
//  * @retval -1 Error happened during cleanup.
//  */
// int __osc_fpga_cleanup_mem(void)
// {
//     /* If register structure is NULL we do not need to un-map and clean up */
//     if(g_osc_fpga_reg_mem) {
//         if(munmap(g_osc_fpga_reg_mem, OSC_FPGA_BASE_SIZE) < 0) {
//             fprintf(stderr, "munmap() failed: %s\n", strerror(errno));
//             return -1;
//         }
//         g_osc_fpga_reg_mem = NULL;
//         if(g_osc_fpga_cha_mem)
//             g_osc_fpga_cha_mem = NULL;
//         if(g_osc_fpga_chb_mem)
//             g_osc_fpga_chb_mem = NULL;
//         if(g_osc_fpga_xcha_mem)
//             g_osc_fpga_xcha_mem = NULL;
//         if(g_osc_fpga_xchb_mem)
//             g_osc_fpga_xchb_mem = NULL;
//     }
//     if(g_osc_fpga_mem_fd >= 0) {
//         close(g_osc_fpga_mem_fd);
//         g_osc_fpga_mem_fd = -1;
//     }
//     return 0;
// }

// /**
//  * @brief Maps FPGA memory space and prepares register and buffer variables.
//  *
//  * This function opens memory device (/dev/mem) and maps physical memory address
//  * OSC_FPGA_BASE_ADDR (of length OSC_FPGA_BASE_SIZE) to logical addresses. It
//  * initializes the pointers g_osc_fpga_reg_mem, g_osc_fpga_cha_mem and
//  * g_osc_fpga_chb_mem to point to FPGA OSC.
//  * If function failes FPGA variables must not be used.
//  *
//  * @retval 0  Success
//  * @retval -1 Failure, error is printed to standard error output.
//  */
// int osc_fpga_init(void)
// {
//     /* Page variables used to calculate correct mapping addresses */
//     void *page_ptr;
//     long page_addr, page_off, page_size = sysconf(_SC_PAGESIZE);

//     /* If module was already initialized once, clean all internals. */
//     if(__osc_fpga_cleanup_mem() < 0)
//         return -1;

//     /* Open /dev/mem to access directly system memory */
//     g_osc_fpga_mem_fd = open("/dev/mem", O_RDWR | O_SYNC);
//     if(g_osc_fpga_mem_fd < 0) {
//         fprintf(stderr, "open(/dev/mem) failed: %s\n", strerror(errno));
//         return -1;
//     }

//     /* Calculate correct page address and offset from OSC_FPGA_BASE_ADDR and
//      * OSC_FPGA_BASE_SIZE
//      */
//     page_addr = OSC_FPGA_BASE_ADDR & (~(page_size-1));
//     page_off  = OSC_FPGA_BASE_ADDR - page_addr;

//     /* Map FPGA memory space to page_ptr. */
//     page_ptr = mmap(NULL, OSC_FPGA_BASE_SIZE, PROT_READ | PROT_WRITE,
//                           MAP_SHARED, g_osc_fpga_mem_fd, page_addr);
//     if((void *)page_ptr == MAP_FAILED) {
//         fprintf(stderr, "mmap() failed: %s\n", strerror(errno));
//         __osc_fpga_cleanup_mem();
//         return -1;
//     }

//     /* Set FPGA OSC module pointers to correct values. */
//     g_osc_fpga_reg_mem = page_ptr + page_off;

//     g_osc_fpga_cha_mem = (uint32_t *)g_osc_fpga_reg_mem +
//         (OSC_FPGA_CHA_OFFSET / sizeof(uint32_t));

//     g_osc_fpga_chb_mem = (uint32_t *)g_osc_fpga_reg_mem +
//         (OSC_FPGA_CHB_OFFSET / sizeof(uint32_t));

//     g_osc_fpga_xcha_mem = (uint32_t *)g_osc_fpga_reg_mem +
//         (OSC_FPGA_XCHA_OFFSET / sizeof(uint32_t));

//     g_osc_fpga_xchb_mem = (uint32_t *)g_osc_fpga_reg_mem +
//         (OSC_FPGA_XCHB_OFFSET / sizeof(uint32_t));

//     page_addr = DIGDAR_FPGA_BASE_ADDR & (~(page_size-1));
//     page_off  = DIGDAR_FPGA_BASE_ADDR - page_addr;

//     page_ptr = mmap(NULL, DIGDAR_FPGA_BASE_SIZE, PROT_READ | PROT_WRITE,
//                           MAP_SHARED, g_osc_fpga_mem_fd, page_addr);

//     if((void *)page_ptr == MAP_FAILED) {
//         fprintf(stderr, "mmap() failed: %s\n", strerror(errno));
//         __osc_fpga_cleanup_mem();
//         return -1;
//     }
//     g_digdar_fpga_reg_mem = page_ptr + page_off;

//     return 0;
// }

// /**
//  * @brief Cleans up FPGA OSC module internals.
//  *
//  * This function closes the memory file descriptor, unmap the FPGA memory space
//  * and cleans also all other internal things from FPGA OSC module.
//  * @retval 0 Sucess
//  * @retval -1 Failure
//  */
// int osc_fpga_exit(void)
// {
//   //    if(g_osc_fpga_reg_mem)
//     /* tell FPGA to stop packing slow ADC values into upper 16 bits of CHA, CHB */
//       //      *(int *)(OSC_FPGA_SLOW_ADC_OFFSET + (char *) g_osc_fpga_reg_mem) = 0;
//     return __osc_fpga_cleanup_mem();
// }

// /** @brief OSC FPGA ARM
//  *
//  * ARM internal oscilloscope FPGA state machine to start writting input buffers.

//  * @retval 0 Always returns 0.
//  */
// int osc_fpga_arm_trigger(void)
// {
//   g_osc_fpga_reg_mem->digdar_extra_options = 21;  // 1: only buffer samples *after* being triggered; (no: 2: negate range of sample values); 4: double-width reads; 16: return sum
//   g_osc_fpga_reg_mem->conf |= OSC_FPGA_CONF_ARM_BIT;
//     return 0;
// }

// /** @brief Sets the trigger source in OSC FPGA register.
//  *
//  * Sets the trigger source in oscilloscope FPGA register.
//  *
//  * @param [in] trig_source Trigger source, as defined in FPGA register
//  *                         description.
//  */
// int osc_fpga_set_trigger(uint32_t trig_source)
// {
//     g_osc_fpga_reg_mem->trig_source = trig_source;
//     return 0;
// }

// /** @brief Sets the decimation rate in the OSC FPGA register.
//  *
//  * Sets the decimation rate in the oscilloscope FPGA register.
//  *
//  * @param [in] decim_factor decimation factor, which must be
//  * one of the valid values for the FPGA build:
//  * 1, 2, 8, 64, 1024, 8192, 65536
//  */
// int osc_fpga_set_decim(uint32_t decim_factor)
// {
//     g_osc_fpga_reg_mem->data_dec = decim_factor;
//     return 0;
// }

// /** @brief Sets the trigger delay in OSC FPGA register.
//  *
//  * Sets the trigger delay in oscilloscope FPGA register.
//  *
//  * @param [in] trig_delay Trigger delay, as defined in FPGA register
//  *                         description.
//  *
//  * @retval 0 Always returns 0.
//  */
// int osc_fpga_set_trigger_delay(uint32_t trig_delay)
// {
//     g_osc_fpga_reg_mem->trigger_delay = trig_delay;
//     return 0;
// }

// /** @brief Checks if FPGA detected trigger.
//  *
//  * This function checks if trigger was detected by the FPGA.
//  *
//  * @retval 0 Trigger not detected.
//  * @retval 1 Trigger detected.
//  */
// int osc_fpga_triggered(void)
// {
//     return ((g_osc_fpga_reg_mem->trig_source & OSC_FPGA_TRIG_SRC_MASK)==0);
// }

// /** @brief Returns memory pointers for both input signal buffers.
//  *
//  * This function returns pointers for input signal buffers for all 4 channels.
//  *
//  * @param [out] cha_signal Output pointer for Channel A buffer
//  * @param [out] chb_signal Output pointer for Channel B buffer
//  * @param [out] xcha_signal Output pointer for Slow Channel A buffer
//  * @param [out] xchb_signal Output pointer for Slow Channel B buffer
//  *
//  * @retval 0 Always returns 0.
//  */
// int osc_fpga_get_sig_ptr(int **cha_signal, int **chb_signal, int **xcha_signal, int **xchb_signal)
// {
//     *cha_signal = (int *)g_osc_fpga_cha_mem;
//     *chb_signal = (int *)g_osc_fpga_chb_mem;
//     *xcha_signal = (int *)g_osc_fpga_xcha_mem;
//     *xchb_signal = (int *)g_osc_fpga_xchb_mem;
//     return 0;
// }

// /** @brief Returns values for current and trigger write FPGA pointers.
//  *
//  * This functions returns values for current and trigger write pointers. They
//  * are an address of the input signal buffer and are the same for both channels.
//  *
//  * @param [out] wr_ptr_curr Current FPGA input buffer address.
//  * @param [out] wr_ptr_trig Trigger FPGA input buffer address.
//  *
//  * @retval 0 Always returns 0.
//   */
// int osc_fpga_get_wr_ptr(int *wr_ptr_curr, int *wr_ptr_trig)
// {
//     if(wr_ptr_curr)
//         *wr_ptr_curr = g_osc_fpga_reg_mem->wr_ptr_cur;
//     if(wr_ptr_trig)
//         *wr_ptr_trig = g_osc_fpga_reg_mem->wr_ptr_trigger;
//     return 0;
// }

func Init() (fpga *OgdarFPGA) {
	fpga = new(OgdarFPGA)
	fpga.memfile, err := os.OpenFile("/dev/mem", os.O_RDWR, 0744)
	if err != nil {
		return nil
	}
	mmap, err := syscall.Mmap(int(fpga.memfile.Fd()), DIGDAR_FPGA_BASE_ADDR, DIGDAR_FPGA_BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.ogd = (*OgdarFPGARegMem)(unsafe.Pointer(&mmap[0]))
	mmap, err := syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR, OSC_FPGA_BASE_SIZE, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.osc = (*OscFPGARegMem)(unsafe.Pointer(&mmap[0]))
	mmap, err := syscall.Mmap(int(fpga.memfile.Fd()), OSC_FPGA_BASE_ADDR + OSC_FPGA_CHA_OFFSET, BUFF_SIZE_BYTES, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		goto cleanup
	}
	fpga.VidBuf = (*[SAMPLES_PER_BUFF]uint32)(unsafe.Pointer(&mmap[0]))
	return fpga
cleanup:
	fpga.memfile.Close()
	return nil
}
