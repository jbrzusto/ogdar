package main

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
	"github.com/jbrzusto/ogdar/buffer"
	"time"
)

func main() {
	fpga := fpga.New()
	if fpga == nil {
		fmt.Print("Unable to access FPGA!\n\nThis program is for the redpitaya, not the RPI3\n\n")
		return
	}
	tc1 := fpga.TrigCount
	time.Sleep(time.Second)
	tc2 := fpga.TrigCount
	fmt.Printf("ARP count: %d\nACP per ARP: %d\nPRF: %d\n", fpga.ARPCount, fpga.ACPPerARP, tc2 - tc1)
	fpga.Close()
	buffer := buffer.SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))

}
