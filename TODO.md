- scanline identified by 8-byte handle (uint64 alias) and this is stored in the samplebuf, occupying 4 samples.
   - [0:1] = 0xffff (NOT_A_SAMPLE)
   - [2:7] = `N`, the scanline serial number; it will be stored in the scanline buffer at `N mod (S)` where
     `S` is the size of the scanline buffer.
   - 0xFFFFFFFFFFFFFFFF is BAD_SCANLINE (would take > 4000 yrs to reach at PRF=2100 Hz)

```go
type ScanlineHandle uint64
const (
	BAD_SCANLINE ScanlineHandle = 0x0000000000000000
)

// scanLineIndex scanlineBuffer must hold < 65535 scanlines i.e. < 31.2s @ PRF=2100 Hz
type ScanlineIndex uint16
const (
	BAD_INDEX ScanlineIndex = 0x0000
)
// return the index of a scanline (by handle) scanbuffer
func (slb *ScanlineBuffer) indexOf (sh ScanlineHandle) ScanlineIndex {
    return (uint64(ScanlineHandle) & 0xFFFFFF) % len(slb.Scanlines)
}

```
- use ReadFrom and WriteTo methods for atomic copying of scanlines into and out
  of the buffer, with the guarantee that either the whole scanline is read or written,
    or none of it is.  ReadFrom returns a ScanlineHandle

- for SweepBuffer, use WriteNext method which accepts a sweep index and outputs the next sweep with that or greater index.  Returns
 int giving number of scanlines written.  Will only write entire scanlines, and only until filling s.)
```go
 func (sb *SweepBuffer) WriteNext(int sweepIndex, h *SweepHeader, s []Sample) int
```


- maybe a goroutine that looks after the scanline buffer:
   - insert() puts a new scanline in with data pointer, metadata; wraps
     when necessary; returns index ?
   - copyout(ID)

- need to re-think from API side:
  - NewSweep() - called when a scanline is received that should start a new sweep
    based on azi clock and cut
  -
