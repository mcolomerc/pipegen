#!/bin/bash
# Development environment setup script

set -e

echo "🚀 Setting up PipeGen development environment..."

# Install required tools
echo "📦 Installing development tools..."

# Install golangci-lint if not present
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    echo "✓ golangci-lint installed"
fi

# Install goimports if not present
if ! command -v goimports &> /dev/null; then
    echo "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
    echo "✓ goimports installed"
fi

# Install pre-commit if available
if command -v pip &> /dev/null || command -v pip3 &> /dev/null; then
    echo "Installing pre-commit framework..."
    if command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    else
        pip install pre-commit
    fi
    echo "✓ pre-commit framework installed"
else
    echo "⚠️  pip not found - skipping pre-commit framework installation"
    echo "   You can install manually: pip install pre-commit"
fi

# Set up git hooks
echo "🔧 Setting up git hooks..."
chmod +x .githooks/pre-commit .githooks/pre-push
git config core.hooksPath .githooks
echo "✓ Git hooks configured"

# Set up pre-commit hooks if framework is available
if command -v pre-commit &> /dev/null; then
    echo "Setting up pre-commit hooks..."
    pre-commit install
    echo "✓ pre-commit hooks installed"
fi

# Run initial quality checks
echo "🔍 Running initial quality checks..."
make quality

echo ""
echo "🎉 Development environment setup complete!"
echo ""
echo "📋 Available make targets:"
echo "   make lint        - Run linters"
echo "   make lint-fix    - Run linters with auto-fixes"
echo "   make test        - Run tests"
echo "   make pre-commit  - Run pre-commit checks"
echo "   make pre-push    - Run comprehensive checks"
echo "   make quality     - Run all quality gates"
echo ""
echo "🔒 To set up GitHub branch protection (requires admin access):"
echo "   ./scripts/setup-branch-protection.sh"
