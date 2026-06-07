$ErrorActionPreference = "Stop"

$version = (Invoke-RestMethod https://api.github.com/repos/laeioun/cue/releases/latest).tag_name

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $arch = "amd64" }
    "ARM64" { $arch = "arm64" }
    default {
        Write-Error "Unsupported arch: $env:PROCESSOR_ARCHITECTURE"
        exit 1
    }
}

$url = "https://github.com/laeioun/cue/releases/download/$version/cue_windows_$arch.exe"
$installDir = "$env:LOCALAPPDATA\Programs\cue"
$dest = "$installDir\cue.exe"
New-Item -ItemType Directory -Force $installDir | Out-Null
Invoke-WebRequest $url -OutFile $dest

if (($env:Path -split ";") -notcontains $installDir) {
    $env:Path += ";$installDir"
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -notcontains $installDir) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
}

& $dest install
Write-Host "cue $version installed - restart your terminal"
