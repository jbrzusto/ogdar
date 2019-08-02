package main

// Show one or more digdar registers at repeated intervals.
//
// Usage:
//
//    showreg N REGNAME1 M1 REGNAME2 M2 ...
//
// where
//  - N is the number of milliseconds to wait between burst reads of the
//    registers
//  - REGNAMEi is the name of a register
//  - Mi is the number of reads to do in a burst from the REGNAMEi

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
	"os"
	"reflect"
)
