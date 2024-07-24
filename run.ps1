$ErrorActionPreference = "Stop"

if (Test-Path .\parental-controls.exe)
{
    Remove-Item .\parental-controls.exe;
}

go build;
.\parental-controls.exe