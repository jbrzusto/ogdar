#!/bin/bash
if make; then
    echo Rebuilding on sgpidev
    echo rsync -av ./ pi@sgpidev:/proj/ogdar/
    ssh pi@sgpidev "cd proj/ogdar; git pull; make; build_cmds.sh"
    mkdir -p arm_targets
    scp pi@sgpidev:proj/ogdar/targets arm_targets
    sshpass -p root scp ~/ogdar root@rp1:
    sshpass -p root ssh root@rp1
fi
