# Struct Scanning and NULLable Columns

> **beer-lms note:** this repo is **pgx-native and scans manually** — `QueryRow(ctx, …).Scan(&a, &b)` / `rows.Scan(…)`. It does **not** use sqlx or `pgx.CollectRows`/`pgx.RowToStructByName`. The sqlx and `CollectRows` sections below are kept for reference only. What *does* apply here: **pointer fields for NULLable columns** and the **JSON marshaling** behavior — both work identically with manual `Scan`.

## Struct Scanning with sqlx — NOT USED in beer-lms (reference only)

Tag struct fields with `db:"column_name"` for sqlx:

```go
type User struct {
    ID        int64      `db:"id"`
    Name      string     `db:"name"`
    Email     string     `db:"email"`
    DeletedAt *time.Time `db:"deleted_at"` // NULLable
}

// Single row
var user User
err := db.GetContext(ctx, &user, "SELECT id, name, email, deleted_at FROM users WHERE id = $1", id)

// Multiple rows
var users []User
err := db.SelectContext(ctx, &users, "SELECT id, name, email, deleted_at FROM users WHERE active = true")
```

## Struct Scanning with pgx

> **beer-lms uses manual `rows.Scan`, not `CollectRows`.** The automatic mapping below is valid pgx but is not the house style — keep new repositories consistent with the existing `Postgres*Repository` files, which scan column-by-column.

beer-lms pattern (manual scan):

```go
rows, err := r.db.Query(ctx, "SELECT id, title FROM courses WHERE org_id = $1", orgID)
if err != nil {
    return nil, fmt.Errorf("list courses: %w", err)
}
defer rows.Close()
courses := make([]domain.Course, 0)
for rows.Next() {
    var c domain.Course
    if err := rows.Scan(&c.ID, &c.Title); err != nil {
        return nil, fmt.Errorf("scan course: %w", err)
    }
    courses = append(courses, c)
}
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("iterate courses: %w", err)
}
```

`pgx.CollectRows` (automatic mapping — NOT used here, reference only):

```go
rows, err := pool.Query(ctx, "SELECT id, name, email FROM users WHERE active = true")
if err != nil {
    return fmt.Errorf("querying users: %w", err)
}
users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
```

### JSONB columns

beer-lms scans a `jsonb` column into a `[]byte`, then `json.Unmarshal`s it into the typed value:

```go
var raw []byte
if err := r.db.QueryRow(ctx, "SELECT payload FROM peer_blobs WHERE id = $1", id).Scan(&raw); err != nil {
    return nil, fmt.Errorf("get blob: %w", err)
}
var blob domain.PeerBlob
if err := json.Unmarshal(raw, &blob); err != nil {
    return nil, fmt.Errorf("decode blob: %w", err)
}
```

## JSON Marshaling

Struct tags for both database and JSON work together. Pointer fields marshal to `null` in JSON when NULL in the database:

```go
type User struct {
    ID        int64      `db:"id"         json:"id"`
    Name      string     `db:"name"       json:"name"`
    Email     string     `db:"email"      json:"email"`
    Bio       *string    `db:"bio"        json:"bio,omitempty"` // NULL → omitted in JSON
    DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"`    // NULL → null in JSON
}
```

## NULLable Columns

Three approaches, from most to least recommended:

**1. Pointer fields (recommended)** — clean, works with JSON marshaling:

```go
type User struct {
    ID        int64      `db:"id"    json:"id"`
    Name      string     `db:"name"  json:"name"`
    DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"` // nil when NULL
}
// Check: if user.DeletedAt != nil { ... }
```

**2. `sql.NullXxx` types** or `sql.Null[T]` generic — explicit but verbose, requires custom JSON marshaling:

```go
type User struct {
    ID        int64          `db:"id"`
    Bio       sql.NullString `db:"bio"`
}
// Check: if user.Bio.Valid { use(user.Bio.String) }
```

**3. `COALESCE` in SQL** — moves NULL handling to the query:

```sql
SELECT id, COALESCE(bio, '') AS bio FROM users WHERE id = $1
```
