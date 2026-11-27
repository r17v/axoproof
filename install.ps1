# AxoProof Windows Installer (PowerShell 5.1+ / PowerShell 7+)
# Usage: irm https://raw.githubusercontent.com/r17v/axoproof/main/install.ps1 | iex

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo = "r17v/axoproof"
$InstallDir = if ($env:AXOPROOF_INSTALL_DIR) { $env:AXOPROOF_INSTALL_DIR } else { Join-Path $HOME ".local\bin" }
$SkillDir = if ($env:AXOPROOF_SKILL_DIR) { $env:AXOPROOF_SKILL_DIR } else { Join-Path $HOME ".agents\skills\axoproof" }

$RawArch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
switch -Wildcard ($RawArch.ToUpperInvariant()) {
    "*ARM64*" { $Arch = "arm64" }
    "*AMD64*" { $Arch = "amd64" }
    "*X86_64*" { $Arch = "amd64" }
    default { throw "Unsupported Windows architecture: $RawArch" }
}

$AssetName = "axoproof-windows-$Arch.exe"
$BaseUrl = "https://github.com/$Repo/releases/latest/download"
$DownloadUrl = "$BaseUrl/$AssetName"
$ChecksumUrl = "$BaseUrl/checksums.txt"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$TmpExe = Join-Path ([System.IO.Path]::GetTempPath()) ("axoproof-" + [guid]::NewGuid().ToString("N") + ".exe")
$TmpSum = Join-Path ([System.IO.Path]::GetTempPath()) ("axoproof-sums-" + [guid]::NewGuid().ToString("N") + ".txt")
$TargetExe = Join-Path $InstallDir "axoproof.exe"

try {
    Write-Host "==> Downloading AxoProof (windows-$Arch) from $DownloadUrl ..."
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpExe -UseBasicParsing
    try {
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $TmpSum -UseBasicParsing
        $ExpectedLine = Get-Content $TmpSum | Where-Object { $_ -match "\s$([regex]::Escape($AssetName))$" } | Select-Object -First 1
        if ($ExpectedLine) {
            $ExpectedHash = ($ExpectedLine -split '\s+')[0].ToLowerInvariant()
            $ActualHash = (Get-FileHash -Path $TmpExe -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($ExpectedHash -ne $ActualHash) {
                throw "SHA-256 mismatch for $AssetName (expected $ExpectedHash, got $ActualHash)"
            }
            Write-Host "==> Verified SHA-256 checksum ($ActualHash)"
        }
    } catch {
        if ($_.Exception.Message -like "SHA-256 mismatch*") { throw }
    }
    Unblock-File -Path $TmpExe -ErrorAction SilentlyContinue
    Move-Item -Force -Path $TmpExe -Destination $TargetExe
    Write-Host "==> Installed AxoProof binary to $TargetExe"
} finally {
    Remove-Item -Force -Path $TmpExe -ErrorAction SilentlyContinue
    Remove-Item -Force -Path $TmpSum -ErrorAction SilentlyContinue
}

# Install 1-door Agent Skill + 6 modules
$RawBase = "https://raw.githubusercontent.com/$Repo/main/skills/axoproof"
New-Item -ItemType Directory -Force -Path (Join-Path $SkillDir "modules") | Out-Null
try {
    Invoke-WebRequest -Uri "$RawBase/SKILL.md" -OutFile (Join-Path $SkillDir "SKILL.md") -UseBasicParsing
    foreach ($Mod in @("no-placebo","cash-register","door-locks","live-persistence","customer-ready","self-repair")) {
        Invoke-WebRequest -Uri "$RawBase/modules/$Mod.md" -OutFile (Join-Path $SkillDir "modules\$Mod.md") -UseBasicParsing
    }
    Write-Host "==> Installed AxoProof Agent Skill to $SkillDir"
} catch {}

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathEntries = if ($UserPath) { $UserPath -split ';' | Where-Object { $_ -ne "" } } else { @() }
$AlreadyInUserPath = $false
foreach ($Entry in $PathEntries) {
    if ($Entry.TrimEnd('\') -ieq $InstallDir.TrimEnd('\')) {
        $AlreadyInUserPath = $true
        break
    }
}
if (-not $AlreadyInUserPath) {
    $NewUserPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("Path", $NewUserPath, "User")
    Write-Host "==> Added $InstallDir to your Windows User PATH"
}
if (($env:Path -split ';') -notcontains $InstallDir) {
    $env:Path = "$InstallDir;$env:Path"
}

& $TargetExe version
