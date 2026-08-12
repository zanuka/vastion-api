# com-scan-api

Module: `github.com/zanuka/com-scan-api`

## Layers

`domain` → `repository/mongodb` → `service` → `handler` | `graphql`

- `internal/domain` — entities + repository interfaces. No bson, ObjectID, mongo, Huma, or gqlgen.
- `internal/repository/mongodb` — only driver import. Map string IDs ↔ ObjectID here.
- `internal/service` — use cases; domain interfaces only.
- `internal/handler` / `internal/graphql` — HTTP only; call service. Never import mongo.
- `cmd/api` — thin. Wire in `internal/app`.

## Style

No comments in Go. Why lives in `docs/`. Env + `.env.example`, not a config framework.

## Stack

Huma v2 + humachi + chi v5. mongo-driver/v2. slog + `request_id`. `DATABASE_URL`, `PORT`, `CORS_ORIGINS`. Auth stub headers `X-Operator-Role`, `X-Operator-Name`. No JWT. No GraphQL until Phase 4. No ack mutations in GraphQL.

## Models

Default: Composer 2.5. Escalate to Grok 4.6 for layering / domain vs Mongo vs HTTP. Codex for ack state machine, indexes, aggregation, dataloaders.

## Docs

Execution: `docs/dev/section-2-build-plan.md`. North star: `docs/dev/com-scan-plan.md`.
