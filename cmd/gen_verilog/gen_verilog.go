package main

// Generate verilog snippets for the digdar FPGA build.
// The snippets create memory maps, register definitions, getters, setters, and pulsers
// for the registers defined in fpga/fpga.Regs

import (
	"fmt"
	"github.com/jbrzusto/ogdar/fpga"
	"os"
	"reflect"
)

// reg represents a 32 or 64-bit register in the FPGA
type reg struct {
	name    string       // name of the register visible to external code
	regname string       // internal name of the register
	size    int          // size in bits
	iswire  bool         // true if register is a wire (e.g. a register in a submodule)
	kind    reflect.Kind // kind (uint32 or uint64 or int32 or int64)
	offset  int          // offset in memory of low order byte of register
	desc    string       // human-readable description of field
	mode    string       // "rw", "r", or "p" (p for pulse or one-shot)
}

// MMap returns the verilog register memory map offset definition.
func (reg reg) MMap() string {
	switch reg.kind {
	case reflect.Uint32:
		fallthrough
	case reflect.Int32:
		return fmt.Sprintf("`define OFFSET_%-30s 20'h%06x // %s\n", reg.name, reg.offset, reg.desc)
	case reflect.Uint64:
		fallthrough
	case reflect.Int64:
		return fmt.Sprintf("`define OFFSET_%-30s 20'h%06x // low 32-bits: %s\n", reg.name+"_LO", reg.offset, reg.desc) +
			fmt.Sprintf("`define OFFSET_%-30s 20'h%06x // high 32-bits\n", reg.name+"_HI", reg.offset+4)
	}
	return ""
}

// Def returns the verilog register definition.
func (reg reg) Def() (rv string) {
	if reg.iswire {
		rv = "   wire"
	} else {
		rv = "   reg "
	}
	return rv + fmt.Sprintf(" [%d-1: 0] %-30s; // %s\n", reg.size, reg.regname, reg.desc)
}

// Getter returns the verilog clause for reading the register value.
// If the register's mode is "p", returns "".
// Uses 'ack' as the acknowledge signal, and 'rdata' as the data bus
func (reg reg) Getter() string {
	const (
		ack  = "ack"
		dbus = "rdata"
	)
	if reg.mode == "p" {
		return ""
	}
	switch reg.kind {
	case reflect.Uint32:
		fallthrough
	case reflect.Int32:
		return fmt.Sprintf("        `OFFSET_%-30s  : begin %s <= 1'b1;  %s <= %-30s[32-1: 0]; end\n", reg.name, ack, dbus, reg.regname)
	case reflect.Uint64:
		fallthrough
	case reflect.Int64:
		return fmt.Sprintf("        `OFFSET_%-30s  : begin %s <= 1'b1;  %s <= %-30s[32-1: 0]; end\n", reg.name+"_LO", ack, dbus, reg.regname) +
			fmt.Sprintf("        `OFFSET_%-30s  : begin %s <= 1'b1;  %s <= %-30s[64-1:32]; end\n", reg.name+"_HI", ack, dbus, reg.regname)
	}
	return ""
}

// Setter prints the verilog clause for writing the register value.
// If the register's mode is "r" or "p", the return value is "".
// Uses 'wdata' as the data bus.
func (reg reg) Setter() string {
	const (
		dbus = "wdata"
	)
	if reg.mode == "r" || reg.mode == "p" {
		return ""
	}
	switch reg.kind {
	case reflect.Uint32:
		fallthrough
	case reflect.Int32:
		return fmt.Sprintf("        `OFFSET_%-30s  : %-30s <= %s[32-1: 0];\n", reg.name, reg.regname, dbus)
	case reflect.Uint64:
		fallthrough
	case reflect.Int64:
		return fmt.Sprintf("        `OFFSET_%-30s  : %-30s[32-1: 0] <= %s[32-1: 0];\n", reg.name+"_LO", reg.regname, dbus) +
			fmt.Sprintf("        `OFFSET_%-30s  : %-30s[64-1:32] <= %s[32-1: 0];\n", reg.name+"_HI", reg.regname, dbus)
	}
	return ""
}

