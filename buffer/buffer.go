/*
 Buff manages transfers radar data from the FPGA into RAM.

 Acquire data from the FPGA into a ring buffer, notifying clients
 of new scanlines or sweeps.  Handle setting of digitizer parameters.
*/
package buffer

import (
	"time"
	"errors"
	//	"os"
	//	"syscall"
	//	"unsafe"
)

// Sample represents the echo strength for a short period of time.  On
// the redpitaya, the fast ADCs return signed 14 bit samples @ 125MSPS
// representing a timestep of 8 ns.  The FPGA converts these to
// unsigned 14 bit integers in the bottom of a 16 bit register,
// or occupying up to 16 bits in the case of summed samples at decimation
// rates of 2, 3, or 4.  However, 0 is never returned by the FPGA; instead,
// 0 is incremented to 1.  This reserves 0 as a sentinel value.
type Sample uint16

const (
	NOT_A_SAMPLE Sample = 0x0000 // value used to flag uninitialized samples or metadata in buffer
)

// DecimRateM1 is the number of ADC clocks per sample, minus one.
// i.e. 0 means 1 clock per sample (125 MSPS); 1 means 2 clocks per sample (62.5 MSPS), etc
type DecimRateM1 uint16

// DecimMode is how multiple samples are combined (if at all) when decimating
type DecimMode uint16
const (
	DECIM_DECIM DecimMode = iota // the last of every n samples is used; n >= 1
	DECIM_SUM // the sum of every n consecutive samples is used; n <= 4
	DECIM_AVG // the average of every n consecutive samples is used; n = 2^m for m = 1, 2, 3, 6, 10, 13, 15
)

// Scanline is a sequence of samples received after one radar pulse
// is emitted, and represents received echo strength versus range (or
// equivalently, time).  It is bundled with metadata from which the
// absolute time and azimuth can be derived:
//
// - ADC clock ticks since some known event
//
// - ACPs (azimuth count pulses) occur at a known number of evenly
// spaced azimuths, but the physical azimuth of the first azimuth
// pulse after a radar restart is unknown.
//
// - ARPs (azimuth return pulses) occur once per rotation at
// approximately the same physical azimuth across radar restarts; due
// to variance in its detection, we treat this statistically.
type Scanline struct {
	ARPCount  uint32 // number of ARP pulses since reset; could wrap, but will take 170 years even at 48 RPM
	TrigClock uint32 // ADC clock ticks since last ACP wraparound
	TrigCount uint32 // low 32-bits of count of trigger pulses since reset, including those not captured
	ACPClock  uint32 // bits 31:20 - ACPs since last ACP wraparound; bits 19:0 - ADC clock ticks since last ACP
	DecimRateM1
	Extra uint16    // bits 15:14: DecimMode; bits 13:0 skipped clocks before first sample (i.e. additional trigger delay)
	Samples []Sample // slice from sample buffer corresponding to
	// samples in this scanline.  There are 2 extra samples stored
	// in the samplebuffer at the start of each scanline's
	// samples: a NOT_A_SAMPLE to mark the start of scanline, and
	// then a uint16 scanline serial number which is the low-order
	// 16 bits of TrigCount.  If the first two slots in this slice
	// are not {NOT_A_SAMPLE, TrigCount & 0xFFFF}, then we know the scanline's
	// storage has been overwritten.
}

// Sweep is a sequence of scanlines from a full rotation of the radar antenna.
type Sweep struct {
	ARP    uint32    // ARP count since reset at first scanline
	ts0    time.Time // time of first scanline in sweep
	tw1    time.Time // time of last scanline in sweep
	clock  uint32    // base rate of sampling clock, in Hz
	uniform bool     // does every scanline in this sweep have the same decimation rate and first sample range?
	n      uint16     // number of scanlines in this sweep (increases as sweep is accumulated)
	s1 ScanlineHandle // handle for first scanline in this sweep
	s2 ScanlineHandle // handle for last scanline in this sweep (changes as sweep is accumulated)
	Lines  []Scanline // scanlines for this sweep
	Lines2 []Scanline // 2nd contiguous segment of scanlines; empty unless sweep wraps over end of Scanline buffer
}

// Amounts of RAM for sample and scanline buffers.
// A typical sweep is ~5000 scanlines, with a max of say 4K samples,
// for a total of 20 M samples = 40 MB.  We try for a buffer
// of roughly 5 of these, so 200 MB for sample memory, and
// 5 * 5000 = 25 K scanlines ~0.8 MB (@ 32 bytes per scanline)
const (
	SWEEP_BUFF_SIZE      = 5 // number of sweeps in sweep buffer
	MAX_PRF              = 2200
	MIN_RPM              = 22
	MAX_SWEEP_SCANLINES  = MAX_PRF * (60 / MIN_RPM)
	MAX_SCANLINE_SAMPLES = 4000
	SAMPLE_BUFF_SIZE     = SWEEP_BUFF_SIZE * MAX_SWEEP_SCANLINES * MAX_SCANLINE_SAMPLES // number of samples in ring buffer
	SCANLINE_BUFF_SIZE   = SWEEP_BUFF_SIZE * MAX_SWEEP_SCANLINES                        // number of scanlines in ring buffer
)

