# Context in the Application Layer

## Overview

Use `context.Context` to propagate deadlines, cancellation signals, and request-scoped metadata through the application and infrastructure layers.

It is always the first parameter of any function or method performing I/O, database queries, or background jobs. Pure domain logic (such as in `core/game` or `core/adjudicator`) must not accept or use context.

## Core Rules

1. **Pass, Don't Store**: Always pass `context.Context` down as a function argument. Never store a context inside a struct field (such as `GameWorkflow`).
2. **First Parameter**: It should always be the first parameter of a function: `func DoWork(ctx context.Context, ...)`.
3. **No Nil**: Never pass a `nil` context. If you are unsure or writing test code, use `context.Background()`.
4. **Use for I/O**: Application methods and operations that can block, wait, or perform I/O should accept a context.
5. **Keep Business Inputs Explicit**: Do not hide game IDs, player IDs, orders, or other required inputs in context values. Put them in command structs.
6. **Cancellation Is Cooperative**: Cancelling a context signals work to stop; it does not forcibly terminate a function or goroutine.

During development and in tests, use a background context with no deadline:

```go
ctx := context.Background()
err := workflow.SubmitOrder(ctx, cmd)
```

Use `context.TODO()` instead when a real request or job context will eventually be supplied but that caller has not been designed yet.

---

## Examples

### 1. Simple Workflow Method

The workflow accepts `ctx` from the presentation layer and passes it directly to repositories:

```go
package gameplay

import (
	"context"
	"github.com/matt-in-space/diplomacy/core/game"
)

func (w *GameWorkflow) SubmitOrder(ctx context.Context, cmd SubmitOrderCommand) error {
	stored, err := w.games.Get(ctx, cmd.GameID)
	if err != nil {
		return err
	}

	gameMap, err := w.maps.Get(ctx, stored.Game.MapID)
	if err != nil {
		return err
	}

	if err := stored.Game.SubmitOrder(cmd.Order, gameMap); err != nil {
		return err
	}

	_, err = w.games.Save(ctx, stored.Game, stored.Version)
	return err
}
```

---

### 2. HTTP Presentation Layer (Future Integration)

The web server automatically generates a context for every incoming request. The handler maps the request body into a command, reads the request-scoped context, and passes it to the workflow.

If the client disconnects or the server reaches its timeout, Go automatically cancels `r.Context()`.

```go
package web

import (
	"encoding/json"
	"net/http"
)

func (h *GameHandler) SubmitOrder(w http.ResponseWriter, r *http.Request) {
	var cmd SubmitOrderCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.workflow.SubmitOrder(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
```

---

### 3. Database Layer Repository (Future SQL Example)

Pass `ctx` directly into SQL queries using SQL drivers that accept contexts. When the context is cancelled, the driver can cancel the corresponding database operation:

```go
package postgres

import (
	"context"
	"database/sql"
)

type PostgresGameRepository struct {
	db *sql.DB
}

func (r *PostgresGameRepository) Save(ctx context.Context, g *game.Game, version uint64) (uint64, error) {
	query := `
		UPDATE games 
		SET state = $1, version = version + 1 
		WHERE id = $2 AND version = $3`

	result, err := r.db.ExecContext(ctx, query, serialize(g), g.ID, version)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		return 0, ErrConcurrentUpdate
	}

	return version + 1, nil
}
```

---

### 4. Background Job / Worker Contexts

When running background deadline checking or cron tasks, the loop uses `context.WithTimeout` to guarantee that slow tasks do not block the worker queue indefinitely:

```go
package worker

import (
	"context"
	"time"
)

func (w *DeadlineWorker) ProcessTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := w.workflow.AdvanceGame(ctx, command)
	if err != nil {
		w.log.Errorf("task failed: %v", err)
	}
}
```
