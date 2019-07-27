# Makefile for ogdar
# Since we are building on the RPi3 for the redpitaya,
# we need to make static executables.

LDFLAGS_STATIC=-ldflags "-linkmode external -extldflags -static"
all: ogdar cmds

ogdar: ogdar.go fpga buffer
	go build $(LDFLAGS_STATIC)
	mkdir -p targets
	mv ogdar targets

cmds: fpga buffer
	./build_cmds.sh
