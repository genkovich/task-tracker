# Testing Database Code

> **beer-lms note:** the house style differs from the upstream snippets below in two ways:
> 1. **Unit tests use hand-written fakes, not `testify/mock` or `go-sqlmock`.** Define the repository port in `app/ports.go` and write a struct that implements it (see `fakeCourseRepo` in `internal/modules/courses/app/service_test.go`). The `testify/mock` and `sqlmock` sections below are kept for reference but are **not used here**.
> 2. **Integration tests use `dbtest.StartPostgres` (testcontainers `postgres:18-alpine`) + `database.RunMigrations(migrations.FS, c.DSN)`**, gated by `//go:build integration` and run with `make test-integration`. The `sqlx.Connect` / `suite.Suite` shape below is the upstream spelling — translate it to the beer-lms recipe in the `go-testing` skill.

## Unit Tests with Mocks — beer-lms uses hand-written fakes instead (reference only)

Define a repository interface so business logic can be tested without a database. The upstream below mocks it with `testify/mock`; beer-lms instead implements the port with a hand-written fake struct (no mock framework):

```go
// Repository interface — the contract
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*User, bool, error)
    Create(ctx context.Context, user *User) error
}

// Production implementation
type pgUserRepository struct {
    db *sqlx.DB
}

func (r *pgUserRepository) GetByID(ctx context.Context, id int64) (*User, bool, error) {
    var user User
    err := r.db.GetContext(ctx, &user, "SELECT id, name, email FROM users WHERE id = $1", id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, false, nil
        }
        return nil, false, fmt.Errorf("querying user %d: %w", id, err)
    }
    return &user, true, nil
}
```

### Mock for service-layer tests

```go
type mockUserRepo struct {
    mock.Mock
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func TestUserService_GetUser(t *testing.T) {
    repo := new(mockUserRepo)
    svc := NewUserService(repo)

    expected := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
    repo.On("GetByID", mock.Anything, int64(1)).Return(expected, nil)

    user, err := svc.GetUser(context.Background(), 1)
    require.NoError(t, err)
    assert.Equal(t, expected, user)
    repo.AssertExpectations(t)
}

func TestUserService_GetUser_NotFound(t *testing.T) {
    repo := new(mockUserRepo)
    svc := NewUserService(repo)

    repo.On("GetByID", mock.Anything, int64(999)).Return(nil, ErrUserNotFound)

    user, err := svc.GetUser(context.Background(), 999)
    assert.Nil(t, user)
    assert.ErrorIs(t, err, ErrUserNotFound)
}
```

Unit tests verify business logic, not SQL correctness. They run fast and without external dependencies.

## sqlmock for Query-Level Testing

When you need to verify exact SQL without a real database, use [DATA-DOG/go-sqlmock](https://github.com/DATA-DOG/go-sqlmock):

```go
func TestGetByID_sqlmock(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    sqlxDB := sqlx.NewDb(db, "postgres")
    repo := &pgUserRepository{db: sqlxDB}

    rows := sqlmock.NewRows([]string{"id", "name", "email"}).
        AddRow(1, "Alice", "alice@example.com")
    mock.ExpectQuery("SELECT id, name, email FROM users WHERE id = \\$1").
        WithArgs(1).
        WillReturnRows(rows)

    user, err := repo.GetByID(context.Background(), 1)
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
    assert.NoError(t, mock.ExpectationsWereMet())
}
```

sqlmock is useful for verifying query structure and error handling paths, but it does not validate that your SQL is correct against a real database schema.

## Integration Tests

Integration tests run against a real database. Gate them with build tags so `go test ./...` skips them by default:

```go
//go:build integration

package repository_test

import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type UserRepoSuite struct {
    suite.Suite
    db *sqlx.DB
    tx *sqlx.Tx
}

func (s *UserRepoSuite) SetupSuite() {
    dsn := os.Getenv("TEST_DATABASE_URL") // e.g., postgres://test:test@localhost:5432/testdb?sslmode=disable
    db, err := sqlx.Connect("postgres", dsn)
    s.Require().NoError(err)
    s.db = db
    // Run migrations here if needed
}

func (s *UserRepoSuite) TearDownSuite() {
    s.db.Close()
}

func (s *UserRepoSuite) SetupTest() {
    tx, err := s.db.Beginx()
    s.Require().NoError(err)
    s.tx = tx
}

func (s *UserRepoSuite) TearDownTest() {
    s.tx.Rollback() // rolls back all changes — each test starts clean
}

func (s *UserRepoSuite) TestCreateAndGet() {
    repo := NewUserRepository(s.tx)
    user := &User{Name: "Alice", Email: "alice@example.com"}

    err := repo.Create(context.Background(), user)
    s.Require().NoError(err)
    s.NotZero(user.ID)

    got, err := repo.GetByID(context.Background(), user.ID)
    s.Require().NoError(err)
    s.Equal("Alice", got.Name)
}

func TestUserRepoSuite(t *testing.T) {
    suite.Run(t, new(UserRepoSuite))
}
```

Run integration tests:

```bash
go test -tags=integration -v ./internal/repository/...
```

### Test database with testcontainers-go

beer-lms wraps this in `dbtest.StartPostgres(ctx, t)` (`internal/platform/database/dbtest/container.go`) — image `postgres:18-alpine`, `t.Skip` when docker is absent, auto-terminate via `t.Cleanup` — then applies migrations:

```go
//go:build integration

func TestCourseRepo_Integration(t *testing.T) {
    ctx := context.Background()
    c := dbtest.StartPostgres(ctx, t)                  // *Container{ DB *database.DB; DSN string }
    if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
        t.Fatalf("RunMigrations: %v", err)
    }
    repo := infra.NewPostgresCourseRepository(c.DB)
    // ... exercise repo against the real schema ...
}
```

The raw upstream form (`postgres.Run` + `sqlx.Connect`) — reference only; beer-lms calls `dbtest.StartPostgres` instead and connects via `database.New`:

```go
func (s *UserRepoSuite) SetupSuite() {
    ctx := context.Background()
    container, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    s.Require().NoError(err)
    s.container = container

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    s.Require().NoError(err)
    s.db, err = sqlx.Connect("postgres", connStr)
    s.Require().NoError(err)
}
```

## What to Test

| What                      | Unit test (mock) | Integration test |
| ------------------------- | :--------------: | :--------------: |
| Business logic            |        ✓         |                  |
| SQL correctness           |                  |        ✓         |
| Error paths (not found)   |        ✓         |        ✓         |
| Transaction boundaries    |                  |        ✓         |
| NULL handling round-trips |                  |        ✓         |
| Constraint violations     |                  |        ✓         |
| Query performance         |                  | ✓ (with EXPLAIN) |

Unit tests MUST use mocks (interface mocks or sqlmock) — no real database connections. Integration tests MUST use build tags (`//go:build integration`) to separate from unit tests. Integration tests SHOULD use testcontainers-go for reproducible database environments in CI. NEVER test against production databases.

→ See the `go-testing` skill for the beer-lms integration recipe (`dbtest.StartPostgres` + `RunMigrations(migrations.FS)` + miniredis), hand-written fakes, and `make test-integration`.
