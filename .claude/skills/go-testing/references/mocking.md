# Mocking and Test Fixtures

> **beer-lms house style: hand-written fakes, not a mock framework.** Implement the consumer-side port with a small struct (see `fakeCourseRepo` in `internal/modules/courses/app/service_test.go`). The `testify/mock` examples below are kept for reference — use them only when you specifically need argument matchers or call-sequence verification; the default is a fake.

## Hand-written fake (the default)

Implement the port the service consumes; store state in maps, inject an `err` field to drive error paths:

```go
type fakeCourseRepo struct {
    courses map[uuid.UUID]*domain.Course
    err     error
}

func newFakeRepo() *fakeCourseRepo {
    return &fakeCourseRepo{courses: map[uuid.UUID]*domain.Course{}}
}

func (f *fakeCourseRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Course, error) {
    if f.err != nil {
        return nil, f.err
    }
    c, ok := f.courses[id]
    if !ok {
        return nil, domain.ErrCourseNotFound
    }
    return c, nil
}
// ... implement the rest of the CourseRepository port the same way ...
```

A fake is real Go the compiler type-checks, reads like the production type, and doesn't couple the test to a call order.

## Mocks with testify/mock — available, NOT the house style (reference only)

Create interfaces for your dependencies, then mock them.

> For the full testify/mock API (argument matchers, call modifiers, verification), see [testify/mock reference](./testify-mock.md).

```go
// Define the interface
type Database interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
}

// Mock implementation
type MockDatabase struct {
    mock.Mock
}

func (m *MockDatabase) GetUser(id string) (*User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockDatabase) CreateUser(user *User) error {
    args := m.Called(user)
    return args.Error(0)
}

// Usage in tests
func TestService_GetUser(t *testing.T) {
    is := assert.New(t)

    mockDB := new(MockDatabase)
    service := NewService(mockDB)

    expectedUser := &User{ID: "1", Name: "John"}
    mockDB.On("GetUser", "1").Return(expectedUser, nil)

    user, err := service.GetUser("1")

    is.NoError(err)
    is.Equal(expectedUser, user)
    mockDB.AssertExpectations(t)
}

func TestService_GetUser_NotFound(t *testing.T) {
    is := assert.New(t)

    mockDB := new(MockDatabase)
    service := NewService(mockDB)

    mockDB.On("GetUser", "999").Return(nil, ErrNotFound)

    user, err := service.GetUser("999")

    is.Error(err)
    is.ErrorIs(err, ErrNotFound)
    is.Nil(user)
    mockDB.AssertExpectations(t)
}
```

## Mock Organization

For larger codebases, organize mocks alongside the code they mock:

```go
// user_service.go
type UserService struct {
    db    Database
    email EmailService
}
type Database interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
}
type EmailService interface {
    SendWelcomeEmail(to string) error
}
```

```go
// user_service_test.go
package mypackage_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "path/to/mypackage"
)

// MockDatabase implements mypackage.Database
type MockDatabase struct {
    mock.Mock
}
func (m *MockDatabase) GetUser(id string) (*mypackage.User, error) {
    args := m.Called(id)
    if args.Get(0) == nil { return nil, args.Error(1) }
    return args.Get(0).(*mypackage.User), args.Error(1)
}
func (m *MockDatabase) CreateUser(user *mypackage.User) error {
    return m.Called(user).Error(0)
}

// MockEmailService implements mypackage.EmailService
type MockEmailService struct {
    mock.Mock
}
func (m *MockEmailService) SendWelcomeEmail(to string) error {
    return m.Called(to).Error(0)
}

func TestUserService_CreateUser(t *testing.T) {
    mockDB := new(MockDatabase)
    mockEmail := new(MockEmailService)
    service := mypackage.NewUserService(mockDB, mockEmail)

    user := &mypackage.User{Name: "Test", Email: "test@example.com"}
    mockDB.On("CreateUser", user).Return(nil)
    mockEmail.On("SendWelcomeEmail", "test@example.com").Return(nil)

    err := service.CreateUser(user)

    assert.NoError(t, err)
    mockDB.AssertExpectations(t)
    mockEmail.AssertExpectations(t)
}
```

## Test Fixtures

Create reusable test data in a separate package or file:

```go
package fixtures

import "time"

var (
    DefaultUser = &User{
        ID:        "user-123",
        Name:      "Jane Doe",
        Email:     "jane@example.com",
        CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    }

    AdminUser = &User{
        ID:        "admin-1",
        Name:      "Admin User",
        Email:     "admin@example.com",
        Role:      "admin",
        CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    }
)

func NewUser(name, email string) *User {
    return &User{
        ID:        "user-" + uuid.New().String(),
        Name:      name,
        Email:     email,
        CreatedAt: time.Now(),
    }
}
```

## Time Mocking — beer-lms uses an injectable `Clock` port (no clockwork)

This repo does **not** depend on `clockwork`. Time-dependent logic takes a `Clock` interface (`Now() time.Time`) declared in the module's `app/ports.go`; production wires `RealClock()` (`time.Now().UTC()`), tests inject a fake:

```go
type fakeClock struct{ now time.Time }
func (c fakeClock) Now() time.Time { return c.now }

func TestService_PublishesAtNow(t *testing.T) {
    fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
    svc := app.NewCourseService(newFakeRepo(), fakeClock{now: fixed})
    // ... assert the timestamp the service recorded equals `fixed` ...
}
```

To advance time, hand the service a fresh `fakeClock{now: fixed.Add(d)}` (or make the fake's `now` a pointer you mutate). For deterministic timer/`context`-deadline tests, prefer the stdlib `testing/synctest` (Go 1.25+) shown in the skill rather than a third-party clock library.
