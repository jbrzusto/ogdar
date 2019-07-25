// register definitions - generated by gen_verilog.go

   reg  [32-1: 0] command                  ; // Command Register: bit[0]: arm trigger; bit[1]: reset
   reg  [32-1: 0] trig_source              ; // Trigger source: 0: don't trigger; 1: trigger immediately upon arming; 2: radar trigger pulse; 3: ACP pulse; 4: ARP pulse
   reg  [32-1: 0] num_samp                 ; // Number of Samples: number of samples to write after being triggered.  Must be even and in the range 2...16384.
   reg  [32-1: 0] dec_rate                 ; // Decimation Rate: number of input samples to consume for one output sample. 0...65536.  For rates 1, 2, 3 and 4, samples can be summed instead of decimated.  For rates 1, 2, 4, 8, 64, 1024, 8192 and 65536, samples can be averaged instead of decimated bits [31:17] - reserved
   reg  [32-1: 0] options                  ; // Options: digdar-specific options; see type DigdarOption bit[0]: Average samples; bit[1]: Sum samples; bit[2]: Negate video; bit[3]: Counting mode
   reg  [32-1: 0] trig_thresh_excite       ; // Trigger Excite Threshold: Trigger pulse is detected after trigger channel ADC value meets or exceeds this value (in direction away from the Trigger Relax Threshold).  -8192...8191
   reg  [32-1: 0] trig_thresh_relax        ; // Trigger Relax Threshold: After a trigger pulse has been detected, the trigger channel ADC value must meet or exceed this value (in direction away from the Trigger Excite Threshold) before a trigger will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -8192...8191
   reg  [32-1: 0] trig_delay               ; // Trigger Delay: How long to wait after trigger is detected before starting to capture samples from the video channel.  The delay is in units of ADC clocks; i.e. the value is multiplied by 8 nanoseconds.
   reg  [32-1: 0] trig_latency             ; // Trigger Latency: how long to wait after trigger relaxation before allowing next excitation.  To further debounce the trigger signal, we can specify a minimum wait time between relaxation and excitation.  0...65535 (which gets multiplied by 8 nanoseconds)
   reg  [32-1: 0] acp_thresh_excite        ; // ACP Excite Threshold: the AC Pulse is detected when the ACP channel value meets or exceeds this value (in direction away from the ACP Relax Threshold).  -2048...2047
   reg  [32-1: 0] acp_thresh_relax         ; // ACP Relax Threshold: After an ACP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from acp_thresh_excite) before an ACP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048...2047
   reg  [32-1: 0] acp_latency              ; // ACP Latency: how long to wait after ACP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)
   reg  [32-1: 0] arp_thresh_excite        ; // ARP Excite Threshold: the AR Pulse is detected when the ARP channel value meets or exceeds this value (in direction away from the ARP Relax Threshold).  -2048..2047
   reg  [32-1: 0] arp_thresh_relax         ; // ARP Relax Threshold: After an ARP has been detected, the acp channel ADC value must meet or exceed this value (in direction away from arp_thresh_excite) before an ARP will be detected again.  (Serves to debounce signal in Schmitt trigger style).  -2048..2047
   reg  [32-1: 0] arp_thresh_latency       ; // ARP Latency: how long to wait after ARP relaxation before allowing next excitation.  To further debounce the acp signal, we can specify a minimum wait time between relaxation and excitation.  0...1000000 (which gets multiplied by 8 nanoseconds)
   reg  [64-1: 0] trig_clock               ; // Trigger Clock: ADC clock count at last trigger pulse
   reg  [64-1: 0] trig_prev_clock          ; // Previous Trigger Clock: ADC clock count at previous trigger pulse
   reg  [64-1: 0] acp_clock                ; // ACP Clock: ADC clock count at last ACP
   reg  [64-1: 0] acp_prev_clock           ; // Previous ACP Clock: ADC clock count at previous ACP
   reg  [64-1: 0] arp_clock                ; // ARP Clock: ADC clock count at last ARP
   reg  [64-1: 0] arp_prev_clock           ; // Previous ARP Clock: ADC clock count at previous ARP
   wire [32-1: 0] trig_count               ; // Trigger Count: number of trigger pulses detected since last reset
   wire [32-1: 0] acp_count                ; // ACP Count: number of Azimuth Count Pulses detected since last reset
   wire [32-1: 0] arp_count                ; // ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset
   reg  [32-1: 0] acp_per_arp              ; // count of ACP between two most recent ARP
   reg  [32-1: 0] adc_counter              ; // ADC Counter: 14-bit ADC counter used in counting mode
   reg  [32-1: 0] acp_at_arp               ; // ACP at ARP: ACP count at most recent ARP
   reg  [32-1: 0] clock_since_acp_at_arp   ; // ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP
   reg  [32-1: 0] trig_at_arp              ; // Trig at ARP: Trigger count at most recent ARP
   reg  [64-1: 0] clocks                   ; // clocks: 64-bit count of ADC clock ticks since reset
   reg  [32-1: 0] acp_raw                  ; // most recent slow ADC value from ACP
   reg  [32-1: 0] arp_raw                  ; // most recent slow ADC value from ARP
   reg  [64-1: 0] saved_trig_clock         ; // Trigger Clock: ADC clock count at last trigger pulse
   reg  [64-1: 0] saved_trig_prev_clock    ; // Previous Trigger Clock: ADC clock count at previous trigger pulse
   reg  [64-1: 0] saved_acp_clock          ; // ACP Clock: ADC clock count at last ACP
   reg  [64-1: 0] saved_acp_prev_clock     ; // Previous ACP Clock: ADC clock count at previous ACP
   reg  [64-1: 0] saved_arp_clock          ; // ARP Clock: ADC clock count at last ARP
   reg  [64-1: 0] saved_arp_prev_clock     ; // Previous ARP Clock: ADC clock count at previous ARP
   reg  [32-1: 0] saved_trig_count         ; // Trigger Count: number of trigger pulses detected since last reset
   reg  [32-1: 0] saved_acp_count          ; // ACP Count: number of Azimuth Count Pulses detected since last reset
   reg  [32-1: 0] saved_arp_count          ; // ARP Count: number of Azimuth Return Pulses (rotations) detected since last reset
   reg  [32-1: 0] saved_acp_per_arp        ; // count of ACP between two most recent ARP
   reg  [32-1: 0] saved_adc_counter        ; // ADC Counter: 14-bit ADC counter used in counting mode
   reg  [32-1: 0] saved_acp_at_arp         ; // ACP at ARP: ACP count at most recent ARP
   reg  [32-1: 0] saved_clock_since_acp_at_arp; // ACP Offset at ARP: count of ADC clocks since last ACP, at last ARP
   reg  [32-1: 0] saved_trig_at_arp        ; // Trig at ARP: Trigger count at most recent ARP
