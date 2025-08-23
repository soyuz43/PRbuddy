$ErrorActionPreference = 'Stop'

$projectPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$binaryName = "prbuddy-go.exe"
$targetBinDir = "$env:USERPROFILE\bin"
$targetBinary = Join-Path $targetBinDir $binaryName

# Ensure target directory exists
if (-not (Test-Path $targetBinDir)) {
    Write-Host "📁 Creating bin directory at $targetBinDir"
    New-Item -ItemType Directory -Path $targetBinDir | Out-Null
}

# Delete old binary if it exists
if (Test-Path $targetBinary) {
    Write-Host "🧹 Removing old $binaryName from $targetBinDir"
    Remove-Item $targetBinary -Force
}

# Build the Go binary
Write-Host "🚧 Building $binaryName from: $projectPath"
go build -o $targetBinary "$projectPath"

# Check for success
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ $binaryName installed to $targetBinDir"
} else {
    Write-Error "❌ Build failed. Check compilation errors."
    exit 1
}

# Suggest adding to PATH if not already there
if (-not ($env:PATH -split ";" | Where-Object { $_ -eq $targetBinDir })) {
    Write-Host "ℹ️  Note: $targetBinDir is not in your PATH."
    Write-Host "   You may want to add it so you can run `prbuddy-go` from anywhere."
}
