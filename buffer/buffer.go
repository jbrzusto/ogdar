/*
 Buffer manages transfers radar data from the FPGA into RAM.

 Acquire data from the FPGA into a ring buffer, notifying clients
 of new scanlines or sweeps.  Handle setting of digitizer parameters.
*/
package buffer

import (
//	"os"
//	"syscall"
//	"unsafe"
)

// Sample represents the echo strength for a short period of time.
// On the redpitaya, the fast ADCs return signed 14 bit samples @
// 125MSPS representing a timestep of 8 ns.  The FPGA converts these
// to unsigned 14 bit integers in the bottom of a 16 bit register.
type Sample uint16

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
	TrigCount uint64 // count of trigger pulses since reset, including those not captured
	TrigClock uint32 // ADC clock ticks since last ACP wraparound
	ACPClock  uint32 // bits 31:20 - ACPs since last ACP wraparound; bits 19:0 - clock ticks since last ACP
	// and M is the fraction of 8 ms represented by the time since the latest ACP
	// i.e. M = elapsed ADC clock ticks (@125MHz) / 1E6
	Samples []Sample // slice from sample buffer corresponding to samples in this scanline
}

// Amounts of RAM for sample and scanline buffers
// a typical sweep is ~5000 scanlines, with a max of say 4K samples,
// for a total of 20 M samples = 40 MB.  We try for a buffer
// of roughly 5 of these, so 200 MB for sample memory, and
// 5 * 5000 = 25 K scanlines ~0.9 MB (@ 36 bytes per scanline)
const (
	SAMPLE_BUFF_SIZE   = 5 * 5000 * 4000 // number of samples in ring buffer
	SCANLINE_BUFF_SIZE = 5 * 5000        // number of scanlines in ring buffer
)

// SampleBuffer stores samples in a ring buffer.  Samples from each
// scanline are stored contiguously, so there will be empty space at
// the end of the sample buffer if the number of samples in a scanline
// doesn't divide into SAMPLE_BUFF_SIZE evenly.
type SampleBuffer struct {
	SampBuff [SAMPLE_BUFF_SIZE]Sample // ring buffer of samples
	iBuff    int                      // location for next sample to be written
	nSamples uint64                   // total samples captured during this run
}

// NextSliceFor Return the next slice in the SampleBuffer large enough to hold n samples,
// or nil if there is none.
func (sb *SampleBuffer) NextSliceFor(n int) (s []Sample) {
	if n > 0 && n <= len(sb.SampBuff) {
		if sb.iBuff+n > len(sb.SampBuff) {
			sb.iBuff = 0
		}
		s = sb.SampBuff[sb.iBuff : sb.iBuff+n]
		sb.iBuff += n
	}
	return
}

// ScanlineBuffer is a ring buffer of scanlines. Their samples are
// stored in the sample buffer.  A sweep might wrap around the end of
// the scanline buffer.
type ScanlineBuffer struct {
	Samples    *SampleBuffer                // location of sample ring buffer
	ScanBuff   [SCANLINE_BUFF_SIZE]Scanline // ring buffer of Scanline structs
	iBuff      int                          // location for next scanline to be written
	nScanlines uint64                       // total scanlines captured during this run
}

// Next returns the next Scanline for holding a scanline with n samples,
// or nil if there is none.
func (scb *ScanlineBuffer) Next(n int) (s *Scanline) {
	samps := scb.Samples.NextSliceFor(n)
	if samps != nil {
		if scb.iBuff >= len(scb.ScanBuff) {
			scb.iBuff = 0
		}
		s = &scb.ScanBuff[scb.iBuff]
		scb.iBuff++
		s.Samples = samps
	}
	return
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
