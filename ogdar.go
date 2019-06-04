package main

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
)

func main() {
	fpga := fpga.Init()
	if fpga == nil {
		fmt.Print("Unable to access FPGA!\nnThis program is for the redpitaya, not the RPI3\n\n")
	fmt.Printf("ARP count: %d\nACP per ARP: %d\n", fpga.Ogd.ARPCount, fpga.Ogd.ACPPerARP)
}
