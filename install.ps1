#Requires -Version 5.1
<#
.SYNOPSIS
  Install or upgrade snitchX on Windows.

.DESCRIPTION
  Downloads a snitchX release from GitHub, verifies its checksum when
  possible, and installs snitchx.exe. Detects the processor architecture,
  existing installations, and whether elevation is available.

.PARAMETER Version
  Release tag or version to install (for example v0.2.0 or 0.2.0).
  Defaults to the latest GitHub release.

.PARAMETER InstallDir
  Installation directory. Defaults to:
    - %ProgramFiles%\snitchX when running as Administrator
    - %LOCALAPPDATA%\Programs\snitchX otherwise

.PARAMETER Force
  Reinstall the same version, allow a downgrade, and stop a running snitchx.exe.

.PARAMETER NoPath
  Do not add the install directory to PATH.

.EXAMPLE
  irm https://raw.githubusercontent.com/ahzslanh/snitchx/master/install.ps1 | iex

.EXAMPLE
  .\install.ps1 -Version v0.2.0 -InstallDir "$env:LOCALAPPDATA\Programs\snitchX"
#>
[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "",
    [switch]$Force,
    [switch]$NoPath
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repo = "ahzslanh/snitchx"
$BinaryName = "snitchx.exe"
$ProjectName = "snitchx"

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-PlatformArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) {
        $arch = $env:PROCESSOR_ARCHITEW6432
    }
    switch ($arch.ToUpperInvariant()) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { throw "unsupported Windows architecture '$arch' (supported: amd64, arm64)" }
    }
}

function Get-NormalizedVersion([string]$Value) {
    if (-not $Value) { return "" }
    return $Value.Trim().TrimStart("v")
}

function Compare-VersionStrings([string]$Left, [string]$Right) {
    try {
        return ([version]$Left).CompareTo([version]$Right)
    } catch {
        return [string]::CompareOrdinal($Left, $Right)
    }
}

function Get-InstalledVersion([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        return $null
    }

    try {
        $json = & $Path version --json 2>$null
        if ($LASTEXITCODE -eq 0 -and $json) {
            $jsonText = $json -join "`n"
            $info = $jsonText | ConvertFrom-Json
            if ($info.version) {
                return [string]$info.version
            }
        }
    } catch {
        # older builds do not support --json
    }

    try {
        $line = & $Path version 2>$null | Select-Object -First 1
        if ($line) {
            $line = $line -replace '\x1b\[[0-9;]*m', ''
            if ($line -match '^\s*snitchx\s+(\S+)') {
                return $matches[1]
            }
        }
    } catch {
        # version output could not be parsed
    }

    return $null
}

function Add-ToPath([string]$Directory, [bool]$Machine) {
    $scope = if ($Machine) { "Machine" } else { "User" }
    $current = [Environment]::GetEnvironmentVariable("Path", $scope)
    if (-not $current) { $current = "" }

    $entries = @($current -split ';' | Where-Object { $_ -ne "" })
    foreach ($entry in $entries) {
        if ($entry.TrimEnd('\') -ieq $Directory.TrimEnd('\')) {
            return $false
        }
    }

    $newPath = if ($current.TrimEnd(';')) {
        $current.TrimEnd(';') + ";" + $Directory
    } else {
        $Directory
    }
    [Environment]::SetEnvironmentVariable("Path", $newPath, $scope)
    return $true
}

try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
} catch {
    # older .NET runtimes may not expose TLS 1.2; downloads will report the error
}

$isAdmin = Test-IsAdmin
$arch = Get-PlatformArch

if (-not $InstallDir) {
    if ($isAdmin) {
        $InstallDir = Join-Path $env:ProgramFiles "snitchX"
    } else {
        $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\snitchX"
    }
}
$InstallDir = [IO.Path]::GetFullPath($InstallDir)
$target = Join-Path $InstallDir $BinaryName

if (-not $isAdmin -and $InstallDir.StartsWith([IO.Path]::GetFullPath($env:ProgramFiles), [StringComparison]::OrdinalIgnoreCase)) {
    throw "installing to '$InstallDir' requires Administrator privileges. Re-run PowerShell as Administrator, or omit -InstallDir for a per-user install."
}

$installedVersion = Get-InstalledVersion $target
if ($installedVersion) {
    Write-Host "existing installation: snitchX $installedVersion ($target)"
} elseif (Test-Path -LiteralPath $target) {
    Write-Host "existing installation at $target (version unknown)"
} else {
    Write-Host "no existing snitchX installation found"
}

if (-not $Version) {
    Write-Host "checking the latest release..."
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
        -Headers @{ "User-Agent" = "snitchx-installer" }
    $tag = [string]$release.tag_name
    if (-not $tag) {
        throw "could not determine the latest snitchX release"
    }
} else {
    $tag = $Version
    if (-not $tag.StartsWith("v")) {
        $tag = "v$tag"
    }
}
$cleanVersion = Get-NormalizedVersion $tag

