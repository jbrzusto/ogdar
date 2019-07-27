#!/bin/bash
echo Rebuilding on sgpidev
echo -n "Hit Enter to continue..."
read x
sshpass -p raspberry rsync -av --exclude .git --exclude targets ~/proj/ogdar/ pi@sgpidev:proj/ogdar/
sshpass -p raspberry ssh pi@sgpidev "cd proj/ogdar; PATH=/home/pi/proj/go/bin:$PATH GOROOT=/home/pi/proj/go make"
mkdir -p targets/arm
sshpass -p raspberry scp pi@sgpidev:proj/ogdar/targets/* targets/arm
