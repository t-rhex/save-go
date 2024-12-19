# PowerShell installation script for Windows
param (
    [string]$Version = "latest",
    [string]$InstallPath = "$env:USERPROFILE\AppData\Local\save"
)

# Color output functions
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

function Write-Info($message) { Write-ColorOutput Blue "INFO: $message" }
function Write-Success($message) { Write-ColorOutput Green "SUCCESS: $message" }
function Write-Warning($message) { Write-ColorOutput Yellow "WARNING: $message" }
function Write-Error($message) { Write-ColorOutput Red "ERROR: $message"; exit 1 }

# Check prerequisites
function Check-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check Go installation
    if (!(Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Error "Go is not installed. Please install Go first."
    }
    
    # Check Git installation
    if (!(Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Error "Git is not installed. Please install Git first."
    }
    
    # Check Go version
    $goVersion = (go version) -replace 'go version go([0-9]+\.[0-9]+).*', '$1'
    if ([version]$goVersion -lt [version]"1.21") {
        Write-Error "Go version 1.21 or higher is required. Current version: $goVersion"
    }
    
    Write-Success "All prerequisites met!"
}

# Create required directories
function Setup-Directories {
    Write-Info "Setting up directories..."
    
    # Create installation directory
    New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null
    
    # Create completion directory
    New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\Documents\WindowsPowerShell\Completions" | Out-Null
}

# Install save
function Install-Save {
    Write-Info "Installing save version $Version..."
    
    # Create temporary directory
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    
    try {
        # Clone repository
        Write-Info "Cloning repository..."
        git clone --quiet https://github.com/t-rhex/save-go.git $tempDir
        
        Push-Location $tempDir
        
        # Checkout specific version if not main
        if ($Version -ne "main") {
            git checkout --quiet "v$Version"
        }
        
        # Build
        Write-Info "Building..."
        go build -o save.exe
        
        # Install
        Write-Info "Installing..."
        Copy-Item save.exe -Destination "$InstallPath\save.exe" -Force
        
        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$InstallPath*") {
            [Environment]::SetEnvironmentVariable(
                "Path",
                "$userPath;$InstallPath",
                "User"
            )
            Write-Info "Added to PATH"
        }
        
        # Setup PowerShell completion
        $completionScript = & "$InstallPath\save.exe" --generate-completion powershell
        Set-Content -Path "$env:USERPROFILE\Documents\WindowsPowerShell\Completions\save.ps1" -Value $completionScript
        
        Write-Success "Installation complete!"
    }
    finally {
        Pop-Location
        Remove-Item -Recurse -Force $tempDir
    }
}

# Main installation process
function Main {
    Write-ColorOutput Blue "=== Save Command Manager Installer ==="
    
    Check-Prerequisites
    Setup-Directories
    Install-Save
    
    Write-Info "To start using save, either:"
    Write-Info "  1. Restart your terminal"
    Write-Info "  2. Or run: `$env:Path = [Environment]::GetEnvironmentVariable('Path', 'User')"
    Write-Info "`nGet started with: save --help"
}

# Run main installation
if ($MyInvocation.InvocationName -eq "&") {
    # Being sourced, export functions
    Export-ModuleMember -Function *
} else {
    # Being run directly
    Main
} 