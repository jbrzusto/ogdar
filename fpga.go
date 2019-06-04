package main

import ()

const (
	OSC_FPGA_BASE_ADDR      = 0x40100000  // Starting address of FPGA registers handling Oscilloscope module.
	OSC_FPGA_BASE_SIZE      = 0x50000     // The size of FPGA registers handling Oscilloscope module.
	OSC_FPGA_SIG_LEN        = (16 * 1024) // Size of data buffer into which input signal is captured , must be 2^n!.
	OSC_FPGA_CONF_ARM_BIT   = 1           // Bit index in FPGA configuration register for arming the trigger.
	OSC_FPGA_CONF_RST_BIT   = 2           // Bit index in FPGA configuration register for reseting write state machine.
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

	// The offsets to the metadata (read-only-by client) in the FPGA register block
	//    Must match values in red_pitaya_digdar.v from FPGA project

	OFFSET_SAVED_TRIG_COUNT           = 0x00068 // (saved) TRIG count since reset (32 bits; wraps)
	OFFSET_SAVED_TRIG_CLOCK_LOW       = 0x0006C // (saved) clock at most recent TRIG (low 32 bits)
	OFFSET_SAVED_TRIG_CLOCK_HIGH      = 0x00070 // (saved) clock at most recent TRIG (high 32 bits)
	OFFSET_SAVED_TRIG_PREV_CLOCK_LOW  = 0x00074 // (saved) clock at previous TRIG (low 32 bits)
	OFFSET_SAVED_TRIG_PREV_CLOCK_HIGH = 0x00078 // (saved) clock at previous TRIG (high 32 bits)
	OFFSET_SAVED_ACP_COUNT            = 0x0007C // (saved) ACP count since reset (32 bits; wraps)
	OFFSET_SAVED_ACP_CLOCK_LOW        = 0x00080 // (saved) clock at most recent ACP (low 32 bits)
	OFFSET_SAVED_ACP_CLOCK_HIGH       = 0x00084 // (saved) clock at most recent ACP (high 32 bits)
	OFFSET_SAVED_ACP_PREV_CLOCK_LOW   = 0x00088 // (saved) clock at previous ACP (low 32 bits)
	OFFSET_SAVED_ACP_PREV_CLOCK_HIGH  = 0x0008C // (saved) clock at previous ACP (high 32 bits)
	OFFSET_SAVED_ARP_COUNT            = 0x00090 // (saved) ARP count since reset (32 bits; wraps)
	OFFSET_SAVED_ARP_CLOCK_LOW        = 0x00094 // (saved) clock at most recent ARP (low 32 bits)
	OFFSET_SAVED_ARP_CLOCK_HIGH       = 0x00098 // (saved) clock at most recent ARP (high 32 bits)
	OFFSET_SAVED_ARP_PREV_CLOCK_LOW   = 0x0009C // (saved) clock at previous ARP (low 32 bits)
	OFFSET_SAVED_ARP_PREV_CLOCK_HIGH  = 0x000A0 // (saved) clock at previous ARP (high 32 bits)
	OFFSET_SAVED_ACP_PER_ARP          = 0x000A4 // (saved) count of ACP pulses between two most recent ARP pulses
	OFFSET_ACP_AT_ARP                 = 0x000B8 // most recent ACP count at ARP pulse
	OFFSET_SAVED_ACP_AT_ARP           = 0x000BC // (saved) most recent ACP count at ARP pulse
	OSC_HYSTERESIS                    = 0x3F    // Hysteresis register default setting

)

