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

[Icons]
Name: "{group}\Bison Poker"; Filename: "{app}\bisonpoker-{#AppVersion}.exe"
Name: "{group}\Uninstall Bison Poker"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\bisonpoker.exe"; Description: "Launch Bison Poker"; Flags: nowait postinstall skipifsilent
