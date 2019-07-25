#!/bin/bash
echo Rebuilding on sgpidev
echo Make sure you have pushed all commits to github.
echo -n "Hit Enter to continue..."
read x
ssh pi@sgpidev "cd proj/ogdar; git pull; make; build_cmds.sh"
mkdir -p targets/arm
scp pi@sgpidev:proj/ogdar/targets/* targets/arm
