package main

import (
	"fmt"
	. "github.com/jbrzusto/ogdar/fpga"
	. "github.com/jbrzusto/ogdar/buffer"
	"time"
)

func main() {
	Init()
//	Reset()
	fmt.Printf("Clocks is %d\n", Regs.Clocks)
	for i := 1; i < NumRegs(); i++ {
		if i != 0 {
			n, _ := GetRegByIndex(i)
			fmt.Printf("%-25s: %x\n", RegName(i), n)
		}
	}
	fmt.Printf("DecRate @%p is %x\n", &Regs.DecRate, Regs.DecRate)
	ttr, _ := GetRegByName("TrigThreshRelax")
	fmt.Printf("TrigThreshRelax @%p is %x or via RegsU32 %x\n", &Regs.TrigThreshRelax, Regs.TrigThreshRelax, ttr )
	Regs.TrigThreshRelax = 0x5678
	SetRegByName("TrigThreshRelax", 0x5678)
	ttr, _ = GetRegByName("TrigThreshRelax")
	fmt.Printf("TrigThreshRelax @%p is %x or via RegsU32 %x\n", &Regs.TrigThreshRelax, Regs.TrigThreshRelax, ttr )
	tc1 := Regs.TrigCount
	arp, _ := GetRegByName("ARPRaw")
	fmt.Printf("Clocks @%p is %d, ARPRaw is %x, TrigAtARP=%d\n", &Regs.Clocks, Regs.Clocks, arp, Regs.TrigAtARP)
	time.Sleep(time.Second)
	arp, _ = GetRegByName("ARPRaw")
	fmt.Printf("Clocks @%p is %d, ARPRaw is %x, TrigAtARP=%d\n", &Regs.Clocks, Regs.Clocks, arp, Regs.TrigAtARP)
	Reset()
	time.Sleep(time.Second)
	arp, _ = GetRegByName("ARPRaw")
	fmt.Printf("Clocks @%p is %d, ARPRaw is %x, TrigAtARP=%d\n", &Regs.Clocks, Regs.Clocks, arp, Regs.TrigAtARP)
	tc2 := Regs.TrigCount
	fmt.Printf("ARP count: %d\nACP per ARP: %d\nPRF: %d\n", Regs.ARPCount, Regs.ACPPerARP, tc2 - tc1)
	buffer := SampleBuff{}
	fmt.Printf("Length of buffer is %d\n", len(buffer.SampBuff))
}
