param([ValidateSet('api','web','mail','admin-create','db-up','db-version','seed-demo','generate','check','build')][string]$Task = 'api', [string]$Name, [string]$Email, [string]$Username, [string]$LanAddress)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
if (Test-Path '.tools/go/bin') { $env:Path = "$PWD/.tools/go/bin;$env:Path" }
$nodeDir = Get-ChildItem '.tools/node-*-win-x64' -Directory -ErrorAction SilentlyContinue | Select-Object -First 1
if ($nodeDir) { $env:Path = "$($nodeDir.FullName);$env:Path" }
if (Test-Path '.tools/pnpm/node_modules/.bin') { $env:Path = "$PWD/.tools/pnpm/node_modules/.bin;$env:Path" }
if ($LanAddress) {
    if ($Task -notin @('api','web')) { throw '-LanAddress is only supported for api and web.' }
    $parsedAddress = $null
    if (![System.Net.IPAddress]::TryParse($LanAddress, [ref]$parsedAddress) -or $parsedAddress.AddressFamily -ne [System.Net.Sockets.AddressFamily]::InterNetwork -or [System.Net.IPAddress]::IsLoopback($parsedAddress)) {
        throw 'Provide the LAN IPv4 address of this computer, for example -LanAddress 192.168.1.94.'
    }
    $LanAddress = $parsedAddress.ToString()
    if (!(Get-NetIPAddress -AddressFamily IPv4 -IPAddress $LanAddress -ErrorAction SilentlyContinue | Where-Object { $_.AddressState -eq 'Preferred' })) {
        throw 'The LAN address is not assigned to an active adapter on this computer. Check ipconfig.'
    }
    Write-Host "Use http://${LanAddress}:5173 on both the computer and phone."
}
function Run([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Command failed (exit $LASTEXITCODE)" }
}
switch ($Task) {
    'api' {
        if ($LanAddress) { $env:APP_ORIGIN = "http://${LanAddress}:5173" }
        Run 'go' @('run','./cmd/api')
    }
    'web' {
        if ($LanAddress) { Run 'pnpm.cmd' @('exec','vite','--host',$LanAddress) }
        else { Run 'pnpm.cmd' @('dev') }
    }
    'mail' {
        if (!(Test-Path '.tools/mailpit/mailpit.exe')) { throw 'Run python tools/install_mailpit.py first.' }
        New-Item -ItemType Directory -Force '.local' | Out-Null
        Run '.tools/mailpit/mailpit.exe' @('--listen','127.0.0.1:8025','--smtp','127.0.0.1:1025','--database','.local/mailpit.db','--disable-version-check','--smtp-disable-rdns','--max','100','--quiet')
    }
    'admin-create' {
        if (!$Name -or (!$Email -and !$Username) -or ($Email -and $Username)) { throw 'Provide -Name and either -Email or -Username. Password is requested privately.' }
        if ($Username) { Run 'go' @('run','./cmd/admin','-name',$Name,'-username',$Username) }
        else { Run 'go' @('run','./cmd/admin','-name',$Name,'-email',$Email) }
    }
    'db-up' { Run 'go' @('run','./cmd/db','up') }
    'db-version' { Run 'go' @('run','./cmd/db','version') }
    'seed-demo' { Run 'go' @('run','./cmd/db','seed-demo') }
    'generate' {
        Run 'go' @('tool','oapi-codegen','--config','api/codegen.yaml','api/openapi.yaml')
        Run 'go' @('tool','sqlc','generate')
        Run 'pnpm.cmd' @('generate:api')
    }
    'check' {
        Run 'go' @('test','./...')
        Run 'go' @('test','-tags=integration','./internal/migrations','./internal/auth','./internal/management','-count=1')
        Run 'go' @('vet','./...')
        Run 'pnpm.cmd' @('check')
    }
    'build' { Run 'go' @('build','./...'); Run 'pnpm.cmd' @('build') }
}
