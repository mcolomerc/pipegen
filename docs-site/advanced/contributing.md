# Contributing to PipeGen

We welcome contributions to PipeGen! This guide will help you get started with contributing to the project.

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

- **Go 1.21+** - Required for building and testing PipeGen
- **Docker & Docker Compose** - For running the local development stack
- **Git** - For version control

### Development Setup

1. **Fork and Clone the Repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/pipegen.git
   cd pipegen
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Build the Project**
   ```bash
   make build
   # Or manually:
   go build -o bin/pipegen ./main.go
   ```

4. **Run Tests**
   ```bash
   make test
   # Or manually:
   go test ./...
   ```

## Development Workflow

### 1. Create a Feature Branch

Always create a new branch for your changes:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Your Changes

- Follow the existing code style and conventions
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass

### 3. Test Your Changes

```bash
# Run unit tests
go test ./...

# Run integration tests (if applicable)
make test-integration

# Test the CLI manually
./bin/pipegen --help
```

### 4. Submit a Pull Request

1. Push your changes to your fork
2. Create a pull request against the `main` branch
3. Provide a clear description of your changes
4. Reference any related issues

## Code Style Guidelines

### Go Code Standards

- Follow standard Go conventions (`gofmt`, `golint`)
- Use meaningful variable and function names
- Add comments for public functions and complex logic
- Keep functions small and focused
- Use proper error handling

### Project Structure

```
pipegen/
├── cmd/              # CLI commands
├── internal/         # Internal packages
│   ├── dashboard/    # Web dashboard components
│   ├── docker/       # Docker deployment logic
│   ├── generator/    # Pipeline generation
│   ├── llm/          # AI/LLM integration
│   ├── pipeline/     # Core pipeline logic
│   ├── templates/    # Template management
│   └── types/        # Type definitions
├── docs-site/        # Documentation source
├── web/              # Web assets
└── main.go          # Entry point
```

## Testing

### Unit Tests

- Write unit tests for all new functions
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for good test coverage

Example test structure:
```go
func TestYourFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

- Test end-to-end functionality
- Use Docker containers for external dependencies
- Clean up resources after tests

## Documentation

### Code Documentation

- Add GoDoc comments for public APIs
- Include examples in documentation
- Document complex algorithms and business logic

### User Documentation

Documentation is located in `docs-site/` and built with VitePress:

```bash
cd docs-site
npm install
npm run dev    # Start development server
npm run build  # Build for production
```

### Adding New Documentation

1. Create or edit Markdown files in `docs-site/`
2. Update navigation in `docs-site/.vitepress/config.js` if needed
3. Test locally with `npm run dev`
4. Include documentation updates in your pull request

## Issue Reporting

### Bug Reports

When reporting bugs, please include:

- PipeGen version (`pipegen version`)
- Operating system and version
- Go version
- Steps to reproduce the issue
- Expected vs actual behavior
- Relevant logs or error messages

### Feature Requests

For feature requests:

- Describe the problem you're trying to solve
- Explain your proposed solution
- Consider alternative approaches
- Provide use cases and examples

## Release Process

Releases are managed by maintainers:

1. Version bumping follows semantic versioning
2. Releases are tagged and published automatically
3. Binaries are built for multiple platforms
4. Documentation is updated automatically

## Community Guidelines

### Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Maintain a professional environment

### Communication

- **GitHub Issues** - Bug reports, feature requests
- **GitHub Discussions** - Questions, ideas, general discussion
- **Pull Requests** - Code contributions and reviews

## Getting Help

If you need help with contributing:

1. Check existing [documentation](https://mcolomerc.github.io/pipegen/)
2. Search [existing issues](https://github.com/mcolomerc/pipegen/issues)
3. Create a new issue with the `question` label
4. Join discussions in GitHub Discussions

## Maintainer Information

### Code Review Process

- All changes require at least one approval
- Maintainers will review pull requests promptly
- Feedback should be addressed before merging
- Keep pull requests focused and atomic

### Release Schedule

- No fixed schedule - releases based on feature completion
- Critical bug fixes may trigger immediate releases
- Feature releases include comprehensive testing

## Development Tips

### Local Testing with Real Data

```bash
# Create a test pipeline
pipegen init test-pipeline

# Deploy local stack
pipegen deploy

# Run with monitoring
pipegen run --message-rate 10 --duration 1m --dashboard
```

### Debugging

- Use `--verbose` flag for detailed logging
- Check Docker logs: `docker-compose logs -f`
- Monitor dashboard at `http://localhost:8080`

### Performance Testing

```bash
# Test with high throughput
pipegen run --message-rate 1000 --duration 5m --generate-report

# Test with traffic patterns
pipegen run --traffic-pattern "1m-2m:200%,3m-4m:500%" --duration 5m
```

## Thank You!

Thank you for contributing to PipeGen! Your contributions help make streaming data pipelines more accessible and powerful for everyone.

---

For questions about contributing, please open an issue or start a discussion in the [PipeGen repository](https://github.com/mcolomerc/pipegen).
