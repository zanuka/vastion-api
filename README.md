# baluardo-api

![baluardo](images/baluardo-api.jpg)

Go API for **Baluardo** (Italian for *bulwark*) — a hosted watchdesk game. You are the duty officer on a multi-leg **Quest**: sites and sensors across a delayed fleet feed an imperfect picture while you keep the spacecraft alive long enough to reach the next docking station. Operators triage detections, type commands, and push decisions back to the edge. Latency, confidence, dwindling supplies, and the cost of being wrong are the gameplay.

The Vue client lives in the sibling [`baluardo`](https://github.com/zanuka/baluardo) repo. Vue is a projection and input surface. This API owns session state, supply ticks, latency simulation, feed generation, and debrief scoring. Vue never talks to Mongo.

## Product vision

Most command games hide uncertainty. Baluardo makes it the gameplay. The longer container is a **Quest** — an Oregon Trail–style interplanetary journey. While the operator triages detections under light-minute delay, the craft also burns propellant, O₂, rations, and spare parts, and knowledge about galaxies, systems, and planets unlocks because they arrived.

You issue an Ack. The edge unit only receives it after a delay that stands in for light-minutes. In the meantime the Detection can shift confidence, move, or vanish — and the craft may already have moved. At each Waypoint the player restocks, optionally plays a short mini-game, and unlocks lore. A Quest ends with a debrief: truth vs decisions, delay and resource cost, knowledge discovered, final score.

This API is the **server of record** for that loop:

1. **Integrates** sites and sensors into one coherent (and always partial) picture
2. **Collates and prioritizes** detections so the watch is not a raw feed
3. **Triages with a human in the loop** — acknowledge, reject, or override before anything reaches the edge
4. **Simulates latency** — commands travel; the world can change while they are in flight
5. **Ticks survival** — SupplyState lives here; journey actions that would leave the craft dead fail clearly
6. **Withholds ground truth** until debrief — scoring (judgment + survival + knowledge) lives here, not in the client

**First playable fantasy:** a 4–5 Waypoint Quest that can be finished in 15–25 minutes. Outfit at origin, travel and triage under delay, restock once, one Signal Intercept mini-game, one knowledge discovery, then reach the final station or fail with a readable debrief. A short single-watch **Scenario** remains as training mode. Co-op shared picture comes after that loop is fun.

## Product metaphor

**quest → outfit → watchdesk (detections + command console) → waypoint (restock / mini-game / knowledge / triage) → delayed edge receipt → next leg → debrief**

| Entity | Role |
| --- | --- |
| **Site** | Location / AO (sector, station, docking waypoint), with a latency profile |
| **Sensor** | Source of detections; coverage and reliability |
| **Detection** | Primary work item (severity, confidence, status, freshness). Truth is hidden until debrief. |
| **Ack** | Player command: acknowledge, reject / false-positive, override |
| **Quest** | Campaign definition: Waypoint sequence, starting loadout, educational themes, win/lose conditions |
| **Waypoint** | Landmark on the journey — local sensors, restock, lore, optional mini-game |
| **SupplyState** | Authoritative resource bag for the session (propellant, O₂, rations, parts, crew). Tick rules live here |
| **Command** | Console input, parsed into Ack, journey action, query, or mini-game trigger |
| **MiniGameResult** | Outcome reported by Vue; this API applies the resource delta and records it for debrief |
| **KnowledgeNode** | Facts / lore about a galaxy, system, or planet (seeded here or served via instruments) |
| **Scenario** | Layout, threat mix, duration, and latency for one short watch (training mode) |
| **Session** | A running (or completed) play of a Quest or Scenario |
| **Debrief** | After-action: truth vs decisions, delay cost, resource trajectory, knowledge discovered |

Status stays small: `open` → `acked` | `rejected`. Commands are idempotent; illegal transitions fail clearly (expect 409). Journey actions that would leave the craft dead fail the same way. That constraint is a game rule.

## This repo

Module: `github.com/zanuka/baluardo-api`. This service owns persistence and the contracts other systems consume. The first client is [`baluardo`](https://github.com/zanuka/baluardo), but the API is not Vue-specific.

Clients and services may include:

- The Baluardo SPA (Vue today; others later)
- Automation / workers that list, filter, or act on detections
- Integrations that read the graph or issue command-style mutations over HTTP

Contracts are the source of truth — **OpenAPI (Huma REST)** for commands (ack, reject, start Quest / session, journey actions, mini-game results), a **WebSocket observation plane** for the live watch (G1 may poll REST as a bridge), and **GraphQL (gqlgen)** later for the read graph (quest layout, nested debrief, filters, rollups). Vue never talks to Mongo; only this API does. Vue never authors detections or supply ticks; the feed generator and resource clock live here.

## North star

Ship a playable Quest early: start a campaign → outfit → live detections + command console → ack / reject and ADVANCE under delay → restock / mini-game at a Waypoint → debrief that shows truth vs decisions and the resource trajectory. Keep the existing detection/ack path working the whole time. Then layer the event stream, knowledge instruments, co-op rooms, content, and hosting.

```
Client need → contract (REST command, WS event, or GraphQL read)
           → Go handler / stream / resolver → Mongo → typed client → UI / service states
```

Every surface should answer: *why this tool, what the player can know now, what is still in flight to the edge, and what the craft still has left.*

## Quest gameplay: API and Mongo

The watchdesk loop (sites, sensors, detections, ack/reject, delayed receipt) does not change. The Quest is a longer container around that loop. Domain types stay in `internal/domain` with string IDs; only `repository/mongodb` maps them to ObjectIDs. Contracts stay a phase ahead of the Vue screens that consume them.

### What this API owns

| Concern | Rule |
| --- | --- |
| **SupplyState** | Authoritative bag on the Session. Vue displays it; it never writes it. Tick, restock, ration, and mini-game deltas are applied in `service`. |
| **Journey commands** | REST mutations (`ADVANCE`, `DOCK`, `REST`, `RATION`, …). Illegal state — including an ADVANCE that would leave the craft dead — returns a clear 4xx, same spirit as ack **409**. |
| **Mini-games** | Vue owns the interaction. It POSTs a `MiniGameResult`; this API applies the resource delta and records the outcome for debrief. |
| **Latency** | Commands that affect edge state still travel the delayed receipt path. `STATUS`, `SUPPLIES`, and knowledge queries can be immediate. |
| **Knowledge** | Seeded `KnowledgeNode` documents (and later instrument tools) are read-only flavor. They do not mutate game state. |
| **Debrief** | Written here: truth vs decisions, delay cost, supply trajectory, knowledge discovered, score. Ground truth never appears on mid-session list, detail, or stream payloads. |

Ack/reject stay on REST and remain idempotent. A Quest Session still surfaces live detections; the existing status machine is the atomic unit. Instruments (command parsing, lore lookup) may sit beside this service later; they are not a second source of truth. Game mutations still go through this API.

### How Mongo should hold it

Follow the same embed-vs-reference rules as the watchdesk collections (see [`docs/data-model.md`](docs/data-model.md)): **bounded current state on the parent document, unbounded history in its own collection, independent lifecycles referenced by id.**

| Collection (direction) | Pattern | Why |
| --- | --- | --- |
| `quests` | Definition document: ordered Waypoints, starting loadout, themes, win/lose. Referenced by sessions. | A Quest template outlives many plays, the way a Site outlives detections. The Waypoint list is small and bounded (4–6 for the first playable), so it embeds on the Quest rather than becoming a hot collection of its own. |
| `sessions` | Running play: `questId`, current Waypoint index, **embedded SupplyState**, status (`running` \| `completed` \| `failed`). | Current bag and current leg are one snapshot — same reason current detection `status` lives on the detection, not in the ack log. One document update applies a tick or restock without a multi-document transaction. |
| `commands` | Immutable row per console verb (operator, raw text, parsed action, result, time). | Unbounded audit, like `acknowledgements`. Do not append an array onto the session. |
| `minigame_results` | Immutable row: session, waypoint, game kind, accuracy/outcome, applied delta. | Vue reports; API records. Debrief reads this log. |
| `knowledge_nodes` | Seeded facts keyed by system / planet / topic. Waypoints reference them. | Content lifecycle is independent of a running watch. |
| Existing four | `sites`, `sensors`, `detections`, `acknowledgements` unchanged. | Docking stations can be Sites (or Waypoints that reference a Site). Detections stay the queue; a Quest Session filters them like today’s site switcher. |

**Tick on command, not on the client clock.** `ADVANCE` (and rest / ration) is the moment `service` subtracts consumables, checks win/lose, and optionally injects a journey event or new Detection. That keeps the resource clock authoritative even if the SPA is paused or reconnecting.

**Two writes, same as ack.** Update the session’s current SupplyState, then insert a command or mini-game row. Do not grow the session document with history arrays. Mongo JSON Schema validation, Redis, change streams, and multi-doc transactions stay out of scope unless an ADR says otherwise.

Seed the first playable as a handful of Quest + KnowledgeNode documents next to the existing demo queue. Indexes to defend later: `sessions` by operator + status, `commands` / `minigame_results` by `sessionId` + time descending — the same shape as acknowledgements by `detectionId`.

## Why this stack

Baluardo is deliberately the same shape as a real watchdesk, at a smaller scale: sites and sensors feed a shared picture; operators triage detections with a clear status machine; a Quest session carries an authoritative supply bag; the Vue client never talks to Mongo; this Go API owns the contracts, the persistence, and the delayed edge.

### Why Go

The decision loop stays close to the sensors and operators instead of shipping every frame or detection back to a distant cloud that may not exist.

A watchdesk or perception node must ingest concurrent sensor streams, run inference side-effects, handle partial network partitions, and still respond to operator commands. Go’s CSP model makes that tractable without the thread explosion or callback hell of other languages.

- **Performance and footprint.** Low memory, fast startup, predictable latency. Edge devices and forward-deployed kits do not have the headroom of a cloud region.
- **Networking and API ergonomics.** The standard library plus mature frameworks — Huma for REST/OpenAPI, gqlgen for GraphQL — give clean contracts that can stay a phase ahead of the Vue client. A session-scoped generator and WebSocket hub fit the same process.
- **Operational simplicity under contested links.** You can run the service locally, buffer state, and reconcile when the link returns. Latency simulation is a first-class mechanic, not a bug.

### Why MongoDB

Sensor and detection data is heterogeneous and mission-shaped, not relational-table-shaped.

- **Flexible document model.** A Detection can carry severity, confidence, a status machine (`open` → `acked` | `rejected`), nested history of operator overrides, model provenance, geospatial context, and arbitrary sensor-specific payloads without constant schema migrations.
- **Write-heavy, append-friendly workloads.** Sites and sensors continuously emit detections; operators triage them. Mongo handles high ingest rates and secondary indexes on the fields that matter for the shared picture (status, severity, site, time, confidence).
- **Scenarios and sessions.** Scenario definitions, running watches, and debrief snapshots sit next to the queue without a second store.
- **Natural fit with Go.** The official driver and BSON are straightforward. API contracts stay the source of truth while the storage layer remains flexible.

Relational systems force rigid tables or endless JSON columns when the shape of a Detection or an Ack changes with the mission. Mongo lets the domain model stay honest.

### Why Vue 3

Operators need a reactive, low-friction SPA for triage: a shared picture of sites, sensors, and detections; a canvas situation map; ack / reject / override actions that must be idempotent and survive delayed links. Vue 3 + Composition API is a strong fit for that UX surface. The client lives in [`baluardo`](https://github.com/zanuka/baluardo); this API stays client-agnostic.

### Mapping back to the vision

Baluardo is the same loop at a smaller scale. Working through the two-repo boundary, Huma/gqlgen, the status machine (including proper 409s on illegal transitions), and delayed edge receipts is the work real perception-and-command platforms do — fuse sensors, surface what matters, let people decide when the link is contested.

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
- `GET /api/v1/sites` — thin site list for the switcher (`id`, `name`, `code`)
- `GET /api/v1/detections` — queue with `site`, `severity`, `status` filters and a keyset `cursor`
- `GET /api/v1/detections/{id}` — detection detail
- `POST /api/v1/detections/{id}/ack` — idempotent **200** with the updated detection; **409** on an illegal transition (for example `rejected` → `acked`)
- `POST /api/v1/detections/{id}/reject` — `{"reason":"..."}` required (**400** if missing); same 200 / 409 rules as ack

Send stub identity headers on `/api/v1` routes: `X-Operator-Role: analyst|supervisor` and `X-Operator-Name`. Missing role is **401**; an unknown role is **403**. Vue should set `VITE_API_URL=http://localhost:8080`.

Env: `DATABASE_URL` (required), `PORT` (default `8080`), `CORS_ORIGINS` (comma-separated; default `http://localhost:5173`). Later, Fly can set the same `DATABASE_URL` with `fly secrets set`.

Layering: `domain` → `repository/mongodb` → `service` → `handler`. GraphQL (gqlgen) is a later phase. Scenario, session, debrief, and the observation stream land on the game track before the Vue screens that consume them.

### Make targets

| Command | What it does |
| --- | --- |
| `make run` | Start the API (`go run ./cmd/api`) |
| `make seed` | Reset collections and insert the demo queue |
| `make compose-up` | Start Mongo via Docker Compose |
| `make build` | Compile all packages |
| `make test` | Run `go test ./...` |
| `make test-integration` | Run tests including the Compose Mongo ack path (`DATABASE_URL` required) |
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
