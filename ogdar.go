package main

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
)

func main() {
	fpga := fpga.Init()
	fmt.Printf("ARP count: %d\nACP per ARP: %d\n", fpga.Ogd.ARPCount, fpga.Ogd.ACPPerARP)
}
