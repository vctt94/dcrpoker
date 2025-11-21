#!/usr/bin/env bash
set -Eeuo pipefail

###############################################################################
# Settings
###############################################################################
SESSION=pokerui_session           # tmux session name

# bison-relay client
BRCLIENT_DIR=$HOME/projects/bisonrelay/brclient
CFG=$HOME/brclientdirs/dir2/brclient.conf
BRSERVER_PORT=12345                    # relay TCP port
BR_RPC_PORT=7778                       # client's WS RPC port

# poker UI
POKERUI_DIR=$HOME/projects/BR/pokerbisonrelay/pokerui

###############################################################################
# Restart session if it already exists
###############################################################################
tmux kill-session -t "$SESSION" 2>/dev/null || true

###############################################################################
# Window 0 – brclient
###############################################################################
tmux new-session -d -s "$SESSION" -n brclient "
until timeout 1 bash -c \"</dev/tcp/localhost/$BRSERVER_PORT\" 2>/dev/null; do
    echo 'waiting for brserver on :$BRSERVER_PORT'; sleep 3
done
cd \"$BRCLIENT_DIR\"
go build -o brclient
./brclient --cfg \"$CFG\"
"

###############################################################################
# Window 1 – poker UI (interactive shell, pane stays open)
###############################################################################
tmux new-window  -t "$SESSION":1 -n pokerui "$SHELL"

tmux send-keys  -t "$SESSION":1 "
until timeout 1 bash -c \"</dev/tcp/localhost/$BR_RPC_PORT\" 2>/dev/null; do
    echo 'waiting for WS on :$BR_RPC_PORT'; sleep 3
done
cd \"$POKERUI_DIR\"
echo 'generate golibbuilder'
go generate ./golibbuilder
echo '--- poker UI running (Ctrl-C to stop, ↑ to rerun) ---'
cd flutterui/pokerui
flutter run -d linux
" C-m

###############################################################################
# Start attached on window 0 (Prefix-2 to jump to poker UI)
###############################################################################
tmux select-window -t "$SESSION":0
tmux attach-session -t "$SESSION"