// Pulser prints the verilog clause for pulsing the register.
// This is similar to writing, except that the value is only stored
// in the register for a single clock cycle, and then the register is
// reset to zero.  If the register's mode is not "p", the return value is "".
// Uses 'wdata' as the data bus and 'addr' as the address bus.
func (reg reg) Pulser() string {
	const (
		dbus = "wdata"
		addr = "addr"
	)
	if reg.mode != "p" {
		return ""
	}
	switch reg.kind {
	case reflect.Uint32:
		fallthrough
	case reflect.Int32:
		return fmt.Sprintf("        %s <= {32{%s[19:0] == `OFFSET_%-30s}} & %s[32-1: 0];\n", reg.regname, addr, reg.name, dbus)
	case reflect.Uint64:
		fallthrough
	case reflect.Int64:
		return fmt.Sprintf("        %s[32-1: 0] <= {32{%s[19:0] == `OFFSET_%-30s}} & %s[32-1: 0];\n", reg.regname, addr, reg.name+"_LO", dbus) +
			fmt.Sprintf("        %s[64-1:32] <= {32{%s[19:0] == `OFFSET_%-30s}} & %s[32-1: 0];\n", reg.regname, addr, reg.name+"_HI", dbus)
	}
	return ""
}

// recExtractor is the type for the recursive register extractor
type recExtractor func(t reflect.Type, prefix string, offset int)

// ExtractRegs reads FPGA register definitions from a possibly nested struct
// and records them in a []reg.  Registers must be 32 or 64-bit int fields
// (signed or unsigned), and have these fields in their tag:
//    desc: human-readable description of register
//    reg: name of register used in FPGA logic (verilog files)
//    mode: "r", "rw", or "p"; these determine whether getter, setter, and or pulser logic is
//       generated for the register
//    prefix: used for nested structs which might be present as more than one copy.
//       The prefix is prepended to the names of registers in this copy.
//    is_wire: if "y", indicates the verilog code should treat this register as wires,
//       e.g. the register is contained in a submodule, such as counts of
//       TRG, ACP, and ARP pulses.  This is ignored if the register is part of
//       a struct with a non-empty prefix, in which case it is treated as a copy of
//       data originally obtained from wires.
func ExtractRegs(regs *[]reg, x interface{}) {
	var ext recExtractor
	ext = func(t reflect.Type, prefix string, offset int) {
		switch t.Kind() {
		case reflect.Ptr:
			// dereference a pointer to a struct
			ext(t.Elem(), prefix, offset)
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				switch f.Type.Kind() {
				case reflect.Struct:
					// recursively read nested struct
					ext(f.Type, f.Tag.Get("reg_prefix"), offset+int(f.Offset))
				case reflect.Uint32:
					fallthrough
				case reflect.Int32:
					fallthrough
				case reflect.Uint64:
					fallthrough
				case reflect.Int64:
					// register, 32 or 64 bits
					*regs = append(*regs, reg{kind: f.Type.Kind(), name: prefix + f.Name, offset: offset + int(f.Offset), desc: f.Tag.Get("desc"), size: 8 * int(f.Type.Size()), regname: prefix + f.Tag.Get("reg"), mode: f.Tag.Get("mode"), iswire: f.Tag.Get("is_wire") == "y" && prefix == ""})
				}
			}
		}
	}
	ext(reflect.TypeOf(x), "", 0)
}

func main() {
	regs := make([]reg, 0, 10)
	r := new(fpga.Regs)
	ExtractRegs(&regs, r)
	f, _ := os.Create("generated_mmap.v")
	fmt.Fprint(f, "// memory map definitions - generated by gen_verilog.go\n\n")
	for i := 0; i < len(regs); i++ {
		fmt.Fprint(f, regs[i].MMap())
	}
	f.Close()

	f, _ = os.Create("generated_regdefs.v")
	fmt.Fprint(f, "// register definitions - generated by gen_verilog.go\n\n")
	for i := 0; i < len(regs); i++ {
		fmt.Fprint(f, regs[i].Def())
	}
	f.Close()

	f, _ = os.Create("generated_getters.v")
	fmt.Fprint(f, "// getter logic - generated by gen_verilog.go\n\n")
	for i := 0; i < len(regs); i++ {
		fmt.Fprint(f, regs[i].Getter())
	}
	f.Close()

	f, _ = os.Create("generated_setters.v")
	fmt.Fprint(f, "// setter logic - generated by gen_verilog.go\n\n")
	for i := 0; i < len(regs); i++ {
		fmt.Fprint(f, regs[i].Setter())
	}
	f.Close()

	f, _ = os.Create("generated_pulsers.v")
	fmt.Fprint(f, "// pulser logic - generated by gen_verilog.go\n\n")
	for i := 0; i < len(regs); i++ {
		fmt.Fprint(f, regs[i].Pulser())
	}
	f.Close()
}
