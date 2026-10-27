; Inno Setup script for the HydroGo CLI. Built by .github/workflows/release.yml:
;   ISCC /DAppVersion=0.2.0 /DArch=amd64 /DSourceDir=... /DOutputDir=... hydrogo.iss
; It installs hydro.exe per machine (or per user without admin rights) and adds
; the install folder to PATH, so `hydro new mysite` works in any new terminal.
#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef Arch
  #define Arch "amd64"
#endif
#ifndef SourceDir
  #define SourceDir "."
#endif
#ifndef OutputDir
  #define OutputDir "."
#endif

[Setup]
AppId={{6F3C0E0A-5B7D-4D6E-9A41-7A8F1C2B9E10}
AppName=HydroGo
AppVersion={#AppVersion}
AppPublisher=HydroGo
AppPublisherURL=https://github.com/ardaozyildirim/hydrogo
DefaultDirName={autopf}\HydroGo
DefaultGroupName=HydroGo
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
OutputDir={#OutputDir}
OutputBaseFilename=HydroGo-Setup-{#AppVersion}-windows-{#Arch}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ChangesEnvironment=yes
LicenseFile={#SourceDir}\LICENSE
#if Arch == "arm64"
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif

[Files]
Source: "{#SourceDir}\hydro.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\config.example.yaml"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Tasks]
Name: addtopath; Description: "Add hydro to PATH (needed to run `hydro` from any terminal)"; GroupDescription: "Command line:"

[Registry]
Root: HKA; Subkey: "{code:EnvKey}"; ValueType: expandsz; ValueName: "Path"; \
  ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NeedsPath(ExpandConstant('{app}'))

[Code]
function EnvKey(Param: String): String;
begin
  if IsAdminInstallMode then
    Result := 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
  else
    Result := 'Environment';
end;

function NeedsPath(Dir: String): Boolean;
var
  Paths: String;
  Root: Integer;
begin
  if IsAdminInstallMode then Root := HKEY_LOCAL_MACHINE else Root := HKEY_CURRENT_USER;
  if not RegQueryStringValue(Root, EnvKey(''), 'Path', Paths) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Uppercase(Dir) + ';', ';' + Uppercase(Paths) + ';') = 0;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Paths: String;
  Root, P: Integer;
  Dir: String;
begin
  if CurUninstallStep <> usPostUninstall then exit;
  if IsAdminInstallMode then Root := HKEY_LOCAL_MACHINE else Root := HKEY_CURRENT_USER;
  if not RegQueryStringValue(Root, EnvKey(''), 'Path', Paths) then exit;
  Dir := ExpandConstant('{app}');
  P := Pos(';' + Uppercase(Dir), ';' + Uppercase(Paths));
  if P > 0 then
  begin
    Delete(Paths, P, Length(Dir) + 1);
    RegWriteExpandStringValue(Root, EnvKey(''), 'Path', Paths);
  end;
end;

[Run]
Filename: "{app}\hydro.exe"; Parameters: "version"; Flags: nowait postinstall skipifsilent runhidden; Description: "Verify the installation"
