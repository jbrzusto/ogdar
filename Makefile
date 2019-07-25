# Makefile for ogdar
# Since we are building on the RPi3 for the redpitaya,
# we need to make static executables.

all: ogdar cmds

ogdar: ogdar.go fpga/fpga.go buffer/buffer.go
	go build  -ldflags "-linkmode external -extldflags -static"
	mkdir -p targets
	mv ogdar targets
cmds:
	./build_cmds.sh