$installedClean = Get-NormalizedVersion $installedVersion
if ($installedClean -and $installedClean -eq $cleanVersion -and -not $Force) {
    Write-Host "snitchX $cleanVersion is already installed (use -Force to reinstall)"
    if (-not $NoPath) {
        if (Add-ToPath $InstallDir $isAdmin) {
            Write-Host "added $InstallDir to PATH; restart your terminal"
        }
    }
    exit 0
}

if ($installedClean -and $installedClean -ne "dev") {
    $comparison = Compare-VersionStrings $installedClean $cleanVersion
    if ($comparison -gt 0 -and -not $Force) {
        throw "requested version $cleanVersion is older than installed version $installedClean (use -Force to downgrade)"
    }
    Write-Host "upgrading snitchX $installedVersion -> $cleanVersion"
} else {
    Write-Host "installing snitchX $cleanVersion"
}

$archiveName = "${ProjectName}_${cleanVersion}_windows_${arch}.tar.gz"
$baseUrl = "https://github.com/$Repo/releases/download/$tag"
$downloadUrl = "$baseUrl/$archiveName"

$running = @()
try {
    $running = @(Get-Process -Name $ProjectName -ErrorAction SilentlyContinue | Where-Object {
        try { $_.Path -and $_.Path -ieq $target } catch { $false }
    })
} catch {
    $running = @()
}

if ($running.Count -gt 0) {
    if (-not $Force) {
        throw "snitchX is running (PID $($running.Id -join ', ')). Close it and retry, or pass -Force."
    }
    Write-Host "stopping running snitchX process..."
    $running | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 500
}

$tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("snitchx-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

try {
    $archivePath = Join-Path $tmpDir $archiveName
    Write-Host "downloading $downloadUrl"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing

    # Verify the checksum when the release publishes checksums.txt.
    $checksumsPath = Join-Path $tmpDir "checksums.txt"
    try {
        Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksumsPath -UseBasicParsing
        $match = Get-Content -LiteralPath $checksumsPath |
            Where-Object { $_ -match "\s+$([regex]::Escape($archiveName))$" } |
            Select-Object -First 1
        if ($match) {
            $expectedHash = ($match -split '\s+')[0].ToLowerInvariant()
            $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
            if ($expectedHash -ne $actualHash) {
                throw "checksum verification failed for $archiveName"
            }
            Write-Host "checksum verified"
        } else {
            Write-Warning "checksum entry for $archiveName not found; skipping verification"
        }
    } catch {
        if ($_.Exception.Message -like "*checksum verification failed*") {
            throw
        }
        Write-Warning "could not verify checksum: $($_.Exception.Message)"
    }

    $tar = Get-Command tar.exe -ErrorAction SilentlyContinue
    if (-not $tar) {
        throw "tar.exe was not found. Windows 10 1803+ includes tar; install it or extract the release manually."
    }

    & $tar.Source -xzf $archivePath -C $tmpDir
    if ($LASTEXITCODE -ne 0) {
        throw "failed to extract $archiveName"
    }

    $newBinary = Join-Path $tmpDir $BinaryName
    if (-not (Test-Path -LiteralPath $newBinary)) {
        throw "$BinaryName was not found in $archiveName"
    }

    try {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        $probe = Join-Path $InstallDir ".snitchx-write-test"
        New-Item -ItemType File -Path $probe -Force | Out-Null
        Remove-Item -LiteralPath $probe -Force
    } catch {
        throw "no write access to '$InstallDir'. Re-run as Administrator or choose a writable -InstallDir."
    }

    $backupPath = "$target.bak"
    Remove-Item -LiteralPath $backupPath -Force -ErrorAction SilentlyContinue

    if (Test-Path -LiteralPath $target) {
        try {
            Move-Item -LiteralPath $target -Destination $backupPath -Force
        } catch {
            throw "could not back up '$target': $($_.Exception.Message)"
        }
    }

    try {
        Copy-Item -LiteralPath $newBinary -Destination $target -Force
    } catch {
        if (Test-Path -LiteralPath $backupPath) {
            Move-Item -LiteralPath $backupPath -Destination $target -Force -ErrorAction SilentlyContinue
        }
        throw "failed to install $BinaryName to '$target': $($_.Exception.Message)"
    }

    Remove-Item -LiteralPath $backupPath -Force -ErrorAction SilentlyContinue
    Unblock-File -LiteralPath $target -ErrorAction SilentlyContinue

    if (-not $NoPath) {
        if (Add-ToPath $InstallDir $isAdmin) {
            Write-Host "added $InstallDir to PATH; restart your terminal"
        }
    }

    $verifiedVersion = Get-InstalledVersion $target
    if (-not $verifiedVersion) {
        $verifiedVersion = $cleanVersion
    }
    Write-Host "installed snitchX $verifiedVersion to $target"
} finally {
    Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
