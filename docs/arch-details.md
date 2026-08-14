# Architecture Decision Records

Decisions that shape `vastion-api`. Each record is **Accepted** unless noted. Update this file when a decision changes or a new one lands.

**Status of the codebase:** Phase 2 — REST vertical slice (`/api/v1` detections list/detail/ack/reject and sites list). GraphQL is later.

---

## ADR-001 — Layered packages and dependency rule

**Status:** Accepted

**Context:** The API must stay swappable at the edges (HTTP vs GraphQL, Mongo vs another store) without rewriting business rules. A flat “handlers talk to the driver” layout couples transport, persistence, and domain.

**Decision:** Enforce a strict layering:

```
domain → repository/mongodb → service → handler | graphql
```

| Package | Responsibility | Forbidden |
| --- | --- | --- |
| `internal/domain` | Entities and repository interfaces | `bson`, `primitive.ObjectID`, mongo, Huma, gqlgen |
| `internal/repository/mongodb` | Driver, BSON mapping, indexes, queries | Returning raw driver types to handlers |
| `internal/service` | Use cases; depends on domain interfaces only | Importing mongo or Huma |
| `internal/handler`, `internal/graphql` | Transport only; call `service` | Importing mongo |
| `internal/app` | Composition root (wire config → deps → server) | Business rules |
| `cmd/api` | Config load, construct app, listen, shutdown | Application logic |

Dependencies point **inward** toward domain. Only `repository/mongodb` imports the Mongo driver.

**Consequences:**

- Handlers and resolvers share the same services.
- Domain stays unit-testable without Mongo or HTTP.
- Mapping `string` IDs ↔ ObjectID/BSON happens only in the repository package.
- Violations (e.g. `bson` on domain structs) are treated as architecture bugs.

```
┌─────────────────────────────────────────┐
│  cmd/api  (main, signals)               │
│  ┌───────────────────────────────────┐  │
│  │  app (wire + http.Server)         │  │
│  │  ┌─────────────────────────────┐  │  │
│  │  │  router / middleware / CORS │  │  │
│  │  │  ┌───────────────────────┐  │  │  │
│  │  │  │  handler (Huma)       │  │  │  │
│  │  │  │  ┌─────────────────┐  │  │  │  │
│  │  │  │  │  service        │  │  │  │  │
│  │  │  │  │  ┌───────────┐  │  │  │  │  │
│  │  │  │  │  │  domain   │  │  │  │  │  │
│  │  │  │  │  └───────────┘  │  │  │  │  │
│  │  │  │  └─────────────────┘  │  │  │  │
│  │  │  └───────────────────────┘  │  │  │
│  │  │  repository/mongodb ────────┘  │  │
│  │  └────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

---

## ADR-002 — Thin `cmd` and composition in `internal/app`

**Status:** Accepted

**Context:** Fat `main` packages hide wiring and make lifecycle (serve vs shutdown) hard to test or reuse.

**Decision:**

- `cmd/api/main.go` loads config, builds `app.App`, serves, and handles `SIGINT`/`SIGTERM` with a bounded `Close`.
- `internal/app` connects Mongo, constructs services and the router, owns `http.Server`.
- Dependencies use constructor injection (`NewHealth(pinger)`, `router.Deps{...}`), not a DI framework.
- Small interfaces may be declared next to the consumer when useful (e.g. `Pinger` on the health service).

**Consequences:**

- Graceful shutdown drains the HTTP server, then disconnects Mongo (10s timeout today).
- `http.Server` sets `ReadHeaderTimeout` at construction time.
- Adding a new binary (`cmd/seed`) reuses the same packages without duplicating wiring patterns.

---

## ADR-003 — HTTP stack: chi + Huma + humachi

**Status:** Accepted

**Context:** The API needs REST with a stable OpenAPI contract for clients, plus room to mount GraphQL on the same process later.

**Decision:**

| Piece | Role |
| --- | --- |
| **chi v5** | Mux and middleware (RequestID, Recoverer, CORS) |
| **Huma v2** | Operation registration, typed request/response, OpenAPI 3.1, RFC 7807 errors |
| **humachi** | Adapter binding Huma to chi |

Handler flow for business endpoints: parse/validate → authz → service → map errors → response. Prefer Huma problem responses over ad-hoc JSON error maps. Do not introduce Gin or parallel HTTP frameworks.

**Consequences:**

- `/openapi.json` and `/docs` come from Huma registration.
- GraphQL (gqlgen) will mount on the same chi mux and call the same services.
- Error bodies stay consistent for clients; Mongo internals must not leak into responses.

---

## ADR-004 — Liveness vs readiness

**Status:** Accepted

**Context:** Orchestrators need to distinguish “process is alive” from “safe to send traffic.”

**Decision:**

| Endpoint | Meaning | Mongo |
| --- | --- | --- |
| `GET /healthz` | Process liveness | Not contacted |
| `GET /readyz` | Readiness | Ping primary (short timeout in service) |

`/readyz` returns `503` when the ping fails.

**Consequences:**

- Deployments can restart or stop routing independently of process death.
- Health unit tests stub a `Pinger`; `/healthz` stays green even when the stub fails.

---

## ADR-005 — Configuration via environment

**Status:** Accepted

**Context:** Local Docker Mongo and hosted deploys (e.g. Fly) should share one connection-string convention. Config frameworks add surface area for a small service.

**Decision:** Read config from the environment (documented in `.env.example`):

| Variable | Role |
| --- | --- |
| `DATABASE_URL` | Mongo connection URI (required) |
| `PORT` | Listen port (default `8080`) |
| `CORS_ORIGINS` | Comma-separated origins (default `http://localhost:5173`) |

