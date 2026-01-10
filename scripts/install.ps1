$ErrorActionPreference = "Stop"

$version = $env:VBUILD_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
  $version = "latest"
}

$arch = $null
try {
  $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
} catch {
}

$archString = "$arch"
if ([string]::IsNullOrWhiteSpace($archString)) {
  $archString = $env:PROCESSOR_ARCHITECTURE
}
if ($archString -eq "x86" -and -not [string]::IsNullOrWhiteSpace($env:PROCESSOR_ARCHITEW6432)) {
  $archString = $env:PROCESSOR_ARCHITEW6432
}
$archString = $archString.ToUpperInvariant()

switch ($archString) {
  "X64" { $archTag = "amd64" }
  "AMD64" { $archTag = "amd64" }
  default {
    Write-Error "unsupported architecture: $archString"
  }
}

if ($version -eq "latest") {
  $suffix = "lastest"
} elseif ($version -match "-lastest$") {
  $suffix = "lastest"
} else {
  $suffix = $version
}

$asset = "windows-$archTag-$suffix.exe"

if ($version -eq "latest") {
  $url = "https://github.com/vietrix/vbuild/releases/latest/download/$asset"
} else {
  $url = "https://github.com/vietrix/vbuild/releases/download/$version/$asset"
}

$installDir = Join-Path $env:USERPROFILE "AppData\Local\Programs\vbuild"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$dest = Join-Path $installDir "vbuild.exe"

Invoke-WebRequest -Uri $url -OutFile $dest

$path = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not ($path -split ";" | Where-Object { $_ -eq $installDir })) {
  if ([string]::IsNullOrWhiteSpace($path)) {
    $newPath = $installDir
  } else {
    $newPath = "$path;$installDir"
  }
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  $env:Path = "$env:Path;$installDir"
  Write-Output "Added vbuild to PATH. Restart your terminal to use it."
}

Write-Output "vbuild installed to $dest"
