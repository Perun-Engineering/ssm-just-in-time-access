# Testing Suite

This directory contains the comprehensive testing suite for the SSM Access Manager.

## Test Structure

```
test/
├── helpers/          # Test utilities, mocks, and fixtures
├── unit/            # Unit tests
│   ├── models/      # Model layer tests
│   ├── services/    # Service layer tests (in progress)
│   ├── repositories/# Repository layer tests (planned)
│   └── validation/  # Validation logic tests (planned)
├── integration/     # Integration tests (planned)
├── manual/          # Manual testing checklists (planned)
└── testdata/        # Test data files
```

## Running Tests

### Run All Model Tests
```bash
go test ./test/unit/models/... -v
```

### Run with Coverage
```bash
go test ./test/unit/models/... -cover
```

### Run Specific Test
```bash
go test ./test/unit/models/... -run TestApprovalGroup_Validate
```

## Test Coverage Status

**Phase 1: Test Infrastructure** ✅ Complete
- Test directory structure created
- Test helpers (fixtures, mocks, assertions) implemented
- Dependencies configured

**Phase 2: Model Tests** ✅ Complete (28 tests passing)
- ApprovalGroup: 8 tests
- AccessRequest: 11 tests  
- User: 9 tests

**Phase 3: Service Tests** 🚧 Blocked
- Service layer requires interface-based dependency injection
- Recommendation: Refactor services to accept interfaces

## Test Helpers

### Fixtures (test/helpers/fixtures.go)
Pre-configured test data generators:
- CreateTestSecurityGroup() - Security approval group
- CreateTestManagerGroup() - Manager approval group
- CreateTestRequest() - Access request with manager group
- CreateTestLegacyRequest() - Legacy request
- CreateTestAdminUser() - Administrator user
- CreateTestRegularUser() - Regular user

### Mocks (test/helpers/mocks.go)
Mock implementations:
- MockApprovalGroupRepository
- MockAuthorizationService
- MockAuditService
- MockGroupMembershipCache

### Assertions (test/helpers/assertions.go)
Custom assertions:
- AssertRequestStatus()
- AssertRequestPartiallyApproved()
- AssertRequestFullyApproved()

## Next Steps

1. Refactor services to accept interfaces for testability
2. Complete service layer tests
3. Add repository tests with DynamoDB mocks
4. Add integration tests for workflows
5. Setup CI/CD with GitHub Actions
