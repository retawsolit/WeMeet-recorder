#!/usr/bin/env bash
set -euxo pipefail

# PulseAudio cleanup
mkdir -p /run/pulse /var/run/pulse /var/lib/pulse /root/.config/pulse "${XDG_RUNTIME_DIR:-/tmp/xdg}"
rm -rf /tmp/.X* /run/pulse/* /var/run/pulse/* /var/lib/pulse/* /root/.config/pulse/* "${XDG_RUNTIME_DIR:-/tmp/xdg}"/pulse 2>/dev/null || true

# Kill PulseAudio if already running (prevent crash on restart)
pulseaudio --check && pulseaudio --kill || true

# Start PulseAudio in system mode (required for root user)
pulseaudio --system -D --verbose --exit-idle-time=-1 --disallow-exit || true

# Move to working dir
mkdir -p /app
cd /app

# Run recorder with optional config path
exec plugnmeet-recorder ${CONFIG_FILE:+--config "$CONFIG_FILE"} "$@"
