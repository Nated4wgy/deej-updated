param([string]$OutputDirectory = (Join-Path $PSScriptRoot '..\bin'), [string]$Version = 'development')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
Push-Location $repoRoot
try {
 $env:CGO_ENABLED = '0'
 $env:GOARCH = 'amd64'
 $env:GOOS = 'windows'
 $OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
 New-Item -ItemType Directory -Force $OutputDirectory | Out-Null
 function Invoke-Go { & go @args; if ($LASTEXITCODE -ne 0) { throw "go failed: $args" } }
 Invoke-Go install github.com/akavel/rsrc@v0.10.2
 $goPath = (& go env GOPATH).Trim()
 $resourceTool = Join-Path $goPath 'bin\rsrc.exe'
 foreach ($package in @('pkg/deej/cmd')) {
  & $resourceTool -arch amd64 -manifest pkg/deej/assets/deej.manifest -ico pkg/deej/assets/logo.ico -o "$package/rsrc_windows.syso"
  if ($LASTEXITCODE -ne 0) { throw 'Resource generation failed' }
 }
 Invoke-Go mod verify
 Invoke-Go test ./...
 Invoke-Go vet ./...
 $commit = (& git rev-parse HEAD).Trim()
 if ($LASTEXITCODE -ne 0) { throw 'Cannot determine source commit' }
 $flags = "-H=windowsgui -X main.gitCommit=$commit -X main.versionTag=$Version -X main.buildType=release"
 Invoke-Go build -mod=readonly -trimpath -ldflags $flags -o (Join-Path $OutputDirectory 'deej.exe') ./pkg/deej/cmd
 # Never overwrite a user's existing configuration when rebuilding into the same folder.
 if (-not (Test-Path (Join-Path $OutputDirectory 'config.yaml'))) { Copy-Item pkg/deej/scripts/misc/default-config.yaml (Join-Path $OutputDirectory 'config.yaml') }
 Copy-Item LICENSE,README.md,THIRD-PARTY-NOTICES.txt $OutputDirectory
 "Version: $Version
Commit: $commit
$(& go version)
Architecture: windows/amd64
" | Set-Content (Join-Path $OutputDirectory 'BUILD-INFO.txt') -Encoding utf8
 Get-Item (Join-Path $OutputDirectory 'deej.exe') | ForEach-Object { $hash = Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256; "$($hash.Hash.ToLower())  $($_.Name)" } | Set-Content (Join-Path $OutputDirectory 'SHA256SUMS.txt') -Encoding ascii
 Write-Host "Built $OutputDirectory"
} finally { Pop-Location }
