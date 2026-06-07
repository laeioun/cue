$version = (Invoke-RestMethod https://api.github.com/repos/laeioun/cue/releases/latest).tag_name

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $arch = "amd64" }
    "ARM64" { $arch = "arm64" }
    default {
        Write-Error "Unsupported arch: $env:PROCESSOR_ARCHITECTURE"
        exit 1
    }
}

$url = "https://github.com/laeioun/cue/releases/download/$version/cue_windows_$arch"
$dest = "$env:LOCALAPPDATA\Programs\cue\cue.exe"
New-Item -ItemType Directory -Force (Split-Path $dest) | Out-Null
Invoke-WebRequest $url -OutFile $dest
$env:Path += ";$(Split-Path $dest)"
[Environment]::SetEnvironmentVariable("Path", $env:Path, "User")
& $dest install
Write-Host "cue $version installed - restart your terminal"
