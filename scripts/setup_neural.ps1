$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

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
    python -m venv .venv-neural
}

.\.venv-neural\Scripts\python.exe -m pip install --upgrade pip
.\.venv-neural\Scripts\python.exe -m pip install -r neural_worker\requirements.txt

$env:PYTHONPATH = (Resolve-Path 'third_party\stonkfly')
$env:STONKFLY_DATA = Join-Path $root '.local\malecns'
.\.venv-neural\Scripts\python.exe -c 'from stonkfly.data import prepare; prepare()'
