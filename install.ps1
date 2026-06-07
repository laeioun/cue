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

$profilePath = $PROFILE.CurrentUserCurrentHost
New-Item -ItemType Directory -Force (Split-Path $profilePath) | Out-Null
$profileContent = if (Test-Path $profilePath) { Get-Content $profilePath -Raw } else { "" }
$installLine = 'Invoke-Expression (& { (cue init powershell | Out-String) })'

if ($profileContent -notlike "*cue init powershell*") {
    Add-Content -Path $profilePath -Value "`n$installLine"
}

Invoke-Expression (& { (& $dest init powershell | Out-String) })
Write-Host "cue $version installed - restart your terminal"
