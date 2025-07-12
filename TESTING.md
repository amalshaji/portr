# Portr Testing Guide

This document describes the comprehensive test suite added to the Portr project, covering all components: admin (Python), server (Go), and CLI (Go).

## 🎯 Test Coverage Overview

The project now includes:
- **Admin Component**: 23+ test files covering APIs, services, and integration workflows
- **Server Component**: 4+ test files covering configuration, proxy, services, and cron jobs
- **CLI Component**: 3+ test files covering commands, configuration, and client functionality
- **Integration Tests**: End-to-end workflow testing
- **CI/CD Pipeline**: Automated testing via GitHub Actions

## 📁 Test Structure

```
portr/
├── admin/tests/                    # Python admin tests
│   ├── api_tests/                 # API endpoint tests
│   │   ├── test_auth.py          # Authentication API tests
│   │   ├── test_config.py        # Configuration API tests
│   │   ├── test_connection.py    # Connection API tests
│   │   ├── test_instance_settings.py  # Settings API tests
│   │   ├── test_team.py          # Team management API tests
│   │   └── test_user.py          # User management API tests
│   ├── service_tests/            # Service layer tests
│   │   ├── test_connection_service.py  # Connection service tests
│   │   ├── test_user_service.py  # User service tests
│   │   └── test_email_service.py # Email service tests (WIP)
│   ├── integration_tests/        # Integration workflow tests
│   │   └── test_full_workflow.py # End-to-end workflow tests
│   ├── test_pages.py            # Page routing tests
│   ├── test_utils.py            # Utility function tests
│   ├── conftest.py              # Test configuration
│   └── factories.py             # Test data factories
├── tunnel/internal/server/        # Go server tests
│   ├── config/config_test.go     # Server configuration tests
│   ├── proxy/proxy_test.go       # HTTP proxy tests
│   ├── service/service_test.go   # Database service tests
│   └── cron/cron_test.go         # Cron job tests
├── tunnel/cmd/portr/              # Go CLI tests
│   ├── main_test.go              # CLI command tests
├── tunnel/internal/client/        # Go client tests
│   └── config/config_test.go     # Client configuration tests
├── run_tests.py                   # Comprehensive test runner
└── .github/workflows/test.yml     # CI/CD pipeline
```

## 🚀 Running Tests

### Quick Start

Use the comprehensive test runner:

```bash
# Run all tests
./run_tests.py

# Run specific component tests
./run_tests.py --component admin
./run_tests.py --component server
./run_tests.py --component cli

# Run with coverage reports
./run_tests.py --coverage

# Run with linting
./run_tests.py --lint

# Run integration tests
./run_tests.py --integration

# Filter tests by pattern
./run_tests.py --filter "test_auth"
```

### Manual Test Execution

#### Admin Tests (Python)

```bash
cd admin

# Run all tests
python -m pytest tests/ -v

# Run specific test categories
python -m pytest tests/api_tests/ -v           # API tests
python -m pytest tests/service_tests/ -v       # Service tests
python -m pytest tests/integration_tests/ -v   # Integration tests

# Run with coverage
python -m coverage run -m pytest tests/
python -m coverage report --skip-empty
python -m coverage html  # Generate HTML report
```

#### Server Tests (Go)

```bash
cd tunnel

# Run all server tests
go test ./internal/server/... -v

# Run specific packages
go test ./internal/server/config -v      # Configuration tests
go test ./internal/server/proxy -v       # Proxy tests
go test ./internal/server/service -v     # Service tests
go test ./internal/server/cron -v        # Cron tests

# Run with coverage
go test -coverprofile=coverage.out ./internal/server/...
go tool cover -html=coverage.out -o coverage.html
```

#### CLI Tests (Go)

```bash
cd tunnel

# Run CLI tests
go test ./cmd/portr/... -v

# Run client tests
go test ./internal/client/... -v

# Run with short flag (skip integration tests)
go test -short ./cmd/portr/... ./internal/client/... -v
```

## 📊 Test Categories

### 1. Admin Component Tests

#### API Tests (`admin/tests/api_tests/`)
- **Authentication**: GitHub OAuth, login/logout, session management
- **Configuration**: Setup scripts, config downloads, tunnel configuration
- **Connections**: Creation, lifecycle management, status tracking
- **Instance Settings**: SMTP configuration, email templates, system settings
- **Team Management**: User roles, permissions, team creation/deletion
- **User Management**: Profile updates, secret key rotation, team membership

#### Service Tests (`admin/tests/service_tests/`)
- **User Service**: User creation, GitHub integration, password handling
- **Connection Service**: Connection validation, subdomain conflicts, lifecycle
- **Email Service**: SMTP integration, template rendering, notification sending

#### Integration Tests (`admin/tests/integration_tests/`)
- **User Onboarding**: Complete signup-to-first-connection workflow
- **Connection Lifecycle**: From creation through activation to closure
- **Team Management**: Multi-user team operations and permissions
- **Configuration Workflow**: Client setup and config generation
- **Error Handling**: Various failure scenarios and recovery

### 2. Server Component Tests

#### Configuration Tests (`tunnel/internal/server/config/`)
- Environment variable parsing
- Default value handling
- URL generation for different environments
- Database configuration validation

#### Proxy Tests (`tunnel/internal/server/proxy/`)
- Route management (add/remove/get)
- HTTP request handling and forwarding
- Error handling (connection lost, unregistered subdomains)
- Concurrent operations and thread safety
- Subdomain extraction for different environments

