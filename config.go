package main

// this file contains all the code that directly uses the viper package
// hopefully the go build system can avoid having to rebuild this every time.
import (
	. "github.com/jbrzusto/ogdar/fpga"
	"github.com/spf13/viper"
)

// loadConfig reads configuration from a TOML-formatted file called 'ogdar.toml'
// It looks for this in the /opt folder (which is the top-level of the SD card, on the
// current redpitaya linux image) and then in the current directory,
// for convenience.  The file must be called "ogdar.toml"
// Returns true if a config file was read.
func loadConfig() bool {
	viper.SetConfigName("ogdar") // name of config file (without extension)
	viper.AddConfigPath("/opt")  // path to look for the config file in
	viper.AddConfigPath(".")     // optionally look for config in the working directory
	err := viper.ReadInConfig()  // Find and read the config file
	if err != nil {              // Error reading the config file
		return false
	}
	// store the values in Regs; this will be pulse detection thresholds, decimation rates
	// and so on.  See 'ogdar.toml' for details.
	viper.UnmarshalKey("digdar", Regs)
	viper.UnmarshalKey("radar", &Radar)
	return true
}

// unsign gets around not being able to cast signed *constants*
// into an unsigned int.
func unsign (x int32) uint32 {
	return uint32(x)
}

// setDefaultConfig sets sane defaults for critical digitizing registers.
// This function should only be called if no other config information is
// available!  There is absolutely no guarantee that the values here make
// any sense for a particular radar, but they work for at least one of
// the test radars (a Furuno FR-8252 with CHS Lab's front-end board.)
func setDefaultConfig() {
	Regs.DecRate = 1
	Regs.NumSamp = 4000
	Regs.Options = 7
	Regs.TrigSource = 2
	Regs.TrigThreshExcite = unsign(-6550)
	Regs.TrigThreshRelax = unsign(-8000)
	Regs.TrigLatency = 12500
	Regs.TrigDelay = 30
	Regs.ACPThreshExcite = unsign(-1638)
	Regs.ACPThreshRelax = 1228
	Regs.ACPLatency = 500000
	Regs.ARPThreshExcite = unsign(-1638)
	Regs.ARPThreshRelax = 1228
	Regs.ARPLatency = 125000000
	Radar.Model = "WARNING: using default (bogus!) config because file ogdar.toml not found"
	Radar.PRF = 2100
	Radar.ACPsPerRotation = 450
	Radar.Power = 25000
}