type OscFPGARegMem struct { // FPGA registry structure for Oscilloscope core module.
	//
	// This structure is direct image of physical FPGA memory. It assures
	// direct read/write FPGA access when it is mapped to the appropriate memory address
	// through /dev/mem device.

	conf uint32 //  Configuration:
	// bit     [0] - arm_trigger
	// bit     [1] - rst_wr_state_machine
	// bits [31:2] - reserved

	trig_source uint32 //  Trigger source:
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

	cha_thr uint32 //  ChA threshold:
	// bits [13: 0] - ChA threshold
	// bits [31:14] - reserved

	chb_thr uint32 //  ChB threshold:
	// bits [13: 0] - ChB threshold
	// bits [31:14] - reserved

	trigger_delay uint32 //  After trigger delay:
	// bits [31: 0] - trigger delay
	// 32 bit number - how many decimated samples should be stored into a buffer.
	// (max 16k samples)

	data_dec uint32 //  Data decimation
	// bits [16: 0] - decimation factor, legal values:
	//   1, 2, 8, 64, 1024, 8192 65536
	//   If other values are written data is undefined
	// bits [31:17] - reserved

	wr_ptr_cur uint32 //  Write pointers - both of the format:
	// bits [13: 0] - pointer
	// bits [31:14] - reserved
	// Current pointer - where machine stopped writing after trigger
	// Trigger pointer - where trigger was detected

	wr_ptr_trigger uint32

	cha_hystersis uint32 //  ChA & ChB hysteresis - both of the format:
	// bits [13: 0] - hysteresis threshold
	// bits [31:14] - reserved

	chb_hystersis uint32

	other uint32 // @brief
	// bits [0] - enable signal average at decimation
	// bits [31:1] - reserved

	reseved uint32

	cha_filt_aa uint32 // ChA Equalization filter
	// bits [17:0] - AA coefficient (pole)
	// bits [31:18] - reserved

	cha_filt_bb uint32 // ChA Equalization filter
	// bits [24:0] - BB coefficient (zero)
	// bits [31:25] - reserved

	cha_filt_kk uint32 // ChA Equalization filter
	// bits [24:0] - KK coefficient (gain)
	// bits [31:25] - reserved

	cha_filt_pp uint32 // ChA Equalization filter
	// bits [24:0] - PP coefficient (pole)
	// bits [31:25] - reserved

	chb_filt_aa uint32 // ChB Equalization filter
	// bits [17:0] - AA coefficient (pole)
	// bits [31:18] - reserved

	chb_filt_bb uint32 // ChB Equalization filter
	// bits [24:0] - BB coefficient (zero)
	// bits [31:25] - reserved

	chb_filt_kk uint32 // ChB Equalization filter
	// bits [24:0] - KK coefficient (gain)
	// bits [31:25] - reserved

	chb_filt_pp uint32 // ChB Equalization filter
	// bits [24:0] - PP coefficient (pole)
	// bits [31:25] - reserved

	digdar_extra_options uint32 // Extra options:
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

//  ChA & ChB data - 14 LSB bits valid starts from 0x10000 and
// 0x20000 and are each 16k samples long

type OgdarFPGARegMem struct {

	// --------------- TRIG -----------------

	trig_thresh_excite uint32 //  trig_thresh_excite: trigger excitation threshold
	//          Trigger is raised for one FPGA clock after trigger channel
	//          ADC value meets or exceeds this value (in direction away
	//          from trig_thresh_relax).
	// bits [13: 0] - threshold, signed
	// bit  [31:14] - reserved

	trig_thresh_relax uint32 //  trig_thresh_relax: trigger relaxation threshold
	//          After a trigger has been raised, the trigger channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from trig_thresh_excite) before a trigger will be raised again.
	//          (Serves to debounce signal in schmitt-trigger style).
	// bits [13: 0] - threshold, signed
	// bit  [31:14] - reserved

	trig_delay uint32 //  trig_delay: (traditional) trigger delay.
	//          How long to wait after trigger is raised
	//          before starting to capture samples from Video channel.
	//          Note: this usage of 'delay' is traditional for radar digitizing
	//          but differs from the red pitaya scope usage, which means
	//          "number of decimated ADC samples to acquire after trigger is raised"
	// bits [31: 0] - unsigned wait time, in ADC clocks.

	trig_latency uint32 //  trig_latency: how long to wait after trigger relaxation before
	//          allowing next excitation.
	//          To further debounce the trigger signal, we can specify a minimum
	//          wait time between relaxation and excitation.
	// bits [31: 0] - unsigned latency time, in ADC clocks.

	trig_count uint32 //  trig_count: number of trigger pulses detected since last reset
	// bits [31: 0] - unsigned count of trigger pulses detected

	trig_clock_low uint32 //  trig_clock_low: ADC clock count at last trigger pulse
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count

	trig_clock_high uint32 //  trig_clock_high: ADC clock count at last trigger pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	trig_prev_clock_low uint32 //  trig_prev_clock_low: ADC clock count at previous trigger pulse,
	//          so we can calculate trigger rate, regardless of capture rate
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count

	trig_prev_clock_high uint32 //  trig_prev_clock_high: ADC clock count at previous trigger pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	// --------------- ACP -----------------

	//  acp_thresh_excite: acp excitation threshold
	//          the acp pulse is detected and counted when the ACP slow ADC
	//          channel meets or exceeds this value in the direction away
	//          from acp_thresh_relax
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

	acp_thresh_excite uint32

	acp_thresh_relax uint32 //  acp_thresh_relax: acp relaxation threshold
	//          After an acp has been detected, the acp channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from acp_thresh_excite) before a acp will be detected again.
	//          (Serves to debounce signal in schmitt-acp style).
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

	acp_latency uint32 //  acp_latency: how long to wait after acp relaxation before
	//          allowing next excitation.
	//          To further debounce the acp signal, we can specify a minimum
	//          wait time between relaxation and excitation.
	// bits [31: 0] - unsigned latency time, in ADC clocks.

	acp_count uint32 //  acp_count: number of acp pulses detected since last reset
	// bits [31: 0] - unsigned count of acp pulses detected

	acp_clock_low uint32 //  acp_clock_low: ADC clock count at last acp pulse
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count

	acp_clock_high uint32 //  acp_clock_high: ADC clock count at last acp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	acp_prev_clock_low uint32 //  acp_prev_clock_low: ADC clock count at previous acp pulse,
	//          so we can calculate acp rate, regardless of capture rate
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count

	acp_prev_clock_high uint32 //  acp_prev_clock_high: ADC clock count at previous acp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	// --------------- ARP -----------------

	//  arp_thresh_excite: arp excitation threshold
	//          the arp pulse is detected and counted when the ARP slow ADC
	//          channel meets or exceeds this value in the direction away
	//          from arp_thresh_relax
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

	arp_thresh_excite uint32

	arp_thresh_relax uint32 //  arp_thresh_relax: arp relaxation threshold
	//          After an arp has been detected, the arp channel ADC value
	//          must meet or exceeds this value (in direction away
	//          from arp_thresh_excite) before a arp will be detected again.
	//          (Serves to debounce signal in schmitt-arp style).
	// bits [11: 0] - threshold, signed
	// bit  [31:14] - reserved

	arp_latency uint32 //  arp_latency: how long to wait after arp relaxation before
	//          allowing next excitation.
	//          To further debounce the arp signal, we can specify a minimum
	//          wait time between relaxation and excitation.
	// bits [31: 0] - unsigned latency time, in ADC clocks.

	arp_count uint32 //  arp_count: number of arp pulses detected since last reset
	// bits [31: 0] - unsigned count of arp pulses detected

	arp_clock_low uint32 //  arp_clock_low: ADC clock count at last arp pulse
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count

	arp_clock_high uint32 //  arp_clock_high: ADC clock count at last arp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	arp_prev_clock_low uint32 //  arp_prev_clock_low: ADC clock count at previous arp pulse,
	//          so we can calculate arp rate, regardless of capture rate
	// bits [31: 0] - unsigned (low 32 bits) of ADC clock count

	arp_prev_clock_high uint32 //  arp_prev_clock_high: ADC clock count at previous arp pulse
	// bits [31: 0] - unsigned (high 32 bits) of ADC clock count

	acp_per_arp uint32 //  acp_per_arp: count of ACP pulses between two most recent ARP pulses
	// bits [31: 0] - unsigned count of ACP pulses

	// --------------------- SAVED COPIES ----------------------------------------
	// For these metadata, we want to record the values at the time of the
	// most recently *captured* pulse.  So if the capture thread is not keeping up
	// with the radar, we still have correct values of these metadata for each
	// captured pulse (e.g. the value of the ACP count at each captured radar pulse).
	// The FPGA knows at trigger detection time whether or not
	// the pulse will be captured, and if so, copies the live metadata values to
	// these saved locations.

	saved_trig_count           uint32 //  saved_trig_count:  value at start of most recently captured pulse
	saved_trig_clock_low       uint32 //  saved_trig_clock_low:  value at start of most recently captured pulse
	saved_trig_clock_high      uint32 //  saved_trig_clock_high:  value at start of most recently captured pulse
	saved_trig_prev_clock_low  uint32 //  saved_trig_prev_clock_low:  value at start of most recently captured pulse
	saved_trig_prev_clock_high uint32 //  saved_trig_prev_clock_high:  value at start of most recently captured pulse
	saved_acp_count            uint32 //  saved_acp_count:  value at start of most recently captured pulse
	saved_acp_clock_low        uint32 //  saved_acp_clock_low:  value at start of most recently captured pulse
	saved_acp_clock_high       uint32 //  saved_acp_clock_high:  value at start of most recently captured pulse
	saved_acp_prev_clock_low   uint32 //  saved_acp_prev_clock_low:  value at start of most recently captured pulse
	saved_acp_prev_clock_high  uint32 //  saved_acp_prev_clock_high:  value at start of most recently captured pulse
	saved_arp_count            uint32 //  saved_arp_count:  value at start of most recently captured pulse
	saved_arp_clock_low        uint32 //  saved_arp_clock_low:  value at start of most recently captured pulse
	saved_arp_clock_high       uint32 //  saved_arp_clock_high:  value at start of most recently captured pulse
	saved_arp_prev_clock_low   uint32 //  saved_arp_prev_clock_low:  value at start of most recently captured pulse
	saved_arp_prev_clock_high  uint32 //  saved_arp_prev_clock_high:  value at start of most recently captured pulse
	saved_acp_per_arp          uint32 //  saved_acp_per_arp:  value at start of most recently captured pulse
	uint64_t                   clocks //  clocks: 64-bit count of ADC clock ticks since reset
	//  most recent slow ADC value from ACP
	acp_raw uint32
	//  most recent slow ADC value from ARP
	arp_raw           uint32
	acp_at_arp        uint32 //  acp_at_arp:  value of acp count at most recent arp pulse
	saved_acp_at_arp  uint32 //  saved_acp_at_arp:  value at start of most recently captured pulse
	trig_at_arp       uint32 //  trig_at_arp:  value of trig count at most recent arp pulse
	saved_trig_at_arp uint32 //  saved_trig_at_arp:  value at start of most recently captured pulse
}
