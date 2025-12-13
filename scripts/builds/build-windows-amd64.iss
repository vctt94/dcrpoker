#define AppVersion "0.0.1"

[Setup]
AppName=Decred Poker
AppVersion={#AppVersion}
DefaultDirName={pf}\Decred Poker
DefaultGroupName=Decred Poker
OutputBaseFilename=dcrpoker-windows-amd64-{#AppVersion}
Compression=lzma
SolidCompression=yes

[Files]
Source: "C:\Users\vctt\projects\pokerbisonrelay\pokerui\flutterui\pokerui\build\windows\x64\runner\Release\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs

[Tasks]
Name: "desktopicon"; Description: "Create &desktop shortcut"; GroupDescription: "Additional icons:";

[Icons]
Name: "{group}\Decred Poker {#AppVersion}"; Filename: "{app}\dcrpoker.exe"
Name: "{group}\Uninstall Decred Poker"; Filename: "{uninstallexe}"

; Desktop (only if the checkbox is ticked)
Name: "{userdesktop}\Decred Poker {#AppVersion}"; Filename: "{app}\dcrpoker.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\dcrpoker.exe"; Description: "Launch Decred Poker {#AppVersion}"; Flags: nowait postinstall skipifsilent
