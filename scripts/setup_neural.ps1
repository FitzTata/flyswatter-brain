$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
    [Environment]::GetEnvironmentVariable('Path', 'User')

if (-not (Get-Command uv -ErrorAction SilentlyContinue)) {
    winget install `
        --id astral-sh.uv `
        --exact `
        --scope user `
        --accept-package-agreements `
        --accept-source-agreements `
        --silent
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
        [Environment]::GetEnvironmentVariable('Path', 'User')
}

git submodule update --init --recursive

if (-not (Get-Command c++ -ErrorAction SilentlyContinue)) {
    winget install `
        --id BrechtSanders.WinLibs.POSIX.UCRT `
        --exact `
        --scope user `
        --accept-package-agreements `
        --accept-source-agreements `
        --silent
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
        [Environment]::GetEnvironmentVariable('Path', 'User')
}

if (-not (Test-Path '.venv-neural\Scripts\python.exe')) {
    uv venv .venv-neural --python 3.14
}

$env:UV_PROJECT_ENVIRONMENT = (Resolve-Path '.venv-neural')
uv sync --project neural_worker --frozen
if ($LASTEXITCODE -ne 0) {
    uv sync --project neural_worker
}

$env:PYTHONPATH = (Resolve-Path 'third_party\stonkfly')
$env:STONKFLY_DATA = Join-Path $root '.local\malecns'
.\.venv-neural\Scripts\python.exe -c 'from stonkfly.data import prepare; prepare()'
