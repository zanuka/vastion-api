# com-scan-api

![com-scan](images/comm-scan-api.jpg)

Go API for **com-scan** — a fictional bridge watchdesk that fuses ship and outpost sensors into a human-in-the-loop triage queue.

## Product vision

In Star Wars lore, a **com-scan** integrated the ship’s many communication and sensor systems — electro-photo receptors, full-spectrum and subspace transceivers, dedicated energy receptors, and more — into one place where skilled officers (and sorting droids) evaluated the flood of readings and decided what was important enough to pass to command. Conditioned alerts could wake reserve sensors for a closer look: power fluctuations, energy spikes behind stealth, anything that matched a warning profile.

This repo borrows that **shape**, not the setting. Civilian **com-scan** is the watchdesk that:

1. **Integrates** heterogeneous sites and sensors into one coherent picture  
2. **Collates and prioritizes** detections so operators are not drowning in raw feeds  
3. **Triages with humans in the loop** — acknowledge, reject, or override before anything escalates further  

Same job as the bridge console: fuse sensors → surface what matters → let people decide. Not a DefenseTech clone; the workflow is the proof. Domain focus: awareness → triage → human judgment.

## Product metaphor

A **Com-Scan** for operators:

**sites → sensors → prioritized detections → acknowledge / override**

| Entity | Role |
| --- | --- |
| **Site** | Location / area of operations |
| **Sensor** | Source of detections |
| **Detection** | Primary work item (severity, confidence, status) |
| **Ack** | Operator action: acknowledge, reject / false-positive, override |

Status stays small: `open` → `acked` | `rejected`. Commands are idempotent; illegal transitions fail clearly.

## This repo

`com-scan-api` is the **server of record** and the contract other systems consume. The first client is [`com-scan-vue`](https://github.com/zanuka/com-scan-vue), but the API is not Vue-specific.

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

## Local development

Mongo in Docker (OrbStack or Docker Desktop):

```
docker compose up -d
```

Copy `.env.example` and run the API:

```
export DATABASE_URL=mongodb://watchdesk:watchdesk@localhost:27017/watchdesk?authSource=admin
export PORT=8080
export CORS_ORIGINS=http://localhost:5173
go run ./cmd/api
```

- `GET /healthz` — process is up
- `GET /readyz` — Mongo ping
- `/openapi.json` and `/docs` — Huma OpenAPI

Env: `DATABASE_URL` (required), `PORT` (default `8080`), `CORS_ORIGINS` (comma-separated; default `http://localhost:5173`). Later, Fly can set the same `DATABASE_URL` with `fly secrets set`.

Layering: `domain` → `repository/mongodb` → `service` → `handler`. GraphQL (gqlgen) is a later phase.
