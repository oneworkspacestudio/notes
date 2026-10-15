$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$exe=Join-Path $root 'dist/MyNotes.exe'
$version=(& $exe version).Trim()
if($LASTEXITCODE -or $version -notmatch '^[0-9A-Za-z.\-]+$'){throw 'Invalid executable version'}
$hash=(Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLowerInvariant();$size=(Get-Item -LiteralPath $exe).Length
$manifest=@{schema=1;package=@{file='MyNotes.exe';size=$size;sha256=$hash};app=@{id='notes';name='notes';description='Private notes, notebooks and ideas';version=$version;kind='process';port=8795;route='/notes';stripPrefix=$true;health='/healthz';depends=@('accounts');oidc=$true;callback='/notes/auth/callback';args=@('--port','${PORT}');env=@{NOTES_DATA_DIR='${DATA}/notes';NOTES_PUBLIC_URL='${ORIGIN}/notes'};source=('https://github.com/oneworkspacestudio/notes/releases/tag/v'+$version)}}
$manifest|ConvertTo-Json -Depth 10|Set-Content -LiteralPath (Join-Path $root 'dist/notes.workspace.json') -Encoding utf8
$archive=Join-Path $root "dist/notes-$version-windows-amd64.zip"
Compress-Archive -LiteralPath @($exe,(Join-Path $root 'config.example.json'),(Join-Path $root 'Start-local.cmd'),(Join-Path $root 'README.md'),(Join-Path $root 'LICENSE'),(Join-Path $root 'NOTICE'),(Join-Path $root 'SECURITY.md'),(Join-Path $root 'docs'),(Join-Path $root 'dist/THIRD-PARTY-LICENSES.txt')) -DestinationPath $archive -Force
$checksums=@($exe,$archive,(Join-Path $root 'dist/notes.workspace.json')) | ForEach-Object {((Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant())+'  '+[IO.Path]::GetFileName($_)}
$checksums|Set-Content -LiteralPath (Join-Path $root 'dist/SHA256SUMS.txt') -Encoding ascii
Write-Output "Packaged notes $version"
