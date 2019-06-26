package main

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
	"github.com/jbrzusto/ogdar/buffer"
)

func main() {
	fmt.Print("About to initiate fpga access...\n")
	fpga := fpga.New()
	if fpga == nil {
		fmt.Print("Unable to access FPGA!\n\nThis program is for the redpitaya, not the RPI3\n\n")
		return
	}
	fmt.Print("About to read fpga registers...\n")
	fmt.Printf("ARP count: %d\nACP per ARP: %d\n", fpga.ARPCount, fpga.ACPPerARP)
	fpga.Close()
	buffer := buffer.SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))

}
