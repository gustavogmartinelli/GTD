# Testing the API with Insomnia

`insomnia-collection.json` is an Insomnia v4 export covering every endpoint the
server exposes, including the error cases.

## 1. Start the server

```bash
go run ./cmd/gtd-server
```

It listens on `:8080` and writes to `gtd.db` in the working directory. Override
either:

```bash
go run ./cmd/gtd-server -addr 127.0.0.1:9000 -db /tmp/scratch.db
```

## 2. Import the collection

Insomnia → **Create** (or the app menu) → **Import** → **From File** → pick
`api/insomnia-collection.json`. It lands as a collection named **GTD API**.

## 3. Set the environment

The collection ships one environment with two variables:

| Variable | Default | Notes |
|---|---|---|
| `base_url` | `http://localhost:8080` | Change if you passed `-addr`. |
| `item_id` | `1` | The item most requests act on. |

Capture an item first, then paste the `id` from the response into `item_id`
(Environment dropdown → **Manage Environments**). Everything in *04 Clarify & do*
and *05 Delete* targets that id.

## 4. Suggested run order

The folders are numbered to be walked top to bottom:

1. **01 Health** — confirms the server is up.
2. **02 Capture** — create an item; note its `id`. Also covers the empty-title
   and malformed-JSON rejections.
3. **03 Read & filter** — list everything, filter by status, fetch one, and see
   the 404 / invalid-id responses.
4. **04 Clarify & do** — process the item into a next action, edit it, complete
   it, then try completing it twice.
5. **05 Delete** — remove it, and confirm deleting a missing id 404s.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | Liveness. |
| `POST` | `/items` | Capture to inbox. |
| `GET` | `/items` | List; `?status=inbox\|next`, `?include_completed=true`. |
| `GET` | `/items/{id}` | Fetch one. |
| `PATCH` | `/items/{id}` | Update title, notes, status or context. |
| `DELETE` | `/items/{id}` | Delete. |
| `POST` | `/items/{id}/process` | Clarify into a next action. |
| `POST` | `/items/{id}/complete` | Mark done. |

Errors come back as `{"error": "..."}` with `400` (validation), `404` (missing)
or `500`. Completed items are hidden from `GET /items` unless you pass
`include_completed=true`.

## Two behaviours worth knowing

- **`PATCH` fields are all optional.** Omitting one leaves it untouched; sending
  `""` clears it. That's why the DTO uses pointers.
- **`POST /items/{id}/process` with no body clears the item's context.**
  `Item.Process` assigns `Context` unconditionally, while notes are only
  overwritten when non-empty. The *Process (no body)* request demonstrates it.
  If that asymmetry is unintended, it belongs in `internal/domain/item.go`.

## Status responses

Verified against a running server — status codes the collection expects:

| Request | Status |
|---|---|
| Capture | `201` |
| Capture, blank title | `400 title is required` |
| List, `status=someday` | `400 status must be 'inbox' or 'next'` |
| Get / Delete missing id | `404 item not found` |
| Get `/items/abc` | `400 invalid id` |
| Complete twice | `400 item is already completed` |
| Delete | `204` (empty body) |
