#!/usr/bin/env pwsh
# Copyright 2022 Deta authors. All rights reserved. MIT license.

$ErrorActionPreference = 'Stop'

if ($v) {
  if ($v[0] -match "v") {
    $Version = "${v}"
  } else {
    $Version = "v${v}"
  }
}

if ($args.Length -eq 1) {
  $Version = $args.Get(0)
}

$SpaceInstall = $env:space_INSTALL
$BinDir = if ($SpaceInstall) {
  "$SpaceInstall\bin"
} else {
  "$Home\.detaspace\bin"
}

$SpaceZip = "$BinDir\space.zip"
$SpaceExe = "$BinDir\space.exe"
$SpaceOldExe = "$env:Temp\spaceold.exe"
$Target = 'windows-x86_64'

# GitHub requires TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$SpaceUri = if (!$Version) {
  "https://github.com/deta/space-cli/releases/latest/download/space-${Target}.zip"
} else {
  "https://github.com/deta/space-cli/releases/download/${Version}/space-${Target}.zip"
}

if (!(Test-Path $BinDir)) {
  New-Item $BinDir -ItemType Directory | Out-Null
}

Invoke-WebRequest $SpaceUri -OutFile $SpaceZip -UseBasicParsing

if (Test-Path $SpaceExe) {
  Move-Item -Path $SpaceExe -Destination $SpaceOldExe -Force
}

if (Get-Command Expand-Archive -ErrorAction SilentlyContinue) {
  Expand-Archive $SpaceZip -Destination $BinDir -Force
} else {
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  [IO.Compression.ZipFile]::ExtractToDirectory($SpaceZip, $BinDir)
}

Remove-Item $SpaceZip

$User = [EnvironmentVariableTarget]::User
$Path = [Environment]::GetEnvironmentVariable('Path', $User)
if (!(";$Path;".ToLower() -like "*;$BinDir;*".ToLower())) {
  [Environment]::SetEnvironmentVariable('Path', "$Path;$BinDir", $User)
  $Env:Path += ";$BinDir"
}

Write-Output "Space was installed successfully to $SpaceExe"
Write-Output "Run 'space --help' to get started"
