// Copyright (c) The Conn3x Authors
// SPDX-License-Identifier: MPL-2.0

//go:build windowshost

package windowshost

import (
	"testing"
)

// A probe for #1107, run on the fork only: what Chocolatey on a runner
// does, as the windows_choco_package design depends on it. It logs and
// asserts nothing.
func TestChocoProbe(t *testing.T) {
	c := connect(t)
	out := run(t, c, `
$ErrorActionPreference = 'Continue'
$r = [ordered]@{}
$root = [Environment]::GetEnvironmentVariable('ChocolateyInstall', 'Machine')
$r.machine_env = $root
$r.session_env = $env:ChocolateyInstall
$r.version = (& choco --version 2>&1 | Out-String).Trim()
$r.lib_is_link = [bool]((Get-Item -LiteralPath (Join-Path $root 'lib')).Attributes -band [IO.FileAttributes]::ReparsePoint)
$r.enhanced_feature = (& choco feature list --limit-output 2>&1 | Select-String -SimpleMatch 'useEnhancedExitCodes' | Out-String).Trim()

# Two packages in a local feed: c3xprobe depends on c3xdep; c3xreboot asks for a reboot.
$feed = Join-Path $env:TEMP ('c3xfeed' + [guid]::NewGuid().ToString('N').Substring(0, 6))
$null = New-Item -ItemType Directory -Path $feed
function Pack($id, $version, $deps, $install) {
  $d = Join-Path $feed "src-$id"
  $null = New-Item -ItemType Directory -Path (Join-Path $d 'tools') -Force
  $depXml = ($deps | ForEach-Object { "<dependency id=""$($_)"" version=""1.0.0"" />" }) -join ''
  Set-Content -LiteralPath (Join-Path $d "$id.nuspec") -Encoding UTF8 -Value @"
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata><id>$id</id><version>$version</version><title>$id</title><authors>probe</authors><description>probe</description>
  <dependencies>$depXml</dependencies></metadata>
  <files><file src="tools\**" target="tools" /></files>
</package>
"@
  Set-Content -LiteralPath (Join-Path $d 'tools\chocolateyInstall.ps1') -Value $install
  Push-Location $d; & choco pack --out $feed 2>&1 | Out-Null; Pop-Location
}
Pack 'c3xdep' '1.0.0' @() 'Write-Host dep'
Pack 'C3xProbe' '1.2' @('c3xdep') 'Write-Host probe'
Pack 'c3xreboot' '1.0.0' @() 'Set-PowerShellExitCode 3010'
$r.feed = @(Get-ChildItem -LiteralPath $feed -Filter *.nupkg | ForEach-Object Name)

& choco install c3xprobe -s $feed -y --no-progress 2>&1 | Out-Null
$r.install_exit = $LASTEXITCODE
$r.lib_after_install = @(Get-ChildItem -LiteralPath (Join-Path $root 'lib') | Where-Object Name -like 'c3x*' | ForEach-Object Name)
$nuspec = Get-ChildItem -LiteralPath (Join-Path $root 'lib') -Recurse -Filter '*.nuspec' | Where-Object FullName -like '*c3xprobe*'
$r.nuspec_path = @($nuspec | ForEach-Object FullName)
$r.nuspec_version = if ($nuspec) { ([xml](Get-Content -LiteralPath $nuspec[0].FullName -Raw)).package.metadata.version } else { '' }
$r.list_local = (& choco list --limit-output 2>&1 | Select-String 'c3x' | Out-String).Trim()

# Pins: what files appear or change.
$before = @{}; Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue | ForEach-Object { $before[$_.FullName] = $_.LastWriteTimeUtc }
Start-Sleep -Seconds 1
& choco pin add -n c3xprobe 2>&1 | Out-Null
$r.pin_exit = $LASTEXITCODE
$r.pin_changed = @(Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue | Where-Object { -not $before.ContainsKey($_.FullName) -or $before[$_.FullName] -ne $_.LastWriteTimeUtc } | ForEach-Object { $_.FullName.Substring($root.Length) })
$r.pin_list = (& choco pin list --limit-output 2>&1 | Out-String).Trim()
& choco pin remove -n c3xprobe 2>&1 | Out-Null

# Reboot exit codes, without and with enhanced exit codes.
& choco install c3xreboot -s $feed -y --no-progress 2>&1 | Out-Null
$r.reboot_exit_plain = $LASTEXITCODE
& choco uninstall c3xreboot -y 2>&1 | Out-Null
& choco install c3xreboot -s $feed -y --no-progress --use-enhanced-exit-codes 2>&1 | Out-Null
$r.reboot_exit_enhanced = $LASTEXITCODE
& choco install c3xreboot -s $feed -y --no-progress --use-enhanced-exit-codes 2>&1 | Out-Null
$r.already_installed_enhanced = $LASTEXITCODE
& choco uninstall c3xreboot -y 2>&1 | Out-Null
& choco install c3xnonesuch -s $feed -y --no-progress --use-enhanced-exit-codes 2>&1 | Out-Null
$r.missing_package_enhanced = $LASTEXITCODE

# Uninstalling the dependent: does its dependency stay?
& choco uninstall c3xprobe -y 2>&1 | Out-Null
$r.uninstall_exit = $LASTEXITCODE
$r.lib_after_uninstall = @(Get-ChildItem -LiteralPath (Join-Path $root 'lib') | Where-Object Name -like 'c3x*' | ForEach-Object Name)
& choco uninstall c3xdep -y 2>&1 | Out-Null
$r | ConvertTo-Json -Depth 4
`, nil)
	t.Logf("choco probe:\n%s", out.Text)
}
