package main

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/fpga"
	. "github.com/jbrzusto/ogdar/buffer"
	"time"
)

func main() {
	Fpga := New()
	fmt.Println("Got past New")
	Fpga.MakeRegMap()
	fmt.Printf("Clocks is %d\n", Fpga.Clocks)
	// for _, k := range ControlKeys {
	// 	if k != "Command" {
	// 		fmt.Printf("%-25s @%p = %d\n", k, ControlMap[k], *ControlMap[k])
	// 	}
	// }
	fmt.Printf("DecRate is %d\n", Fpga.DecRate)
	fmt.Printf("TrigThreshRelax is %d\n", Fpga.TrigThreshRelax)
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
