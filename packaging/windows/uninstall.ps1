#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$InstallDir = Join-Path $env:ProgramFiles "print.it"
$TaskName = "print.it"

Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue

Get-Process -Name "print.it" -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 1

if (Test-Path $InstallDir) {
    Remove-Item -Recurse -Force $InstallDir
}

Write-Host "print.it desinstalado."
