#!/bin/sh
cd "$(dirname "$0")"
export LD_LIBRARY_PATH="./libs:.:$LD_LIBRARY_PATH:/customer/lib:/mnt/SDCARD/miyoo/lib"
export SDL_VIDEODRIVER=mmiyoo
export SDL_AUDIODRIVER=mmiyoo
export EGL_VIDEODRIVER=mmiyoo

# Stop UI/audio to avoid conflicts
killall -9 MainUI 2>/dev/null
killall -9 audioserver 2>/dev/null
killall -9 audioserver.mod 2>/dev/null
sleep 1

# Run app with logging
./hmbrg > hmbrg.log 2>&1

# Restart MainUI
cd /mnt/SDCARD/miyoo/app
./MainUI &
