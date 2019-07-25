#!/bin/bash
#
# build accessories
#

mkdir -p targets
cd cmd
for x in *; do
    cd $x
    go build -ldflags "-linkmode external -extldflags -static"
    mv $x ../../targets
done
