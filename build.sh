#!/bin/bash
if make; then
    echo Rebuilding on sgpidev
    echo rsync -av ./ pi@sgpidev:/proj/ogdar/
    ssh pi@sgpidev "cd proj/ogdar; git pull; make"
    scp pi@sgpidev:proj/ogdar/ogdar ~
    sshpass -p root scp ~/ogdar root@rp1:
    sshpass -p root ssh root@rp1
fi
