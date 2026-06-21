#define MyAppName "print.it"
#ifndef MyAppVersion
#define MyAppVersion "0.3.2"
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

[Files]
Source: "..\..\dist\print.it-windows-amd64.exe"; DestDir: "{app}"; DestName: "{#MyAppExeName}"; Flags: ignoreversion
Source: "install-task.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "uninstall.ps1"; DestDir: "{app}"; Flags: ignoreversion

[Run]
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\install-task.ps1"""; StatusMsg: "Configurando inicializacao automatica..."; Flags: runhidden waituntilterminated

[UninstallRun]
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\uninstall.ps1"""; Flags: runhidden waituntilterminated

[Messages]
FinishedLabel=print.it foi instalado.%n%nConfigure a impressora no sistema de gestao.%n%nO agente iniciara automaticamente no proximo login.
