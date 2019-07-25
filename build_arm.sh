#!/bin/bash
echo Rebuilding on sgpidev
echo Make sure you have pushed all commits to github.
echo -n "Hit Enter to continue..."
read x
sshpass -p raspberry ssh pi@sgpidev "cd proj/ogdar; git pull; GOPATH=/home/pi/proj/go/bin make"
mkdir -p targets/arm
sshpass -p raspberry scp pi@sgpidev:proj/ogdar/targets/* targets/arm
