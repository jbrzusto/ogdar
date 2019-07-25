#!/bin/bash
#
# build accessories
#

mkdir -p targets
cd cmd
for x in *; do
    cd $x
    go build $LDFLAGS_STATIC
    mv $x ../../targets
done
