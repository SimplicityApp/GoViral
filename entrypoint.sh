#!/bin/sh
set -e

# Fix volume permissions (Fly.io mounts as root)
if [ -d /home/goviral/.goviral ]; then
  chown -R goviral:goviral /home/goviral/.goviral
fi

# Warm up Python venv + packages on first boot so the first API call isn't slow.
# Skip if venv already exists (persisted on volume across restarts).
if [ ! -f /home/goviral/.goviral/venv/bin/python3 ]; then
  echo "First boot: creating Python venv and installing packages..."
  gosu goviral sh -c '
    python3 -m venv /home/goviral/.goviral/venv
    /home/goviral/.goviral/venv/bin/pip install -q twikit linkitin
  '
  echo "Python setup complete."
fi

# gosu replaces the current process (no fork), so the Go binary becomes PID 1
# and receives SIGTERM directly from Fly.io for graceful shutdown.
exec gosu goviral goviral
