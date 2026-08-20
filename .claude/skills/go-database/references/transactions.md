# Transactions, Isolation Levels, and Locking

> **beer-lms note:** this repo uses **pgx**, not `database/sql`. The `BeginTxx` / `sql.TxOptions` calls below are the `database/sql` spelling — translate them to the pgx equivalents shown in each section. The platform wrapper exposes `db.Begin(ctx) (*Tx, error)` with default isolation.

## Basic transaction pattern

beer-lms (pgx, via the `database.DB` wrapper):

```go
tx, err := r.db.Begin(ctx)
if err != nil {
    return fmt.Errorf("begin tx: %w", err)
}
defer tx.Rollback(ctx) //nolint:errcheck // no-op once Commit succeeds

// ... execute queries using tx ...

if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("commit tx: %w", err)
}
```

`database/sql` spelling (NOT used here — reference only):

```go
tx, err := db.BeginTxx(ctx, nil) // default isolation (READ COMMITTED)
if err != nil {
    return fmt.Errorf("beginning transaction: %w", err)
}
defer tx.Rollback() // no-op if already committed
// ...
if err := tx.Commit(); err != nil {
    return fmt.Errorf("committing transaction: %w", err)
}
```

## Custom isolation level

pgx (beer-lms): pass `pgx.TxOptions` to `BeginTx` on the underlying pool when you genuinely need a non-default level:

```go
tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
```

`database/sql` spelling (reference only):

```go
tx, err := db.BeginTxx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable, // strongest guarantee
})
```

| Level | Use when |
| --- | --- |
| `LevelReadCommitted` | Default — good for most operations |
| `LevelRepeatableRead` | Need consistent reads within a transaction |
| `LevelSerializable` | Financial operations, inventory, anything with strict consistency |

## SELECT FOR UPDATE — prevent race conditions

beer-lms (pgx — manual scan):

```go
var balance decimal.Decimal
err := tx.QueryRow(ctx, "SELECT balance FROM accounts WHERE id = $1 FOR UPDATE", accountID).Scan(&balance)
// Row is locked until tx.Commit(ctx) or tx.Rollback(ctx)
```

For the `FOR UPDATE SKIP LOCKED` concurrent-claim pattern across instances, see `internal/platform/outbox` (`ClaimBatch`) and the `go-concurrency` rule.

`database/sql` spelling (reference only):

```go
var balance int
err := tx.GetContext(ctx, &balance, "SELECT balance FROM accounts WHERE id = $1 FOR UPDATE", accountID)
```

Use `FOR UPDATE` when you read a value, compute something from it, and then write it back. Without the lock, concurrent transactions can read stale data.

## Locking variants

| Clause | Effect |
| --- | --- |
| `FOR UPDATE` | Locks rows for write — other transactions block on same rows |
| `FOR UPDATE NOWAIT` | Same, but fails immediately instead of waiting |
| `FOR SHARE` | Locks rows for read — prevents writes but allows other reads |
