# Makefile for ogdar
# Since we are building on the RPi3 for the redpitaya,
# we need to make static executables.

ogdar: ogdar.go fpga/fpga.go
	go build  -ldflags "-linkmode external -extldflags -static"
