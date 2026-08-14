# vastion-api

![vastion](images/vastion-api.jpg)

Go API for **Vastion** — a fictional station watchdesk that fuses ship and outpost sensors into a human-in-the-loop triage queue.

## Product vision

A nod to how Star Wars bridges and space stations fused many sensor and communication feeds — electro-photo receptors, full-spectrum and subspace transceivers, dedicated energy receptors, and more — into one place where skilled officers (and sorting droids) evaluated the flood of readings and decided what was important enough to pass to command. Conditioned alerts could wake reserve sensors for a closer look: power fluctuations, energy spikes behind stealth, anything that matched a warning profile.

**Vastion** borrows that **shape** for a fictional fleet watchdesk that:

1. **Integrates** sites and sensors across ships and outposts into one coherent picture  
2. **Collates and prioritizes** detections so bridge crews are not drowning in raw feeds  
3. **Triages with humans in the loop** — acknowledge, reject, or override before anything escalates further  

Same job as the station console: fuse sensors → surface what matters → let people decide. Domain focus: detection → triage → human judgment.

## Product metaphor

A **Vastion** for operators:

**sites → sensors → prioritized detections → acknowledge / override**

| Entity | Role |
| --- | --- |
| **Site** | Location / area of operations |
| **Sensor** | Source of detections |
| **Detection** | Primary work item (severity, confidence, status) |
| **Ack** | Operator action: acknowledge, reject / false-positive, override |

Status stays small: `open` → `acked` | `rejected`. Commands are idempotent; illegal transitions fail clearly.

## This repo

