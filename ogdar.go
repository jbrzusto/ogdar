package main

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
	"github.com/jbrzusto/ogdar/buffer"
	"time"
)

func main() {
	for x := range fpga.ControlMap {
		fmt.Printf("%s = %p\n", x, fpga.ControlMap[x])
	}
	tc1 := fpga.Fpga.TrigCount
	time.Sleep(time.Second)
	tc2 := fpga.Fpga.TrigCount
	fmt.Printf("ARP count: %d\nACP per ARP: %d\nPRF: %d\n", fpga.Fpga.ARPCount, fpga.Fpga.ACPPerARP, tc2 - tc1)
	buffer := buffer.SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))

}
