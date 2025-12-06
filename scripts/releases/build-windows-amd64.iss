#define AppVersion "0.0.1"

[Setup]
AppName=Bison Poker
AppVersion={#AppVersion}
DefaultDirName={pf}\Bison Poker
DefaultGroupName=Bison Poker
OutputBaseFilename=bisonpoker-windows-amd64-{#AppVersion}
Compression=lzma
SolidCompression=yes

[Files]
Source: "C:\Users\vctt\projects\pokerbisonrelay\pokerui\flutterui\pokerui\build\windows\x64\runner\Release\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs

[Tasks]
Name: "desktopicon"; Description: "Create &desktop shortcut"; GroupDescription: "Additional icons:";

[Icons]
Name: "{group}\Bison Poker {#AppVersion}"; Filename: "{app}\bisonpoker.exe"
Name: "{group}\Uninstall Bison Poker"; Filename: "{uninstallexe}"

; Desktop (only if the checkbox is ticked)
Name: "{userdesktop}\Bison Poker {#AppVersion}"; Filename: "{app}\bisonpoker.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\bisonpoker.exe"; Description: "Launch Bison Poker {#AppVersion}"; Flags: nowait postinstall skipifsilent
