#ifndef AppVersion
#define AppVersion "0.0.1"
#endif

#ifndef RepoRoot
#define RepoRoot "C:\pokerbisonrelay"
#endif

[Setup]
AppName=Decred Poker
AppVersion={#AppVersion}
DefaultDirName={pf}\Decred Poker
DefaultGroupName=Decred Poker
SetupIconFile={#RepoRoot}\pokerui\flutterui\pokerui\windows\runner\resources\app_icon.ico
UninstallDisplayIcon={app}\dcrpoker.exe
OutputBaseFilename=dcrpoker-windows-amd64-{#AppVersion}
OutputDir={#RepoRoot}\releases
Compression=lzma
SolidCompression=yes

[InstallDelete]
Type: files; Name: "{app}\app_icon.ico"

[Files]
Source: "{#RepoRoot}\pokerui\flutterui\pokerui\build\windows\x64\runner\Release\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs
Source: "{#RepoRoot}\pokerui\flutterui\pokerui\windows\runner\resources\app_icon.ico"; DestDir: "{app}"

[Tasks]
Name: "desktopicon"; Description: "Create &desktop shortcut"; GroupDescription: "Additional icons:";

[Icons]
Name: "{group}\Decred Poker {#AppVersion}"; Filename: "{app}\dcrpoker.exe"; IconFilename: "{app}\app_icon.ico"
Name: "{group}\Uninstall Decred Poker"; Filename: "{uninstallexe}"

; Desktop (only if the checkbox is ticked)
Name: "{userdesktop}\Decred Poker {#AppVersion}"; Filename: "{app}\dcrpoker.exe"; IconFilename: "{app}\app_icon.ico"; Tasks: desktopicon

[Run]
Filename: "{app}\dcrpoker.exe"; Description: "Launch Decred Poker {#AppVersion}"; Flags: nowait postinstall skipifsilent