// SampleBuff stores samples in a ring buffer.  Samples from each
// scanline are stored contiguously, so there will be empty space at
// the end of the sample buffer if the number of samples in a scanline
// doesn't divide into SAMPLE_BUFF_SIZE evenly.
type SampleBuff struct {
	SampBuff [SAMPLE_BUFF_SIZE]Sample // ring buffer of samples
	iSample  int                      // location for next sample to be written
	nSamples uint64                   // total samples captured during this run
}

// NextSliceFor returns the next slice in the SampleBuff large enough to hold n samples,
// or nil if there is none.
func (sb *SampleBuff) NextSliceFor(n int) (s []Sample) {
	if n <= 0 {
		return
	}
	if n <= len(sb.SampBuff) {
		if sb.iSample+n > len(sb.SampBuff) {
			sb.iSample = 0
		}
		s = sb.SampBuff[sb.iSample : sb.iSample+n]
		sb.iSample += n
		sb.nSamples += uint64(n) // assumes slice will be filled
	}
	return
}

// ScanlineBuff is a ring buffer of scanlines. Their samples are
// stored in the sample buffer.  A sweep might wrap around the end of
// the scanline buffer.
type ScanlineBuff struct {
	*SampleBuff                              // location of sample ring buffer
	ScanBuff    [SCANLINE_BUFF_SIZE]Scanline // ring buffer of Scanline structs
	iScanline   int                          // location for next scanline to be written
	nScanlines  uint64                       // total scanlines captured during this run
}

// ScanlineHandle represents a captured scanline which might or might
// not still exist in the buffer.
// Bits [31:16] index in the scanline buffer
// Bits [15:0] low 16 bits of TrigCount value of Scanline
// We can quickly check whether a ScanlineHandle represents a Scanline
// which is still in the buffer by testing whether the Scanline at the
// purported index has the matching low 16 bits of TrigCount.  This
// will wrap every 31 seconds or so at PRF 2100.
type ScanlineHandle uint32

// Next returns the next Scanline for holding a scanline with n
// samples, or nil if there is none.  trig is the trigger count of the
// scanline.  This is used to create a fingerprint for this scanline
// in the sample buffer, so we can tell when it has been overwritten.
func (slb *ScanlineBuff) Next(n int, trig uint64) (i int, err error) {
	// add two for the {NOT_A_SAMPLE, ID} fingerprint
	var samps []Sample
	if samps = slb.NextSliceFor(n + 2); samps == nil {
		err = errors.New("not enough storage for scanline")
		return
	}
	if slb.iScanline >= len(slb.ScanBuff) {
		slb.iScanline = 0
	}
	i = slb.iScanline
	slb.iScanline++
	slb.ScanBuff[i].Samples = samps
	samps[0] = NOT_A_SAMPLE
	samps[1] = Sample(trig)
	return
}

func (s Scanline) Valid() (bool) {
	return s.Samples[0] == NOT_A_SAMPLE && s.Samples[1] == Sample(s.TrigCount)
}

// SweepHandle is an opaque type representing a specific sweep
// Bits [31:28] index in sweep buffer
// Bits [27:0]  ARP for that sweep
// We can quickly check whether a SweepHandle represents
// a Sweep that is still in the SweepBuffer by testing whether the sweep at the
// purported index has the correct lower 28 bits of ARP count.
// This would take 97 days to wrap around even at 60 RPM.
type SweepHandle uint32

// SweepBuffer is a ring buffer of sweeps
type SweepBuffer struct {
	Sweeps []Sweep // slice of sweeps
	i int // index of next slot to fill in Sweeps
	n int // number of (valid) sweeps in Sweeps
}

// Clients can request to be informed of:
//
//  - each new sweep; this is a sequence of scanlines from a specific
//    azimuth (the "cut") around the circle and back.
//
//  - the first new scanline which is at least a specified amount of
//    time more recent than the previously received scanline.  This
//    permits clients to receive an ongoing sequence of digitized
//    scanlines at a rate they can handle; e.g. 1 per 200 milliseconds
//
// A sweep is communicated as two slices from the scanline buffer.
// Usually, the second slice will be empty, because the sweep is
// contained in a contiguous portion of the scanline buffer.  However,
// a sweep might wrap around the end of the scanline buffer, and in
// this case, the first (earlier) slice of the sweep is at the end of
// the scanline buffer and the second (later) slice is at the
// beginning.
//
// A pulse is communicated as a slice of length 1 from the scanline buffer.
//
// Clients can also request parameter settings by writing them to a channel.
// The buffer object will ensure that parameter settings only happen *between*
// pulse or sweep acquisitions, according to the parameter.
