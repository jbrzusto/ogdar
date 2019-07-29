package main

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/fpga"
	. "github.com/jbrzusto/ogdar/buffer"
	"time"
//	"unsafe"
)

func main() {
	Fpga := New()
	fmt.Println("Got past New")
	fmt.Printf("Clocks is %d\n", Fpga.Clocks)
	for i, k := range Fpga.ControlKeys {
		if i != 0 {
			fmt.Printf("%-25s: %d\n", k, Fpga.RegsU32[i])
		}
	}
	fmt.Printf("DecRate @%p is %d\n", &Fpga.DecRate, Fpga.DecRate)
	fmt.Printf("TrigThreshRelax @%p is %d\n", &Fpga.TrigThreshRelax, Fpga.TrigThreshRelax)
	tc1 := Fpga.TrigCount
	fmt.Printf("Clocks @%p is %d\n", &Fpga.Clocks, Fpga.Clocks)
	time.Sleep(time.Second)
	fmt.Printf("Clocks is %d\n", Fpga.Clocks)
	Fpga.Clocks=12345678
	fmt.Printf("Clocks is %d\n", Fpga.Clocks)
	tc2 := Fpga.TrigCount
	fmt.Printf("ARP count: %d\nACP per ARP: %d\nPRF: %d\n", Fpga.ARPCount, Fpga.ACPPerARP, tc2 - tc1)
	buffer := SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))

}
