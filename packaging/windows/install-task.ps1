#Requires -Version 5.1
param(
    [string]$InstallUser = $env:USERNAME
)

$ErrorActionPreference = "Stop"

$InstallDir = Join-Path $env:ProgramFiles "print.it"
$TaskName = "print.it"
$ExePath = Join-Path $InstallDir "print.it.exe"
$LogDir = Join-Path $env:ProgramData "print.it"
$LogPath = Join-Path $LogDir "install.log"

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

function Write-InstallLog {
    param([string]$Message)
    $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $Message"
    Add-Content -Path $LogPath -Value $line
    Write-Host $Message
}

if (-not (Test-Path $ExePath)) {
    throw "print.it.exe nao encontrado em $InstallDir"
}

if ($InstallUser -match '\\') {
    $UserId = $InstallUser
} else {
    $UserId = "$env:USERDOMAIN\$InstallUser"
}
$UserNameOnly = $UserId.Split('\')[-1]

Write-InstallLog "Registrando print.it para $UserId"

$action = New-ScheduledTaskAction -Execute $ExePath -WorkingDirectory $InstallDir
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $UserNameOnly
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 999 `
    -RestartInterval (New-TimeSpan -Minutes 1)
$principal = New-ScheduledTaskPrincipal -UserId $UserId -LogonType Interactive -RunLevel Limited

Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue
Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null

Write-InstallLog "Tarefa agendada criada. Iniciando agente agora..."
Start-Process -FilePath $ExePath -WorkingDirectory $InstallDir -WindowStyle Hidden

Start-Sleep -Seconds 3

try {
    $resp = Invoke-WebRequest -Uri "http://127.0.0.1:9280/printit/health" -UseBasicParsing -TimeoutSec 5
    if ($resp.StatusCode -eq 200) {
        Write-InstallLog "Agente respondendo em http://127.0.0.1:9280/printit/health"
    }
} catch {
    Write-InstallLog "AVISO: agente ainda nao respondeu. Verifique $env:APPDATA\print.it\logs\print.it.log"
}

Write-InstallLog "Instalacao concluida."
