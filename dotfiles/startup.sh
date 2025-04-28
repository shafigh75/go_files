#! /bin/bash

xrandr --output HDMI-1 --pos 1920x0 --output HDMI-2 --pos 0x0
setxkbmap -layout us,ir -option 'grp:alt_shift_toggle'
copyq --start-server
