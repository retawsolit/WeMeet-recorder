#!/usr/bin/env bash
set -euxo pipefail

export HOME=/home/wemeet
export XDG_RUNTIME_DIR=/tmp/xdg

mkdir -p /tmp/xdg /app
rm -rf /tmp/.X* /tmp/xdg/pulse 2>/dev/null || true

pulseaudio --kill || true

pulseaudio -D --exit-idle-time=-1 --disallow-exit

export DISPLAY=:99
Xvfb :99 -screen 0 1920x1080x24 -ac -nolisten tcp &

sleep 2

cd /app
exec WeMeet-recorder ${CONFIG_FILE:+--config "$CONFIG_FILE"} "$@"