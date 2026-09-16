# GTD

Getting Things Done, with AI.

A minimal JSON API for capturing inbox items and turning them into next
actions, backed by SQLite.

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
