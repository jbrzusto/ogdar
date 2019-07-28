// memory map definitions - generated by gen_verilog.go

`define OFFSET_Command              20'h000000 // Command Register: bit[0]: arm trigger; bit[1]: reset
`define OFFSET_TrigSource           20'h000004 // Trigger source: 0: don't trigger; 1: trigger immediately upon arming; 2: radar trigger pulse; 3: ACP pulse; 4: ARP pulse
`define OFFSET_NumSamp              20'h000008 // Number of Samples: number of samples to write after being triggered.  Must be even and in the range 2...16384.
`define OFFSET_DecRate              20'h00000c // Decimation Rate: number of input samples to consume for one output sample. 0...65536.  For rates 1, 2, 3 and 4, samples can be summed instead of decimated.  For rates 1, 2, 4, 8, 64, 1024, 8192 and 65536, samples can be averaged instead of decimated bits [31:17] - reserved
`define OFFSET_Options              20'h000010 // Options: digdar-specific options; see type DigdarOption bit[0]: Average samples; bit[1]: Sum samples; bit[2]: Negate video; bit[3]: Counting mode
`define OFFSET_TrigThreshExcite     20'h000014 // Trigger Excite Threshold: Trigger pulse is detected after trigger channel ADC value meets or exceeds this value (in direction away from the Trigger Relax Threshold).  -8192...8191
`define OFFSET_TrigThreshRelax      20'h000018 // Trigger Relax Threshold: After a trigger pulse has been detected, the trigger channel ADC value must meet or exceed this value (in direction away from the Trigger Excite Threshold) before a trigger will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -8192...8191
`define OFFSET_TrigDelay            20'h00001c // Trigger Delay: How long to wait after trigger is detected before starting to capture samples from the video channel.  The delay is in units of ADC clocks; i.e. the value is multiplied by 8 nanoseconds.
`define OFFSET_TrigLatency          20'h000020 // Trigger Latency: how long to wait after trigger relaxation before allowing next excitation.  To further debounce the trigger signal, we can specify a minimum wait time between relaxation and excitation.  0...65535 (which gets multiplied by 8 nanoseconds)
`define OFFSET_TrigCount            20'h000024 // Trigger Count: number of trigger pulses detected since last reset
`define OFFSET_ACPThreshExcite      20'h000028 // ACP Excite Threshold: the AC Pulse is detected when the ACP channel value meets or exceeds this value (in direction away from the ACP Relax Threshold).  -2048...2047
`define OFFSET_ACPThreshRelax       20'h00002c // ACP Relax Threshold: After an ACP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from acp_thresh_excite) before an ACP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048...2047
`define OFFSET_ACPLatency           20'h000030 // ACP Latency: how long to wait after ACP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)
`define OFFSET_ARPThreshExcite      20'h000034 // ARP Excite Threshold: the AR Pulse is detected when the ARP channel value meets or exceeds this value (in direction away from the ARP Relax Threshold).  -2048..2047
`define OFFSET_ARPThreshRelax       20'h000038 // ARP Relax Threshold: After an ARP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from arp_thresh_excite) before an ARP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048..2047
`define OFFSET_ARPLatency           20'h00003c // ARP Latency: how long to wait after ARP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)
`define OFFSET_TrigClock_LO         20'h000040 // low 32-bits: Trigger Clock: ADC clock count at last trigger pulse
`define OFFSET_TrigClock_HI         20'h000044 // high 32-bits
`define OFFSET_TrigPrevClock_LO     20'h000048 // low 32-bits: Previous Trigger Clock: ADC clock count at previous trigger pulse
`define OFFSET_TrigPrevClock_HI     20'h00004c // high 32-bits
`define OFFSET_ACPClock_LO          20'h000050 // low 32-bits: ACP Clock: ADC clock count at last ACP
`define OFFSET_ACPClock_HI          20'h000054 // high 32-bits
`define OFFSET_ACPPrevClock_LO      20'h000058 // low 32-bits: Previous ACP Clock: ADC clock count at previous ACP
`define OFFSET_ACPPrevClock_HI      20'h00005c // high 32-bits
`define OFFSET_ARPClock_LO          20'h000060 // low 32-bits: ARP Clock: ADC clock count at last ARP
`define OFFSET_ARPClock_HI          20'h000064 // high 32-bits
`define OFFSET_ARPPrevClock_LO      20'h000068 // low 32-bits: Previous ARP Clock: ADC clock count at previous ARP
`define OFFSET_ARPPrevClock_HI      20'h00006c // high 32-bits
`define OFFSET_ACPCount             20'h000070 // ACP Count: number of Azimuth Count Pulses detected since last reset
`define OFFSET_ARPCount             20'h000074 // ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset
`define OFFSET_ACPPerARP            20'h000078 // count of ACP between two most recent ARP
`define OFFSET_ADCCounter           20'h00007c // ADC Counter: 14-bit ADC counter used in counting mode
`define OFFSET_ACPAtARP             20'h000080 // ACP at ARP: ACP count at most recent ARP
`define OFFSET_ClockSinceACPAtARP   20'h000084 // ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP
`define OFFSET_TrigAtARP            20'h000088 // Trig at ARP: Trigger count at most recent ARP
`define OFFSET_Clocks_LO            20'h00008c // low 32-bits: clocks: 64-bit count of ADC clock ticks since reset
`define OFFSET_Clocks_HI            20'h000090 // high 32-bits
`define OFFSET_ACPRaw               20'h000094 // most recent slow ADC value from ACP
`define OFFSET_ARPRaw               20'h000098 // most recent slow ADC value from ARP
`define OFFSET_SavedTrigClock_LO    20'h00009c // low 32-bits: Trigger Clock: ADC clock count at last trigger pulse
`define OFFSET_SavedTrigClock_HI    20'h0000a0 // high 32-bits
`define OFFSET_SavedTrigPrevClock_LO 20'h0000a4 // low 32-bits: Previous Trigger Clock: ADC clock count at previous trigger pulse
`define OFFSET_SavedTrigPrevClock_HI 20'h0000a8 // high 32-bits
`define OFFSET_SavedACPClock_LO     20'h0000ac // low 32-bits: ACP Clock: ADC clock count at last ACP
`define OFFSET_SavedACPClock_HI     20'h0000b0 // high 32-bits
`define OFFSET_SavedACPPrevClock_LO 20'h0000b4 // low 32-bits: Previous ACP Clock: ADC clock count at previous ACP
`define OFFSET_SavedACPPrevClock_HI 20'h0000b8 // high 32-bits
`define OFFSET_SavedARPClock_LO     20'h0000bc // low 32-bits: ARP Clock: ADC clock count at last ARP
`define OFFSET_SavedARPClock_HI     20'h0000c0 // high 32-bits
`define OFFSET_SavedARPPrevClock_LO 20'h0000c4 // low 32-bits: Previous ARP Clock: ADC clock count at previous ARP
`define OFFSET_SavedARPPrevClock_HI 20'h0000c8 // high 32-bits
`define OFFSET_SavedTrigCount       20'h0000cc // Trigger Count: number of trigger pulses detected since last reset
`define OFFSET_SavedACPCount        20'h0000d0 // ACP Count: number of Azimuth Count Pulses detected since last reset
`define OFFSET_SavedARPCount        20'h0000d4 // ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset
`define OFFSET_SavedACPPerARP       20'h0000d8 // count of ACP between two most recent ARP
`define OFFSET_SavedADCCounter      20'h0000dc // ADC Counter: 14-bit ADC counter used in counting mode
`define OFFSET_SavedACPAtARP        20'h0000e0 // ACP at ARP: ACP count at most recent ARP
`define OFFSET_SavedClockSinceACPAtARP 20'h0000e4 // ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP
`define OFFSET_SavedTrigAtARP       20'h0000e8 // Trig at ARP: Trigger count at most recent ARP
