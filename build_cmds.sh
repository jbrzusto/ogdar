#!/bin/bash
#
# build accessories
#

mkdir -p targets
cd cmd
for x in *; do
    pushd $x
    go build $LDFLAGS_STATIC
    mv $x ../../targets
    popd
done
