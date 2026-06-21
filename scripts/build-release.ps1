# Build Windows GUI release binary (version from version/version.go).
$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

$versionLine = Get-Content "version\version.go" -Raw
if ($versionLine -match 'Version = "(v[^"]+)"') {
    $ver = $Matches[1]
} else {
    throw "cannot read version from version/version.go"
}

Write-Host "Building $ver -> qzone-history-gui.exe"
go build -ldflags="-H windowsgui -s -w -X qzone-history/version.Version=$ver" -o qzone-history-gui.exe ./cmd/main.go

$fi = Get-Item qzone-history-gui.exe
Write-Host "Done: $($fi.Length) bytes"
Write-Host "Commit: version/version.go README.md qzone-history-gui.exe"
