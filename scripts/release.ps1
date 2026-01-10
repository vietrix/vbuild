$ErrorActionPreference = "Stop"

$version = $env:VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
  $version = "dev"
}

if ($version -eq "latest" -or $version -match "-lastest$") {
  $suffix = "lastest"
} else {
  $suffix = $version
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = Resolve-Path (Join-Path $scriptDir "..")
$distDir = Join-Path $rootDir "dist"

if (-not (Test-Path $distDir)) {
  New-Item -ItemType Directory -Path $distDir | Out-Null
}

Set-Location $rootDir

function Build([string]$os, [string]$arch, [string]$output) {
  $env:GOOS = $os
  $env:GOARCH = $arch
  $env:CGO_ENABLED = "0"
  $outPath = Join-Path $distDir $output
  go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$version" -o $outPath ./cmd/vbuild
}

function WriteChecksum([string]$file) {
  $path = Join-Path $distDir $file
  $hash = (Get-FileHash -Algorithm SHA256 -Path $path).Hash.ToLower()
  $line = "$hash  $file"
  $checksumPath = Join-Path $distDir "$file.sha256"
  Set-Content -Path $checksumPath -Value $line -Encoding ASCII
}

Build "linux" "amd64" "linux-amd64-$suffix"
Build "linux" "arm64" "linux-arm64-$suffix"
Build "darwin" "amd64" "darwin-amd64-$suffix"
Build "darwin" "arm64" "darwin-arm64-$suffix"
Build "windows" "amd64" "windows-amd64-$suffix.exe"

WriteChecksum "linux-amd64-$suffix"
WriteChecksum "linux-arm64-$suffix"
WriteChecksum "darwin-amd64-$suffix"
WriteChecksum "darwin-arm64-$suffix"
WriteChecksum "windows-amd64-$suffix.exe"

Write-Output "release artifacts in $distDir"
