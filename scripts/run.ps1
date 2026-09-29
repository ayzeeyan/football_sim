param(
    [switch]$Dev
)

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

# Frontend is Bun-only (bun install, bun run dev, bun run build). Never npm.
if ($Dev) {
    Write-Host "Start the Go API in another terminal:"
    Write-Host "  go -C backend_go run ./cmd/server"
    Write-Host ""
    try {
        Set-Location (Join-Path $Root "frontend")
        if (-not (Test-Path "node_modules")) {
            bun install
        }
        bun run dev
    }
    finally {
        # Ctrl+C during `bun run dev` must not strand the shell in frontend/.
        Set-Location $Root
    }
    exit 0
}

$dist = Join-Path $Root "frontend\dist\index.html"
if (-not (Test-Path $dist)) {
    Write-Host "Building the React client..."
    try {
        Set-Location (Join-Path $Root "frontend")
        if (-not (Test-Path "node_modules")) {
            bun install
        }
        bun run build
    }
    finally {
        Set-Location $Root
    }
}

$dataset = Join-Path $Root "dataset.json"
$static = Join-Path $Root "frontend\dist"
$save = Join-Path $Root "saves\career.json"
# `go -C` enters the module for the go command only: the shell's working
# directory never changes, so Ctrl+C leaves you exactly where you were.
go -C backend_go run ./cmd/server -dataset $dataset -static $static -save $save
