package main

// Radar represents information about a specific radar
type radar struct {
	Model string // name of the radar make/model; used for display and possibly in output files
	PRF uint16 // the approximate Pulse Repetition Frequency for the mode you want to digitize
	ACPsPerRotation uint16 // how many ACPs in one rotation of the antenna?
	Power uint16 // power radar transmits at, in watts.
}
