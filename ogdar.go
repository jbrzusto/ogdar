package main

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/buffer"
	. "github.com/jbrzusto/ogdar/fpga"
	"time"
)

// Radar holds information about the radar.  This will be filled in from
// the config file.
var Radar radar


// keep track of whether a valid config file was found
// so we can show the user on the web interface.
var configFound bool

func main() {
	Init()
	configFound = loadConfig()
	if !configFound {
		fmt.Println("--- CRITICAL WARNING! ---\n\n  Config file 'ogdar.toml' not found.\n\nI am using a (likely bogus) default config.\n\n")
		setDefaultConfig()
	}
	fmt.Printf("Using radar: \n%+v\n", Radar)
	buffer := SampleBuff{}
	clks, _ := GetRegPtrByName("Clocks_lo")
	fmt.Printf("Clocks pointer is %p\n", clks)
	fmt.Printf("Clocks is %d\n", *clks)
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))
	for i := 1; i < NumRegs(); i++ {
		if i != 0 {
			p, _ := GetRegPtrByIndex(i)
			fmt.Printf("%-25s: *(%p) = %d\n", RegName(i), p, *p)
		}
	}
	nt := Regs.TrigCount
	for i := 1; i < 100; i++ {
		time.Sleep(time.Second)
		fmt.Printf("Clocks = %d, PRF = %d, ARPCount = %d, ACPPerARP = %d\n", *clks, (Regs.TrigCount-nt)/uint32(i), Regs.ARPCount, Regs.ACPPerARP)
	}
	Fini()
}
