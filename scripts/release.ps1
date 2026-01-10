$ErrorActionPreference = "Stop"

$version = $env:VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
  $version = "dev"
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

Build "linux" "amd64" "linux-amd64-$version"
Build "linux" "arm64" "linux-arm64-$version"
Build "darwin" "amd64" "darwin-amd64-$version"
Build "darwin" "arm64" "darwin-arm64-$version"
Build "windows" "amd64" "windows-amd64-$version.exe"

WriteChecksum "linux-amd64-$version"
WriteChecksum "linux-arm64-$version"
WriteChecksum "darwin-amd64-$version"
WriteChecksum "darwin-arm64-$version"
WriteChecksum "windows-amd64-$version.exe"

Write-Output "release artifacts in $distDir"
