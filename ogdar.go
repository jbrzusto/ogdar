package main

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
)

func main() {
	fpga := fpga.Open()
	if fpga == nil {
		fmt.Print("Unable to access FPGA!\n\nThis program is for the redpitaya, not the RPI3\n\n")
		return
	}
	fmt.Printf("ARP count: %d\nACP per ARP: %d\n", fpga.ARPCount, fpga.ACPPerARP)
	fpga.Close()
}
