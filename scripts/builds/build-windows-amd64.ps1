param(
  [string]$Version = "v0.0.2",
  [string]$Iscc = "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$pokeruiDir = Join-Path $repoRoot "pokerui"
$flutterAppDir = Join-Path $pokeruiDir "flutterui\pokerui"
$issScript = Join-Path $repoRoot "scripts\builds\build-windows-amd64.iss"
$runnerDir = Join-Path $flutterAppDir "build\windows\x64\runner\Release"
$appVersion = $Version.TrimStart("v")

if (-not (Test-Path $Iscc)) {
  $cmd = Get-Command ISCC.exe -ErrorAction SilentlyContinue
  if ($cmd) {
    $Iscc = $cmd.Source
  } else {
    throw "ISCC.exe not found. Install Inno Setup or pass -Iscc with the full path."
  }
}

Push-Location $pokeruiDir
try {
  Write-Host "Generating golib.dll for Windows"
  go generate ./golibbuilder

  Write-Host "Building Windows release app"
  Push-Location $flutterAppDir
  try {
    flutter build windows --release
  } finally {
    Pop-Location
  }

  if (-not (Test-Path $runnerDir)) {
    throw "Runner output not found after build: $runnerDir"
  }

  if (-not (Test-Path $issScript)) {
    throw "Inno Setup script not found: $issScript"
  }

  Write-Host "Packaging Windows installer with $Iscc"
  & $Iscc "/DAppVersion=$appVersion" "/DRepoRoot=$repoRoot" $issScript

  if ($LASTEXITCODE -ne 0) {
    throw "Inno Setup packaging failed with exit code $LASTEXITCODE"
  }
} finally {
  Pop-Location
}
