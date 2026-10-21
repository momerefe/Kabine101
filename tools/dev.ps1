param([ValidateSet('api','web','mail','admin-create','db-up','db-version','seed-demo','generate','check','build')][string]$Task = 'api', [string]$Name, [string]$Email)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
if (Test-Path '.tools/go/bin') { $env:Path = "$PWD/.tools/go/bin;$env:Path" }
$nodeDir = Get-ChildItem '.tools/node-*-win-x64' -Directory -ErrorAction SilentlyContinue | Select-Object -First 1
if ($nodeDir) { $env:Path = "$($nodeDir.FullName);$env:Path" }
if (Test-Path '.tools/pnpm/node_modules/.bin') { $env:Path = "$PWD/.tools/pnpm/node_modules/.bin;$env:Path" }
function Run([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Command failed (exit $LASTEXITCODE)" }
}
switch ($Task) {
    'api' { Run 'go' @('run','./cmd/api') }
    'web' { Run 'pnpm.cmd' @('dev') }
    'mail' {
        if (!(Test-Path '.tools/mailpit/mailpit.exe')) { throw 'Run python tools/install_mailpit.py first.' }
        New-Item -ItemType Directory -Force '.local' | Out-Null
        Run '.tools/mailpit/mailpit.exe' @('--listen','127.0.0.1:8025','--smtp','127.0.0.1:1025','--database','.local/mailpit.db','--disable-version-check','--smtp-disable-rdns','--max','100','--quiet')
    }
    'admin-create' {
        if (!$Name -or !$Email) { throw 'Provide -Name and -Email. Password is requested privately by the Go command.' }
        Run 'go' @('run','./cmd/admin','-name',$Name,'-email',$Email)
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
        Run 'go' @('test','-tags=integration','./internal/migrations','./internal/auth','-count=1')
        Run 'go' @('vet','./...')
        Run 'pnpm.cmd' @('check')
    }
    'build' { Run 'go' @('build','./...'); Run 'pnpm.cmd' @('build') }
}
