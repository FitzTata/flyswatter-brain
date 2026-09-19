$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$unformatted = gofmt -l cmd internal
if ($unformatted) {
    Write-Error "Run gofmt on:`n$($unformatted -join "`n")"
}

go vet ./...
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Push-Location frontend
try {
    npm run lint
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
    npm run typecheck
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}

$python = if (Test-Path '.venv-neural\Scripts\python.exe') {
    '.\.venv-neural\Scripts\python.exe'
}
else {
    'python'
}

& $python -m ruff check neural_worker
exit $LASTEXITCODE
