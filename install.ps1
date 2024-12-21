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

# Setup shell environment
function Setup-ShellEnvironment {
    Write-Info "Setting up shell environment..."
    
    # Set install path based on installation type
    if ($InstallType -eq "system") {
        if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
            Write-Error "System-wide installation requires Administrator privileges. Please run as Administrator or use user installation."
            exit 1
        }
        $InstallPath = "C:\Program Files\save"
    } else {
        # User installation path
        $InstallPath = Join-Path $env:USERPROFILE "bin"
    }
    
    # Create bin directory if it doesn't exist
    if (-not (Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null
    }
    
    # Get the current user's PATH from the registry
    if ($InstallType -eq "system") {
        $RegPath = 'Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment'
    } else {
        $RegPath = 'Registry::HKEY_CURRENT_USER\Environment'
    }
    $CurrentPath = (Get-ItemProperty -Path $RegPath -Name PATH).Path
    
    # Check if our install path is already in PATH
    if ($CurrentPath -notlike "*$InstallPath*") {
        Write-Info "Adding $InstallPath to user PATH..."
        
        # Add the new path
        $NewPath = "$CurrentPath;$InstallPath"
        
        # Update the registry
        Set-ItemProperty -Path $RegPath -Name PATH -Value $NewPath
        
        # Update current session's PATH
        $env:Path = "$env:Path;$InstallPath"
        
        Write-Success "Updated PATH environment variable"
        
        # Notify user about PATH update
        Write-Info "PATH has been updated. Changes will take effect in new PowerShell windows."
        Write-Info "To use 'save' in current window, run:"
        Write-Host "    `$env:Path = [System.Environment]::GetEnvironmentVariable('Path', 'User')"
    } else {
        Write-Info "PATH already configured with $InstallPath"
    }
    
    # Setup PowerShell profile for autocompletion
    $ProfileDir = Split-Path $PROFILE
    $CompletionPath = Join-Path $ProfileDir "Completions"
    
    if (-not (Test-Path $CompletionPath)) {
        New-Item -ItemType Directory -Force -Path $CompletionPath | Out-Null
    }
    
    # Add completion script to profile if not already present
    $CompletionScript = & "$InstallPath\save.exe" --generate-completion powershell
    $CompletionFile = Join-Path $CompletionPath "save.ps1"
    Set-Content -Path $CompletionFile -Value $CompletionScript
    
    # Add to PowerShell profile
    if (-not (Test-Path $PROFILE)) {
        New-Item -ItemType File -Force -Path $PROFILE | Out-Null
    }
    
    $ProfileContent = @"

# Added by save installer
Import-Module "$CompletionFile"
"@
    
    if (-not (Get-Content $PROFILE -Raw -ErrorAction SilentlyContinue) -like "*Added by save installer*") {
        Add-Content -Path $PROFILE -Value $ProfileContent
        Write-Success "Added completion to PowerShell profile"
    }
}

# Install save
function Install-Save {
    param (
        [string]$Version = $script:Version
    )
    
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
            Write-Info "Checking out version v$Version..."
            git checkout --quiet "v$Version"
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to checkout version v$Version"
                return
            }
        }
        
        # Build with correct version
        Write-Info "Building..."
        $buildCmd = "go build -ldflags `"-X main.Version=$Version`" -o save.exe"
        Invoke-Expression $buildCmd
        
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Build failed"
            return
        }
        
        # Install
        Write-Info "Installing..."
        Copy-Item save.exe -Destination "$InstallPath\save.exe" -Force
        
        Write-Success "Installation complete!"
    }
    catch {
        Write-Error "Installation failed: $_"
    }
    finally {
        Pop-Location
        Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
    }
}

# Get latest version from GitHub API
function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/t-rhex/save-go/releases/latest"
        $version = $response.tag_name
        return $version.TrimStart('v')  # Remove 'v' prefix if present
    }
    catch {
        Write-Warning "Could not fetch latest version from GitHub: $_"
        return "0.1.0"  # Fallback version
    }
}

# Main installation process
function Main {
    Write-ColorOutput Blue "=== Save Command Manager Installer ==="
    
    # Get latest version first
    if ($Version -eq "latest") {
        $script:Version = Get-LatestVersion
        Write-Info "Latest version is: $script:Version"
    } else {
        $script:Version = $Version
    }
    
    Check-Prerequisites
    Setup-Directories
    Setup-ShellEnvironment
    Install-Save -Version $script:Version
    
    Write-Success "`nSave Command Manager has been installed successfully!"
    Write-Info "`nTo start using save, either:"
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