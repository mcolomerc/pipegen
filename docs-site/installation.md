# Installation

Get PipeGen up and running on your system in minutes. Choose the installation method that works best for your environment.

## Prerequisites

### Required
- **Docker & Docker Compose** - For local development stack
  - Docker 20.10+ recommended
  - Docker Compose 2.0+ recommended

### Optional
- **Go 1.21+** - Only needed if building from source
- **Git** - For cloning the repository

## Quick Install (Recommended)

The fastest way to get started:

```bash
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash
```

This script:
- ✅ Detects your operating system and architecture
- ✅ Downloads the appropriate binary
- ✅ Installs to `/usr/local/bin/pipegen`
- ✅ Makes the binary executable
- ✅ Verifies the installation

### Manual Quick Install

If you prefer to inspect the script first:

```bash
# Download and inspect the install script
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh -o install.sh
cat install.sh

# Run the installer
bash install.sh
```

## Alternative Installation Methods

### Go Install

If you have Go installed:

```bash
go install github.com/mcolomerc/pipegen@latest
```

This installs the latest version to your `$GOPATH/bin` directory.

### Manual Binary Download

1. **Download from GitHub Releases**

   Visit the [releases page](https://github.com/mcolomerc/pipegen/releases) and download the appropriate binary for your platform:

   - **Linux (AMD64)**: `pipegen-linux-amd64.tar.gz`
   - **Linux (ARM64)**: `pipegen-linux-arm64.tar.gz`
   - **macOS (Intel)**: `pipegen-darwin-amd64.tar.gz`
   - **macOS (Apple Silicon)**: `pipegen-darwin-arm64.tar.gz`
   - **Windows (AMD64)**: `pipegen-windows-amd64.zip`

2. **Extract and Install**

   ```bash
   # Example for Linux AMD64
   tar -xzf pipegen-linux-amd64.tar.gz
   sudo mv pipegen /usr/local/bin/
   sudo chmod +x /usr/local/bin/pipegen
   ```

3. **Verify Installation**

   ```bash
   pipegen --version
   ```

### Build from Source

For the latest development version or custom builds:

```bash
# Clone the repository
git clone https://github.com/mcolomerc/pipegen.git
cd pipegen

# Build the binary
go build -o pipegen ./cmd/main.go

# Install to system PATH
sudo mv pipegen /usr/local/bin/
```

### Docker Installation

Run PipeGen in a container:

```bash
# Pull the Docker image
docker pull mcolomerc/pipegen:latest

# Create an alias for convenience
echo 'alias pipegen="docker run --rm -v \$(pwd):/workspace -w /workspace --net host mcolomerc/pipegen:latest"' >> ~/.bashrc
source ~/.bashrc

# Verify installation
pipegen --version
```

## Platform-Specific Instructions

### macOS

**Using Homebrew (if available):**
```bash
# Add the tap (if configured)
brew tap mcolomerc/pipegen
brew install pipegen
```

**Manual Installation:**
```bash
# Download and install
curl -L -o pipegen-darwin-arm64.tar.gz \
  https://github.com/mcolomerc/pipegen/releases/latest/download/pipegen-darwin-arm64.tar.gz
tar -xzf pipegen-darwin-arm64.tar.gz
sudo mv pipegen /usr/local/bin/
```

### Linux

**Ubuntu/Debian:**
```bash
# Quick install
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash

# Or manual install
wget https://github.com/mcolomerc/pipegen/releases/latest/download/pipegen-linux-amd64.tar.gz
tar -xzf pipegen-linux-amd64.tar.gz
sudo mv pipegen /usr/local/bin/
```

**CentOS/RHEL/Fedora:**
```bash
# Quick install
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash

# Or using dnf/yum (if configured)
sudo dnf install pipegen  # Fedora
sudo yum install pipegen  # CentOS/RHEL
```

**Arch Linux:**
```bash
# Using AUR (if available)
yay -S pipegen

# Or manual install
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash
```

### Windows

**PowerShell Installation:**
```powershell
# Download the latest release
Invoke-WebRequest -Uri "https://github.com/mcolomerc/pipegen/releases/latest/download/pipegen-windows-amd64.zip" -OutFile "pipegen.zip"

# Extract and install
Expand-Archive -Path "pipegen.zip" -DestinationPath "C:\Program Files\pipegen\"

# Add to PATH (run as Administrator)
$env:Path += ";C:\Program Files\pipegen"
[Environment]::SetEnvironmentVariable("Path", $env:Path, [EnvironmentVariableTarget]::Machine)
```

**Using Chocolatey (if available):**
```powershell
choco install pipegen
```

**Windows Subsystem for Linux (WSL):**
```bash
# Use the Linux installation method within WSL
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash
```

## Post-Installation Setup

### Verify Installation

```bash
# Check version
pipegen --version

# Show help
pipegen --help

# Check AI provider setup
pipegen check
```

**Expected Output:**
```
PipeGen v1.0.0
Build: a4c8079 (2025-08-26)
Go version: go1.21.0
```

### Docker Setup

Ensure Docker is running and accessible:

```bash
# Check Docker installation
docker --version
docker-compose --version

# Test Docker access (should not require sudo)
docker ps

# If you get permission errors on Linux:
sudo usermod -aG docker $USER
# Log out and log back in
```

### Configuration

Create a global configuration file:

```bash
# Create config directory
mkdir -p ~/.config/pipegen

# Create basic configuration
cat > ~/.pipegen.yaml << EOF
# PipeGen Global Configuration
bootstrap_servers: "localhost:9093"
flink_url: "http://localhost:8081"
schema_registry_url: "http://localhost:8082"
local_mode: true

# Default pipeline settings
default_message_rate: 100
default_duration: "5m"
topic_prefix: "pipegen"
cleanup_on_exit: true
EOF
```

### AI Provider Setup (Optional)

**For Ollama (Local AI):**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Download a model
ollama pull llama3.1

# Configure PipeGen
echo 'export PIPEGEN_OLLAMA_MODEL="llama3.1"' >> ~/.bashrc
echo 'export PIPEGEN_OLLAMA_URL="http://localhost:11434"' >> ~/.bashrc
source ~/.bashrc
```

**For OpenAI:**
```bash
# Set up API key
echo 'export PIPEGEN_OPENAI_API_KEY="your-api-key"' >> ~/.bashrc
echo 'export PIPEGEN_LLM_MODEL="gpt-4"' >> ~/.bashrc
source ~/.bashrc
```

## Troubleshooting

### Common Installation Issues

**Permission denied errors:**
```bash
# Linux/macOS - use sudo for system installation
sudo mv pipegen /usr/local/bin/
sudo chmod +x /usr/local/bin/pipegen

# Or install to user directory
mkdir -p ~/bin
mv pipegen ~/bin/
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**Binary not found after installation:**
```bash
# Check if installation directory is in PATH
echo $PATH

# Add to PATH if needed
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**Docker permission issues (Linux):**
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Apply group membership (logout/login required)
newgrp docker

# Test Docker access
docker ps
```

**Go installation issues:**
```bash
# Check Go version
go version

# If Go is too old, update:
# Visit https://golang.org/dl/ and install latest version
```

### Verification Tests

Run these commands to ensure everything is working:

```bash
# Test basic functionality
pipegen --help

# Test Docker integration
pipegen deploy --dry-run

# Test AI integration (if configured)
pipegen check

# Create a test project
pipegen init test-installation
cd test-installation
pipegen validate
```

### Platform-Specific Issues

**macOS Security (Gatekeeper):**
```bash
# If you get "cannot be opened because it is from an unidentified developer"
sudo spctl --master-disable
# Or right-click the binary and select "Open"
```

**Windows Antivirus:**
- Some antivirus software may flag the binary as suspicious
- Add `pipegen.exe` to your antivirus whitelist
- Download from official GitHub releases only

**Linux SELinux:**
```bash
# If SELinux blocks execution
sudo semanage fcontext -a -t bin_t "/usr/local/bin/pipegen"
sudo restorecon -v /usr/local/bin/pipegen
```

## Updates

### Automatic Updates

PipeGen will notify you of new versions:

```bash
🆕 A new version of PipeGen is available!
   Current: v1.0.0
   Latest:  v1.1.0
   
   Run: curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash
```

### Manual Updates

```bash
# Re-run the installation script
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash

# Or download the latest release manually
# Follow the same installation steps with the new binary
```

### Check for Updates

```bash
# Check current version
pipegen --version

# Check latest release
curl -s https://api.github.com/repos/mcolomerc/pipegen/releases/latest | jq -r .tag_name
```

## Uninstallation

To remove PipeGen from your system:

```bash
# Remove binary
sudo rm /usr/local/bin/pipegen

# Remove configuration (optional)
rm -rf ~/.pipegen.yaml ~/.config/pipegen

# Remove environment variables
# Edit ~/.bashrc and remove PIPEGEN_* exports
```

## Next Steps

Now that PipeGen is installed:

1. **[Getting Started](./getting-started)** - Create your first pipeline
2. **[Commands](./commands)** - Learn available commands  
3. **[Examples](./examples)** - See PipeGen in action
4. **[Configuration](./configuration)** - Customize your setup

Need help? Check our [troubleshooting guide](./advanced/troubleshooting) or [open an issue](https://github.com/mcolomerc/pipegen/issues).
