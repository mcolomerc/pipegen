# Code Quality and Lint Automation

This document describes the multi-layered approach to ensure code quality and prevent lint issues from reaching the main branch.

## 🏗️ Quality Automation Layers

### 1. **Local Development (Pre-commit)**
- **Git Hooks**: Automatic checks before each commit
- **Pre-commit Framework**: Enhanced formatting and validation
- **Make Targets**: Easy quality commands

### 2. **Push Protection (Pre-push)**
- **Git Pre-push Hook**: Comprehensive checks before pushing to main
- **Includes**: Lint, tests, build verification

### 3. **CI/CD Pipeline (GitHub Actions)**
- **Automatic**: Runs on all pushes and pull requests
- **Comprehensive**: Formatting, linting, testing, building
- **Blocking**: Prevents merges if quality gates fail

### 4. **Branch Protection (GitHub)**
- **Enforcement**: Requires CI checks to pass
- **Review Process**: Mandatory code reviews for main branch

## 🚀 Quick Setup

### Initial Setup
```bash
# Run the development environment setup
./scripts/setup-dev.sh

# Or manually:
make setup-hooks          # Set up git hooks
make setup-pre-commit     # Set up pre-commit framework
```

### Daily Usage
```bash
# Before committing (automatic via hooks)
make pre-commit           # Format, lint, test

# Before pushing to main (automatic via hooks)
make pre-push            # Comprehensive quality check

# Manual quality check
make quality             # All quality gates
```

## 🛠️ Available Commands

### Make Targets
| Command | Description |
|---------|-------------|
| `make lint` | Run golangci-lint |
| `make lint-fix` | Run golangci-lint with auto-fixes |
| `make fmt` | Format code with go fmt |
| `make test` | Run all tests |
| `make pre-commit` | Pre-commit checks (fmt + lint + test) |
| `make pre-push` | Pre-push checks (comprehensive) |
| `make quality` | All quality gates |
| `make setup-hooks` | Configure git hooks |

### Git Hooks
- **Pre-commit**: Runs before each commit
  - golangci-lint check
  - go fmt check
  - go mod tidy check
  
- **Pre-push**: Runs before pushing to main
  - Full lint suite
  - All tests
  - Build verification

## 🔒 GitHub Branch Protection

### Setup Branch Protection (Admin Required)
```bash
# Requires GitHub CLI and admin permissions
gh auth login
./scripts/setup-branch-protection.sh
```

### Protection Rules
- ✅ Require status checks to pass (Lint, Test)
- ✅ Require branches to be up to date
- ✅ Require pull request reviews
- ✅ Dismiss stale reviews
- ✅ Include administrators
- ❌ Prevent force pushes
- ❌ Prevent branch deletion

## 🔧 Configuration Files

### `.pre-commit-config.yaml`
Pre-commit framework configuration with multiple hooks for code quality.

### `.githooks/`
- `pre-commit`: Local commit validation
- `pre-push`: Push validation for main branch

### `.github/workflows/ci.yml`
Enhanced CI pipeline with comprehensive quality checks:
- Go formatting validation
- golangci-lint with timeout
- go mod tidy verification
- Test coverage reporting

## 🚨 Troubleshooting

### Lint Issues
```bash
# Auto-fix many issues
make lint-fix

# Manual fix required issues
golangci-lint run
```

### Hook Issues
```bash
# Reinstall hooks
make setup-hooks

# Skip hooks (not recommended)
git commit --no-verify
```

### CI Failures
1. Check local quality: `make quality`
2. Fix issues locally: `make lint-fix`
3. Commit and push fixes

## 📊 Quality Metrics

The automation ensures:
- **Zero tolerance** for lint issues in main branch
- **Consistent formatting** across the codebase  
- **Test coverage** maintenance
- **Clean dependencies** (go mod tidy)
- **Build verification** before deployment

## 🎯 Benefits

1. **Early Detection**: Issues caught at commit time
2. **Consistent Quality**: Same standards for all contributors
3. **Automated Fixes**: Many issues auto-resolved
4. **CI/CD Reliability**: Fewer pipeline failures
5. **Code Review Focus**: Reviews focus on logic, not style