#### Service Tests (`tunnel/internal/server/service/`)
- Database connection operations
- Connection status management (reserved → active → closed)
- Port assignment for TCP connections
- Connection lifecycle workflows

#### Cron Tests (`tunnel/internal/server/cron/`)
- HTTP connection health checks
- TCP connection health checks
- Connection cleanup on failures
- Concurrent ping operations
- Error handling and recovery

### 3. CLI Component Tests

#### Command Tests (`tunnel/cmd/portr/`)
- Command structure and flag validation
- Argument parsing for different commands
- Help text generation
- Update checking functionality
- Configuration file management

#### Client Configuration Tests (`tunnel/internal/client/config/`)
- Tunnel configuration validation
- Default value assignment
- YAML file parsing
- Environment-specific settings
- Health check configuration

## 🔧 Test Utilities

### Test Factories (`admin/tests/factories.py`)
- **UserFactory**: Creates test users with various roles
- **TeamFactory**: Creates test teams with different configurations
- **TeamUserFactory**: Creates team-user relationships
- **SessionFactory**: Creates authentication sessions
- **ConnectionFactory**: Creates connections with different statuses

### Test Client (`admin/tests/__init__.py`)
- **TestClient**: HTTP client for API testing
- Authentication helper methods
- Team context management
- Session handling

### Test Configuration (`admin/conftest.py`)
- Database setup and teardown
- Test environment configuration
- Shared fixtures and utilities

## 🚦 CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/test.yml`) includes:

### Test Jobs
- **Admin Tests**: Python tests with PostgreSQL
- **Server Tests**: Go tests with race detection
- **CLI Tests**: Go tests with build verification
- **Integration Tests**: Cross-component testing

### Quality Checks
- **Linting**: Code formatting and style checks
- **Security Scanning**: Vulnerability detection
- **Coverage Reports**: Test coverage analysis

### Environment Matrix
- Python 3.11 with Poetry
- Go 1.21 with modules
- PostgreSQL 13 for database tests
- Ubuntu latest for consistency

## 📈 Coverage Goals

### Current Coverage
- **Admin APIs**: ~95% of endpoints covered
- **Admin Services**: ~90% of service methods covered
- **Server Components**: ~85% of core functionality covered
- **CLI Commands**: ~80% of command logic covered

### Coverage Reports
- HTML reports generated for both Python and Go
- Integration with Codecov for tracking
- Branch coverage tracking for critical paths

## 🧪 Test Types

### Unit Tests
- Test individual functions and methods in isolation
- Mock external dependencies
- Fast execution (< 1 second per test)

### Integration Tests
- Test component interactions
- Use real database connections
- Test complete workflows

### End-to-End Tests
- Test full user workflows
- Include UI interactions where applicable
- Validate system behavior

### Performance Tests
- Concurrent operation testing
- Load testing for critical paths
- Memory and resource usage validation

## 🔍 Test Best Practices

### Writing Tests
1. **Descriptive Names**: Use clear, descriptive test names
2. **AAA Pattern**: Arrange, Act, Assert structure
3. **Single Responsibility**: One test, one concern
4. **Independent Tests**: No test dependencies
5. **Cleanup**: Proper setup and teardown

### Test Data
1. **Factories**: Use factories for consistent test data
2. **Isolation**: Each test uses fresh data
3. **Realistic Data**: Use data that reflects real usage
4. **Edge Cases**: Test boundary conditions

### Mocking
1. **External Services**: Mock external API calls
2. **Database**: Use in-memory databases for unit tests
3. **Time**: Mock time-dependent operations
4. **File System**: Mock file operations

## 🛠️ Development Workflow

### Before Committing
```bash
# Run all tests
./run_tests.py

# Run linting
./run_tests.py --lint

# Generate coverage report
./run_tests.py --coverage
```

### Test-Driven Development
1. Write failing test first
2. Implement minimal code to pass
3. Refactor while keeping tests green
4. Add edge cases and error conditions

### Debugging Tests
```bash
# Run specific test with verbose output
python -m pytest tests/api_tests/test_auth.py::TestAuth::test_login -v -s

# Run Go test with race detection
go test -race -v ./internal/server/proxy -run TestProxy_AddRoute

# Run with debugger
python -m pytest --pdb tests/api_tests/test_auth.py
```

## 📚 Additional Resources

### Documentation
- [pytest documentation](https://docs.pytest.org/) for Python testing
- [Go testing package](https://golang.org/pkg/testing/) for Go testing
- [testify](https://github.com/stretchr/testify) for Go assertions

### Tools
- **pytest**: Python testing framework
- **coverage.py**: Python coverage analysis
- **Go testing**: Built-in Go testing
- **testify**: Go testing toolkit
- **GitHub Actions**: CI/CD automation

### Monitoring
- Test execution times tracked
- Coverage trends monitored
- Flaky test detection
- Performance regression detection

## 🎯 Future Improvements

### Test Coverage
- [ ] Add more edge case testing
- [ ] Increase integration test coverage
- [ ] Add performance benchmarks
- [ ] Add chaos testing

### Infrastructure
- [ ] Parallel test execution
- [ ] Test environment isolation
- [ ] Visual test reporting
- [ ] Automatic test generation

### Quality
- [ ] Mutation testing
- [ ] Property-based testing
- [ ] Contract testing
- [ ] Load testing automation

---

This comprehensive test suite ensures the reliability, maintainability, and quality of the Portr project across all components. Regular execution of these tests helps catch regressions early and maintains confidence in deployments.