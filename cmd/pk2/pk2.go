package main

// Peek/Poke memory FPGA registers

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/fpga"
)

func main() {
	Fpga := New()
	Fpga.Reset()
	fmt.Println("Initializing Fpga\n")
}