Local Mongo is provided by `docker-compose.yml` (`mongo:7`, named volume, database/user `watchdesk`). Clients talk only to the API URL — never to Mongo directly.

**Consequences:**

- Fail fast on missing `DATABASE_URL`.
- Production can set the same secret name without code changes.
- No Viper (or similar) unless config complexity clearly outgrows env vars.

---

## ADR-006 — Domain IDs and status model

**Status:** Accepted

**Context:** Embedding Mongo types in domain structs couples every layer to the driver and complicates GraphQL/DTO mapping.

**Decision:** Domain entities use string IDs and plain Go types. Repository implementations map to/from BSON and ObjectID. Detection status is a small state machine: `open` → `acked` | `rejected`.

```go
type Detection struct {
	ID         string
	SiteID     string
	SensorID   string
	Status     DetectionStatus
	Severity   DetectionSeverity
	Summary    string
	DetectedAt time.Time
}
```

Ack/reject: idempotent success returns **200** with the same body; illegal transitions return **409**.

**Consequences:**

- Repository interfaces live in `domain`; Mongo implementations live under `repository/mongodb`.
- Acknowledgements are a separate collection — not an unbounded array on the detection document. See [`data-model.md`](./data-model.md).

---

## ADR-007 — REST for commands, GraphQL for reads

**Status:** Accepted (GraphQL not implemented yet)

**Context:** Clients need both imperative operator actions (ack/reject) and flexible nested reads (detection → sensor → site).

**Decision:**

| Surface | Use |
| --- | --- |
| REST (Huma) under `/api/v1/...` | Commands: ack, reject, override — HTTP status and idempotency |
| GraphQL (gqlgen) | Reads: nested graph, filters, rollups |

Ack/reject mutations are **not** exposed on GraphQL in v1. Both surfaces call `service`; neither imports the Mongo driver.

**Consequences:**

- OpenAPI remains the contract for command semantics.
- GraphQL schema can evolve nested fields without duplicating command rules.
- Dataloaders and force-resolvers are expected when GraphQL lands, to avoid N+1.

---

## ADR-008 — Authz stub (no JWT in v1)

**Status:** Accepted

**Context:** Role-based behavior (analyst vs supervisor) must be real on the server without standing up an IdP for the demo.

**Decision:** Stub identity via headers:

- `X-Operator-Role: analyst | supervisor`
- `X-Operator-Name: …`

