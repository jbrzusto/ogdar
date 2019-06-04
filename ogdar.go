package main

import (
	"fpga"
)

func main() {
	fpga := fpga.Init()
	fmt.Printf("ARP count: %d\nACP per ARP: %d\n", fpga.ARPCount, fpga.ACPPerARP)
}
