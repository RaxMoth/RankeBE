# Contributing to REST API Template

First off, thank you for considering contributing to this project! 🎉

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples**
- **Describe the behavior you observed**
- **Explain which behavior you expected to see instead**
- **Include logs and error messages**
- **Include your environment details** (OS, Go version, etc.)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, include:

- **Use a clear and descriptive title**
- **Provide a detailed description of the suggested enhancement**
- **Explain why this enhancement would be useful**
- **List some examples of how it would be used**

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows the existing code style
6. Write a clear commit message

## Development Setup

1. Clone your fork:
```bash
git clone https://github.com/your-username/gin-rest-template.git
cd gin-rest-template
```

2. Install dependencies:
```bash
make install
```

3. Set up your environment:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run tests:
```bash
make test
```

## Coding Guidelines

### Go Style Guide

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` to format your code
- Run `golint` and address warnings
- Keep functions small and focused
- Write meaningful variable names
- Add comments for exported functions

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests after the first line

Example:
```
Add rate limiting middleware

- Implement IP-based rate limiting
- Add configuration options
- Include tests for rate limiter

Closes #123
```

### Code Review Process

1. All submissions require review
2. We use GitHub pull requests for this purpose
3. The core team looks at pull requests on a regular basis
4. After feedback has been given, we expect responses within two weeks

## Testing

- Write unit tests for new features
- Ensure all tests pass before submitting PR
- Aim for high test coverage
- Test both success and error cases

Run tests:
```bash
make test
make test-coverage
```

## Documentation

- Update README.md if you change functionality
- Add/update Swagger comments for API endpoints
- Document new environment variables
- Update examples if behavior changes

## Project Structure

Please maintain the existing project structure:
- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public library code
- `migrations/` - Database migrations
- `docs/` - Documentation

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inspiring community for all.

### Our Standards

Examples of behavior that contributes to a positive environment:
- Using welcoming and inclusive language
- Being respectful of differing viewpoints
- Gracefully accepting constructive criticism
- Focusing on what is best for the community

Examples of unacceptable behavior:
- The use of sexualized language or imagery
- Trolling, insulting/derogatory comments, and personal attacks
- Public or private harassment
- Publishing others' private information without permission

Thank you for contributing! 🚀
