#define MyAppName "print.it"
#ifndef MyAppVersion
#define MyAppVersion "0.1.1"
#endif
#ifndef SetupLanguage
#define SetupLanguage "pt"
#endif
#define MyAppPublisher "IdeYou"
#define MyAppURL "https://github.com/ioisaque/print.it"
#define MyAppExeName "print.it.exe"

[Setup]
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
DefaultDirName={autopf}\print.it
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\..\dist
OutputBaseFilename=print.it-{#MyAppVersion}-windows-amd64
Compression=lzma2
SolidCompression=yes
ArchitecturesInstallIn64BitMode=x64
PrivilegesRequired=admin
UninstallDisplayIcon={app}\{#MyAppExeName}
ShowLanguageDialog=no

#if SetupLanguage == "en"
[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
#else
[Languages]
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"
#endif

[Files]
Source: "..\..\dist\print.it-windows-amd64.exe"; DestDir: "{app}"; DestName: "{#MyAppExeName}"; Flags: ignoreversion
Source: "install-task.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "uninstall.ps1"; DestDir: "{app}"; Flags: ignoreversion

[Run]
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\install-task.ps1"" -InstallDir ""{app}"" -InstallUser ""{username}"""; StatusMsg: "Configurando inicializacao automatica..."; Flags: runhidden waituntilterminated
Filename: "{app}\{#MyAppExeName}"; Description: "Iniciar print.it"; StatusMsg: "Iniciando print.it..."; Flags: nowait runascurrentuser skipifsilent
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\install-task.ps1"" -InstallDir ""{app}"" -VerifyOnly"; StatusMsg: "Verificando print.it..."; Flags: runhidden waituntilterminated

[UninstallRun]
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\uninstall.ps1"""; Flags: runhidden waituntilterminated

[Code]
var
  Upgrading: Boolean;

function PreviouslyInstalled(): Boolean;
begin
  Result :=
    RegKeyExists(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}_is1') or
    RegKeyExists(HKLM, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}_is1');
end;

function InitializeSetup(): Boolean;
begin
  Upgrading := PreviouslyInstalled();
  Result := True;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
  StopCmd: String;
begin
  NeedsRestart := False;
  StopCmd := '-NoProfile -ExecutionPolicy Bypass -Command "$p = Get-Process -Name print.it -ErrorAction SilentlyContinue; if ($p) { $p | Stop-Process -Force; Start-Sleep -Seconds 2; if (Get-Process -Name print.it -ErrorAction SilentlyContinue) { exit 1 } }; exit 0"';

  if not Exec('powershell.exe', StopCmd, '', SW_HIDE, ewWaitUntilTerminated, ResultCode) then
  begin
    Result := 'Nao foi possivel parar o print.it. Feche o agente manualmente e tente novamente.';
    Exit;
  end;

  if ResultCode <> 0 then
  begin
    Result := 'O print.it ainda esta em execucao. Feche o agente manualmente e tente novamente.';
    Exit;
  end;

  Result := '';
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if (CurStep = ssPostInstall) and Upgrading then
  begin
    MsgBox('print.it atualizado para {#MyAppVersion}.', mbInformation, MB_OK);
  end;
end;

[Messages]
FinishedLabel=print.it foi instalado.%n%nO agente ja deve estar rodando.%n%nConfigure a impressora no sistema de gestao.
