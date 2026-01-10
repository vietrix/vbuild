$ErrorActionPreference = "Stop"

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

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

function Test-Url([string]$uri) {
  try {
    $params = @{
      Method             = "Head"
      Uri                = $uri
      Headers            = @{ "User-Agent" = "vbuild-installer" }
      MaximumRedirection = 10
      TimeoutSec         = 20
    }
    if ($PSVersionTable.PSVersion.Major -lt 6) {
      $params.UseBasicParsing = $true
    }
    $resp = Invoke-WebRequest @params
    if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400) {
      return $true
    }
    if ($resp.StatusCode -eq 404) {
      return $false
    }
  } catch {
    $response = $_.Exception.Response
    if ($response -and $response.StatusCode -eq 404) {
      return $false
    }
  }
  return $true
}

function Get-LatestTag {
  $headers = @{ "User-Agent" = "vbuild-installer" }
  $params = @{
    Uri        = "https://api.github.com/repos/vietrix/vbuild/releases/latest"
    Headers    = $headers
    TimeoutSec = 30
  }
  $resp = Invoke-RestMethod @params
  return $resp.tag_name
}

function Download-File([string]$uri, [string]$outFile) {
  if (-not (Test-Url $uri)) {
    return $false
  }
  $headers = @{ "User-Agent" = "vbuild-installer" }
  $maxAttempts = 3
  for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
    try {
      $params = @{
        Uri                = $uri
        OutFile            = $outFile
        Headers            = $headers
        MaximumRedirection = 10
        TimeoutSec         = 60
      }
      if ($PSVersionTable.PSVersion.Major -lt 6) {
        $params.UseBasicParsing = $true
      }
      Invoke-WebRequest @params
      return $true
    } catch {
      if ($attempt -lt $maxAttempts) {
        Start-Sleep -Seconds (2 * $attempt)
        continue
      }
    }
  }

  $curl = Get-Command curl.exe -ErrorAction SilentlyContinue
  if ($curl) {
    & $curl.Source -fsSL $uri -o $outFile
    if ($LASTEXITCODE -eq 0) {
      return $true
    }
  }

  $bits = Get-Command Start-BitsTransfer -ErrorAction SilentlyContinue
  if ($bits) {
    Start-BitsTransfer -Source $uri -Destination $outFile
    return $true
  }

  throw "download failed from $uri"
}

if ($version -eq "latest") {
  if (-not (Download-File $url $dest)) {
    $tag = Get-LatestTag
    if ([string]::IsNullOrWhiteSpace($tag)) {
      throw "failed to resolve latest release tag"
    }
    $asset = "windows-$archTag-$tag.exe"
    $url = "https://github.com/vietrix/vbuild/releases/download/$tag/$asset"
    if (-not (Download-File $url $dest)) {
      throw "release asset not found: $url"
    }
  }
} else {
  if (-not (Download-File $url $dest)) {
    throw "release asset not found: $url"
  }
}

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
