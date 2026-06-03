Set-PSReadLineKeyHandler -Key Tab -ScriptBlock {
    $line = $null
    $cursor = $null
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    $result = & cue complete $line $cursor 2>$null
    if ($LASTEXITCODE -eq 0 -and $result) {
        [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, $result)
        [Microsoft.PowerShell.PSConsoleReadLine]::SetCursorPosition($result.Length)
    }
}