`vastion-api` is the **server of record** and the contract other systems consume. The first client is [`vastion`](https://github.com/zanuka/vastion), but the API is not Vue-specific.

Clients and services may include:

- Ops UIs (Vue today; others later)
- Automation / workers that list, filter, or act on detections
- Integrations that read the graph or issue command-style mutations over HTTP

Contracts are the source of truth — **OpenAPI (Huma REST)** for command mutations (ack, reject, assign) and **GraphQL (gqlgen)** for the read graph (detection → sensor → site, filters, rollups). Vue never talks to Mongo; only this API does.

## North star

Ship a thin vertical slice early: detections list → detail → ack, with honest async and error states. Then layer authz, GraphQL, indexes/aggregation, and HITL UX. Every surface should answer: *why this tool, what state lives where, what the operator can do next.*

```
Client need → contract (REST command or GraphQL graph) → Go handler/resolver → Mongo → typed client → UI / service states
```

## Why this stack

Vastion is deliberately the same shape as a real watchdesk, at a smaller scale: sites and sensors feed a shared picture; operators triage detections with a clear status machine; the Vue client never talks to Mongo; this Go API owns the contracts and the persistence.

### Why Go

The decision loop stays close to the sensors and operators instead of shipping every frame or detection back to a distant cloud that may not exist.

A watchdesk or perception node must ingest concurrent sensor streams, run inference side-effects, handle partial network partitions, and still respond to operator commands. Go’s CSP model makes that tractable without the thread explosion or callback hell of other languages.

- **Performance and footprint.** Low memory, fast startup, predictable latency. Edge devices and forward-deployed kits do not have the headroom of a cloud region.
- **Networking and API ergonomics.** The standard library plus mature frameworks — Huma for REST/OpenAPI, gqlgen for GraphQL — give clean contracts that can stay a phase ahead of the Vue client. Go is also heavily used in security tooling and mesh/networking code for the same reasons.
- **Operational simplicity under contested links.** You can run the service locally, buffer state, and reconcile when the link returns. No heavy framework magic that assumes always-on connectivity.

### Why MongoDB

Sensor and detection data is heterogeneous and mission-shaped, not relational-table-shaped.

- **Flexible document model.** A Detection can carry severity, confidence, a status machine (`open` → `acked` | `rejected`), nested history of operator overrides, model provenance, geospatial context, and arbitrary sensor-specific payloads without constant schema migrations. That matches how real multi-source feeds arrive (imagery metadata + FMV tracks + model scores).
- **Write-heavy, append-friendly workloads.** Sites and sensors continuously emit detections; operators triage them. Mongo handles high ingest rates and secondary indexes on the fields that matter for the shared picture (status, severity, site, time, confidence).
- **Edge-friendly deployment.** A local Mongo (or compatible store) on the node can keep a working set of open detections and recent models, then sync/reconcile when connectivity allows. Document-oriented storage maps cleanly to the “what was seen, how sure we are, who owns the next move” shared-awareness model.
- **Natural fit with Go.** The official driver and BSON are straightforward. API contracts (OpenAPI + GraphQL schema) stay the source of truth while the storage layer remains flexible.

Relational systems force rigid tables or endless JSON columns when the shape of a Detection or an Ack changes with the mission. Mongo lets the domain model stay honest.

### Why Vue 3

Operators need a reactive, low-friction SPA for triage: a shared picture of sites, sensors, and detections; confidence visualization; ack / reject / override actions that must be idempotent and survive delayed links. Vue 3 + Composition API + a design system is a strong fit for that UX surface. The client lives in [`vastion`](https://github.com/zanuka/vastion); this API stays client-agnostic.

### Mapping back to the vision

Vastion is the same loop at a smaller scale. Working through the two-repo boundary, Huma/gqlgen, the status machine (including proper 409s on illegal transitions), and the edge-resilient patterns is the work real perception-and-command platforms do — fuse sensors, surface what matters, let people decide when the link is contested.

## Local development

Prerequisites: Go 1.24+, Docker (OrbStack or Docker Desktop), and [golangci-lint](https://golangci-lint.run/welcome/install/) v2.

Mongo in Docker:

```
docker compose up -d
```

Copy `.env.example` to `.env`. Then:

```
make compose-up
make seed
make run
```

Or without Make:

```
export DATABASE_URL=mongodb://watchdesk:watchdesk@localhost:27017/watchdesk?authSource=admin
export PORT=8080
export CORS_ORIGINS=http://localhost:5173
go run ./cmd/seed
go run ./cmd/api
```

`make seed` resets the four collections and inserts a demo queue (mixed severity and status, including acked and rejected rows).

- `GET /healthz` — process is up
- `GET /readyz` — Mongo ping
- `/openapi.json` and `/docs` — Huma OpenAPI

Env: `DATABASE_URL` (required), `PORT` (default `8080`), `CORS_ORIGINS` (comma-separated; default `http://localhost:5173`). Later, Fly can set the same `DATABASE_URL` with `fly secrets set`.

Layering: `domain` → `repository/mongodb` → `service` → `handler`. GraphQL (gqlgen) is a later phase.

### Make targets

| Command | What it does |
| --- | --- |
| `make run` | Start the API (`go run ./cmd/api`) |
| `make seed` | Reset collections and insert the demo queue |
| `make compose-up` | Start Mongo via Docker Compose |
| `make build` | Compile all packages |
| `make test` | Run `go test ./...` |
| `make lint` | Run `golangci-lint run` |
| `make check` | Build, test, and lint (same gates as CI and the pre-push hook) |
| `make hooks` | Install `.git/hooks/pre-push` (also happens on any `make` target) |

### Git hooks

Git only runs scripts in `.git/hooks/` (that directory is not committed). Any `make` target installs a symlink from `.git/hooks/pre-push` to `.githooks/pre-push`. To install without running another target:

```
make hooks
```

After that, `git push` runs `make check` so the branch compiles, tests pass, and lint is clean before anything reaches the remote. A failing check aborts the push. Skip only in an emergency with `git push --no-verify`.

## Author

Created by [zanuka](https://github.com/zanuka) (Michael Delucchi)

## License

Copyright © 2026 Michael Delucchi. Released under the [MIT License](LICENSE).
