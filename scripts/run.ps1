param(
    [switch]$Dev
)

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

# Frontend is Bun-only (bun install, bun run dev, bun run build). Never npm.
if ($Dev) {
    Write-Host "Start the Go API in another terminal:"
    Write-Host "  cd backend_go"
    Write-Host "  go run ./cmd/server"
    Write-Host ""
    Set-Location (Join-Path $Root "frontend")
    if (-not (Test-Path "node_modules")) {
        bun install
    }
    bun run dev
    exit 0
}

$dist = Join-Path $Root "frontend\dist\index.html"
if (-not (Test-Path $dist)) {
    Write-Host "Building the React client..."
    Set-Location (Join-Path $Root "frontend")
    if (-not (Test-Path "node_modules")) {
        bun install
    }
    bun run build
    Set-Location $Root
}

$dataset = Join-Path $Root "dataset.json"
$static = Join-Path $Root "frontend\dist"
$save = Join-Path $Root "saves\career.json"
Set-Location (Join-Path $Root "backend_go")
go run ./cmd/server -dataset $dataset -static $static -save $save
