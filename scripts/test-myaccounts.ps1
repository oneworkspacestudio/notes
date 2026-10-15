param([Parameter(Mandatory=$true)][string]$MyAccountsExe,[string]$NotesExe=(Join-Path $PSScriptRoot '../dist/MyNotes.exe'))
$ErrorActionPreference='Stop'
$fixture=Join-Path ([IO.Path]::GetTempPath()) ('notes-oidc-'+[Guid]::NewGuid().ToString('N'))
function FreePort { $listener=[Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0);$listener.Start();$port=$listener.LocalEndpoint.Port;$listener.Stop();return $port }
$accountsPort=FreePort;$notesPort=FreePort
$issuer="http://localhost:$accountsPort";$origin="http://127.0.0.1:$notesPort"
$password=[Guid]::NewGuid().ToString('N')+[Guid]::NewGuid().ToString('N')
New-Item -ItemType Directory -Path (Join-Path $fixture 'accounts/secrets'),(Join-Path $fixture 'notes') -Force | Out-Null
$config=@{version=1;issuer=$issuer;port=$accountsPort;adminEmail='admin@example.com';adminName='Notes Fixture';tunnelMode='local';sharedEmail=$false}
$config | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $fixture 'accounts/config.json') -Encoding utf8
Add-Type -AssemblyName System.Security.Cryptography.ProtectedData
$bootstrap=[Text.Encoding]::UTF8.GetBytes((@{name='Notes Fixture';email='admin@example.com';password=$password}|ConvertTo-Json))
$protected=[Security.Cryptography.ProtectedData]::Protect($bootstrap,$null,[Security.Cryptography.DataProtectionScope]::LocalMachine)
[IO.File]::WriteAllBytes((Join-Path $fixture 'accounts/secrets/bootstrap.dpapi'),$protected)
$process=$null
try {
 $process=Start-Process -FilePath (Resolve-Path -LiteralPath $MyAccountsExe).Path -ArgumentList @('-data',('"'+(Join-Path $fixture 'accounts')+'"'),'-no-browser') -WindowStyle Hidden -PassThru
 $ready=$false
 for($n=0;$n -lt 100;$n++){if($process.HasExited){throw 'Disposable MyAccounts exited'};try{if((Invoke-RestMethod "$issuer/healthz").status -eq 'ok'){$ready=$true;break}}catch{};Start-Sleep -Milliseconds 300}
 if(!$ready){throw 'Disposable MyAccounts did not start'}
 $env:NOTES_TEST_ISSUER=$issuer;$env:NOTES_TEST_ORIGIN=$origin;$env:NOTES_TEST_PASSWORD=$password;$env:NOTES_TEST_DATA=(Join-Path $fixture 'notes');$env:NOTES_TEST_EXE=(Resolve-Path -LiteralPath $NotesExe).Path
 Push-Location (Join-Path $PSScriptRoot '../web')
 try {node scripts/test-myaccounts.mjs;if($LASTEXITCODE){throw 'Real MyAccounts integration failed'}}finally{Pop-Location}
 Write-Output 'PASS: real MyAccounts discovery, code/PKCE callback, signed identity, stable subject mapping and private-note isolation'
} finally {
 if($process -and !$process.HasExited){& $MyAccountsExe -data (Join-Path $fixture 'accounts') stop | Out-Null;if(!$process.WaitForExit(10000)){Stop-Process -Id $process.Id -Force}}
 foreach($name in @('NOTES_TEST_ISSUER','NOTES_TEST_ORIGIN','NOTES_TEST_PASSWORD','NOTES_TEST_DATA','NOTES_TEST_EXE')){Remove-Item ('Env:'+ $name) -ErrorAction SilentlyContinue}
 $password=$null;$bootstrap=$null;$protected=$null
}
