#Requires -Version 5.1
param(
    [Parameter(Mandatory = $true)]
    [string]$InstallDir,
    [string]$InstallUser = $env:USERNAME,
    [switch]$VerifyOnly
)

$ErrorActionPreference = "Stop"

$TaskName = "print.it"
$ExePath = Join-Path $InstallDir "print.it.exe"
$LogDir = Join-Path $env:ProgramData "print.it"
$InstallLog = Join-Path $LogDir "install.log"
$AgentLog = Join-Path $LogDir "logs\print.it.log"
$StartupLog = Join-Path $LogDir "logs\startup.log"

New-Item -ItemType Directory -Force -Path (Split-Path $AgentLog) | Out-Null

function Write-InstallLog {
    param([string]$Message)
    $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $Message"
    Add-Content -Path $InstallLog -Value $line
    Write-Host $Message
}

if ($VerifyOnly) {
    Start-Sleep -Seconds 3
    $healthy = $false
    try {
        $resp = Invoke-WebRequest -Uri "http://127.0.0.1:9280/printit/health" -UseBasicParsing -TimeoutSec 5
        $healthy = ($resp.StatusCode -eq 200)
    } catch {}

    if ($healthy) {
        Write-InstallLog "Agente respondendo em http://127.0.0.1:9280/printit/health"
        exit 0
    }

    Write-InstallLog "ERRO: agente nao respondeu apos instalacao"
    if (Test-Path $StartupLog) {
        Write-InstallLog "--- startup.log ---"
        Get-Content $StartupLog -Tail 20 | ForEach-Object { Write-InstallLog $_ }
    }
    if (Test-Path $AgentLog) {
        Write-InstallLog "--- print.it.log ---"
        Get-Content $AgentLog -Tail 20 | ForEach-Object { Write-InstallLog $_ }
    }

    Add-Type -AssemblyName PresentationFramework
    [System.Windows.MessageBox]::Show(
@"

O print.it foi instalado, mas o agente nao iniciou.

Teste manualmente no PowerShell:

  cd "$InstallDir"
  .\print.it.exe

Logs:
  $StartupLog

"@,
        "print.it",
        "OK",
        "Warning"
    ) | Out-Null
    exit 0
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

Write-InstallLog "Instalando em $InstallDir para $UserId"

$action = New-ScheduledTaskAction -Execute $ExePath -WorkingDirectory $InstallDir
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $UserNameOnly
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1)
$principal = New-ScheduledTaskPrincipal -UserId $UserId -LogonType Interactive -RunLevel Limited

Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue
Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null

Write-InstallLog "Tarefa agendada registrada."
Write-InstallLog "Instalacao concluida."
