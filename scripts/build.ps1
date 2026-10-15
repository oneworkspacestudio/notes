param([string]$Go='go',[switch]$BrowserTests)
$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Push-Location $root
try {
 npm --prefix web ci
 if($LASTEXITCODE){throw 'Dependency installation failed'}
 npm --prefix web run build
 if($LASTEXITCODE){throw 'Frontend build failed'}
 & $Go test -timeout 120s ./...
 if($LASTEXITCODE){throw 'Go tests failed'}
 & $Go vet ./...
 if($LASTEXITCODE){throw 'Go vet failed'}
 New-Item -ItemType Directory -Path dist -Force | Out-Null
 $env:CGO_ENABLED='0';$env:GOOS='windows';$env:GOARCH='amd64'
 & $Go build -trimpath -ldflags '-s -w' -o dist/MyNotes.exe ./cmd/notes
 if($LASTEXITCODE){throw 'Windows build failed'}
 if($BrowserTests){npm --prefix web run test:e2e;if($LASTEXITCODE){throw 'Browser tests failed'}}
 node (Join-Path $PSScriptRoot 'licenses.mjs') $Go
 if($LASTEXITCODE){throw 'License inventory failed'}
 & (Join-Path $PSScriptRoot 'package.ps1')
} finally {Pop-Location}
