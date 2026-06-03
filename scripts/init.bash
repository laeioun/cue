_cue_complete() {
    local result
    result=$(cue complete "$READLINE_LINE" "$READLINE_POINT" 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$result" ]; then
        READLINE_LINE="$result"
        READLINE_POINT="${#result}"
    fi
}

bind -x '"\t": _cue_complete'
