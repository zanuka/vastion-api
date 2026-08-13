# vastion-api

Module: `github.com/zanuka/vastion-api`

## Layers

`domain` → `repository/mongodb` → `service` → `handler` | `graphql`

- `internal/domain` — entities + repository interfaces. No bson, ObjectID, mongo, Huma, or gqlgen.
- `internal/repository/mongodb` — only driver import. Map string IDs ↔ ObjectID here.
- `internal/service` — use cases; domain interfaces only.
- `internal/handler` / `internal/graphql` — HTTP only; call service. Never import mongo.
- `cmd/api` — thin. Wire in `internal/app`.

## Style

No comments in Go. Prefer env + `.env.example` over a config framework.

## Stack

Huma v2 + humachi + chi v5. mongo-driver/v2. slog + `request_id`. `DATABASE_URL`, `PORT`, `CORS_ORIGINS`. Auth stub headers `X-Operator-Role`, `X-Operator-Name`. No JWT. GraphQL (gqlgen) is planned later; ack/reject stay on REST.
