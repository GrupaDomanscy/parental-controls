$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true

go generate ./components
go test -c -o tmp\test.exe
