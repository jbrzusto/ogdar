package main

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/fpga"
	. "github.com/jbrzusto/ogdar/buffer"
	"time"
)

func main() {
	for _, k := range ControlKeys {
		if k != "Command" {
			fmt.Printf("%-25s @%p = %d\n", k, ControlMap[k], *ControlMap[k])
		}
	}
	tc1 := Fpga.TrigCount
	time.Sleep(time.Second)
	tc2 := Fpga.TrigCount
	fmt.Printf("ARP count: %d\nACP per ARP: %d\nPRF: %d\n", Fpga.ARPCount, Fpga.ACPPerARP, tc2 - tc1)
	buffer := SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))

}
