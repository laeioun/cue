_cue_complete() {
    local result
    result=$(cue complete "$BUFFER" "$CURSOR" 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$result" ]; then
        BUFFER="$result"
        CURSOR="${#BUFFER}"
    fi
    zle reset-prompt
}

zle -N _cue_complete
bindkey '\t' _cue_complete