CORS already allows these headers. Constants live in `internal/authz`. Phase 2 requires a valid role on `/api/v1` routes (missing → 401, unknown → 403). Analyst vs supervisor capability splits (override) come in a later phase. UI may hide controls; the API remains authoritative.

**Consequences:**

- No JWT/OAuth dependency in v1.
- Easy to replace the stub with a real identity middleware later without changing service rules.

---

## ADR-009 — Logging

**Status:** Accepted

**Context:** Production debugging needs correlated request logs without a custom logging framework.

**Decision:** Use `log/slog` with JSON to stdout. Chi RequestID plus middleware logs `request_id`, method, path, status, and duration. Expose `X-Request-Id` via CORS.

**Consequences:**

- No dedicated `pkg/logger` unless it stays trivial.
- OpenTelemetry export is out of scope for v1.

---

## ADR-010 — Testing approach

**Status:** Accepted

**Context:** Layers should be testable in isolation; full Mongo should not be required for HTTP contract checks.

**Decision:**

| Layer | Approach |
| --- | --- |
| Unit | Table-driven; mock domain repository (or small) interfaces; no Mongo |
| HTTP | `httptest` against the chi + Huma mux |
| Integration | Compose Mongo; keep the set small |

Phase 0 example: `internal/handler/health_test.go` stubs `Pinger` and asserts `/healthz` vs `/readyz` behavior, plus `/openapi.json`.

**Consequences:**

- Service tests own business rules (including future ack idempotency and conflict cases).
- Testcontainers are not required for v1.

---

## ADR-011 — Mongo data model direction

**Status:** Accepted

**Context:** The primary operator query is “open detections for a site, newest first.” Unbounded nested ack history on the detection document would grow without bound and complicate updates.

**Decision:**

| Collection | Pattern |
| --- | --- |
| `sites`, `sensors` | Referenced documents with independent lifecycles |
| `detections` | Primary work item; compound index `siteId + status + detectedAt` desc |
| `acknowledgements` | Separate audit collection keyed by `detectionId + createdAt` desc |

List pagination will use **keyset** cursors on `(detectedAt, id)`, not `skip`/`limit`.

**Consequences:**

- Indexes are ensured at startup (`EnsureIndexes`, idempotent).
- Rationale for embed vs reference and the separate acks collection: [`data-model.md`](./data-model.md).
- Mongo JSON Schema validation, Redis, change streams, and multi-doc transactions are out of scope for v1 unless explicitly adopted later.

---

## ADR-012 — Explicit non-goals for v1

**Status:** Accepted

**Context:** Scope creep (second frameworks, premature observability, auth theater) slows a demo-ready vertical slice.

**Decision:** Do not introduce for v1 unless a later ADR supersedes this:

- Gin, Viper, Redis, change streams, OpenTelemetry exporter
- JWT / external IdP
- GraphQL mutations for ack/reject
- Multi-document transactions
- Testcontainers as a default CI requirement

**Consequences:** The locked stack stays small and explicit: Huma, chi, official Mongo driver v2, slog, env config.

---

## Package reference (current tree)

| Path | Role |
| --- | --- |
| `cmd/api` | Process entry |
| `cmd/seed` | Demo queue fixture |
| `internal/config` | Env loading |
| `internal/domain` | Entities + repository interfaces |
| `internal/repository/mongodb` | Driver, BSON mapping, indexes, CRUD |
| `docs/data-model.md` | Embed vs reference; ack collection; indexes |
| `internal/service` | Use cases (health, detections list/ack/reject, sites list) |
| `internal/handler` | Huma REST (`/healthz`, `/readyz`, `/api/v1`) |
| `internal/middleware` | Request logging |
| `internal/authz` | Role/header constants |
| `internal/router` | chi + CORS + Huma registration |
| `internal/app` | Composition root |
| `docker-compose.yml` | Local Mongo 7 |

---

## Related

- Product overview and local run: [`README.md`](../README.md)
- Collections and indexes: [`data-model.md`](./data-model.md)
- Agent/contributor layering summary: [`AGENTS.md`](../AGENTS.md)
