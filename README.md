# GTD

Getting Things Done, with AI.

A minimal JSON API for capturing inbox items and turning them into next
actions, backed by SQLite.

## Architecture

Structured as Clean Architecture (Robert C. Martin), with dependencies
pointing inward only — nothing in an inner layer imports from an outer one:

```
cmd/gtd-server/            frameworks & drivers — composition root, wires
                            concrete adapters into use cases, starts the
                            HTTP server. The only package that knows about
                            every other layer.

internal/domain/           entities — the Item type and its business rules
                            (title can't be empty, can't complete twice, ...).
                            Zero imports from this project.

internal/usecase/          application business rules — one type per
                            operation (CaptureItemUseCase, ListItemsUseCase,
                            ...), each depending only on domain and on the
                            ItemRepository port it defines. Use cases don't
                            know whether that port is backed by SQLite, an
                            in-memory map, or anything else.

internal/adapter/
  repository/sqlite/       interface adapter — implements ItemRepository
                            against SQLite. Only this package imports
                            database/sql or the sqlite driver.
  rest/                    interface adapter — HTTP controllers/presenters.
                            Decodes requests into use case input, calls the
                            use case, encodes domain.Item into response
                            DTOs. Only this package imports net/http.
```

The dependency rule: `domain` depends on nothing, `usecase` depends only
on `domain`, `adapter/*` depend on `usecase` and `domain`, and `cmd/*` (the
composition root) depends on everything to wire it together. Swapping
storage (e.g. Postgres instead of SQLite) means adding a new
`adapter/repository/...` package that implements `usecase.ItemRepository`
— no change to `domain`, `usecase`, or `adapter/rest`.

## Run

```sh
go run ./cmd/gtd-server -addr :8080 -db gtd.db
```

## API

| Method | Path                 | Description                                   |
|--------|----------------------|------------------------------------------------|
| GET    | `/healthz`           | Health check                                    |
| POST   | `/items`              | Capture a new inbox item (`{title, notes}`)     |
| GET    | `/items`              | List items (`?status=inbox\|next`, `?include_completed=true`) |
| GET    | `/items/{id}`         | Get a single item                               |
| PATCH  | `/items/{id}`         | Update fields (`title`, `notes`, `status`, `context`) |
| POST   | `/items/{id}/process` | Clarify an inbox item into a next action (`{context, notes}`) |
| POST   | `/items/{id}/complete`| Mark an item done                               |
| DELETE | `/items/{id}`         | Delete an item                                  |

## Example

```sh
curl -X POST localhost:8080/items -d '{"title":"Buy milk"}'
curl -X POST localhost:8080/items/1/process -d '{"context":"@errands"}'
curl "localhost:8080/items?status=next"
curl -X POST localhost:8080/items/1/complete
```
