param(
  [string]$Version = "v0.0.1",
  [string]$Iscc = "iscc"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$pokeruiDir = Join-Path $repoRoot "pokerui"
$flutterAppDir = Join-Path $pokeruiDir "flutterui\pokerui"
$issScript = Join-Path $repoRoot "scripts\builds\build-windows-amd64.iss"
$runnerDir = Join-Path $flutterAppDir "build\windows\x64\runner\Release"
$appVersion = $Version.TrimStart("v")

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

  Write-Host "Packaging Windows installer"
  & $Iscc "/DAppVersion=$appVersion" "/DRepoRoot=$repoRoot" $issScript
  if ($LASTEXITCODE -ne 0) {
    throw "Inno Setup packaging failed with exit code $LASTEXITCODE"
  }
} finally {
  Pop-Location
}
