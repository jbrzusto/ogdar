package main

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/fpga"
	. "github.com/jbrzusto/ogdar/buffer"
	"time"
//	"unsafe"
)

func main() {
	Init()
	fmt.Println("Got past New")
	fmt.Printf("Clocks is %d\n", Regs.Clocks)
	for i, k := range ControlKeys {
		if i != 0 {
			n, _ := GetRegByName(k)
			fmt.Printf("%-25s: %d\n", k, n)
		}
	}
	fmt.Printf("DecRate @%p is %d\n", &Regs.DecRate, Regs.DecRate)
	fmt.Printf("TrigThreshRelax @%p is %d\n", &Regs.TrigThreshRelax, Regs.TrigThreshRelax)
	tc1 := Regs.TrigCount
	arp, _ := GetRegByName("ARPRaw")
	fmt.Printf("Clocks @%p is %d, ARPRaw is %d, TrigAtARP=%d\n", &Regs.Clocks, Regs.Clocks, arp, Regs.TrigAtARP)
	time.Sleep(time.Second)
	arp, _ = GetRegByName("ARPRaw")
	fmt.Printf("Clocks @%p is %d, ARPRaw is %d, TrigAtARP=%d\n", &Regs.Clocks, Regs.Clocks, arp, Regs.TrigAtARP)
	fmt.Printf("Clocks is %d\n", Regs.Clocks)
	tc2 := Regs.TrigCount
	fmt.Printf("ARP count: %d\nACP per ARP: %d\nPRF: %d\n", Regs.ARPCount, Regs.ACPPerARP, tc2 - tc1)
	buffer := SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))
}
