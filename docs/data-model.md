# Data model

Watchdesk stores four collections. The operator query that matters is **open detections for a site, newest first**. Indexes and document shape follow that path. Mongo JSON Schema validation is out of scope for v1.

## Collections

| Collection | Document | Identity |
| --- | --- | --- |
| `sites` | Location / area of operations | `_id` ObjectID; unique `code` |
| `sensors` | Detection source, owned by a site | `_id` ObjectID; unique `(siteId, code)` |
| `detections` | Primary work item | `_id` ObjectID; references `siteId` + `sensorId` |
| `acknowledgements` | Operator action audit | `_id` ObjectID; references `detectionId` |

Domain types use **string IDs**. `repository/mongodb` is the only layer that maps those strings to ObjectIDs and BSON.

### Site

`name`, unique `code`, `status` (`active` \| `offline` \| `maintenance`).

### Sensor

`siteId`, `name`, `code` unique within the site.

### Detection

Current snapshot of a work item: `siteId`, `sensorId`, `status` (`open` → `acked` \| `rejected`), `severity` (`critical` \| `high` \| `medium` \| `low`), `summary`, `detectedAt`, plus Vue-facing snapshot fields `confidence`, `lastUpdated`, and `provenance` (sensor id/name and a model/version stub). Override audit remains a later HITL pass.

### Acknowledgement

One row per operator action: `detectionId`, `action` (`ack` \| `reject` \| `override`), optional `reason`, `operator`, `fromStatus`, `toStatus`, `createdAt`.

## Embed vs reference

**Sites and sensors are referenced, not embedded on the detection.**

A site outlives thousands of detections. A sensor is reconfigured, renamed, or moved independently of the queue. Embedding a site or sensor document on every detection would duplicate that lifecycle onto the hottest collection and make a rename a multi-document rewrite.

The detection **does** carry `siteId` and `sensorId` (foreign keys, stored as ObjectIDs). That is enough for the queue filter and for later GraphQL dataloaders to resolve nested `sensor` / `site` without a client round-trip. Nested display fields are a read-graph concern, not a reason to copy parent documents.

**Status and severity stay on the detection.** Those values are bounded (one current status, one severity). Embedding the *current* triage state on the work item is the query: list `status=open` for a site. History of how it got there does not belong on that document.

## Why acknowledgements are a separate collection

Acks are an **unbounded** audit log: every ack, reject, and later override is another event. Storing them as an array on the detection would:

- Grow the work-item document without bound (16MB document cap is the hard stop; write amplification is the practical one).
- Couple “update current status” with “append history” on the same document.
- Make “show recent acks on detail” a projection problem instead of a keyed query.

A separate `acknowledgements` collection keeps the detection document small and stable. The detection holds the **current** status; each acknowledgement is an immutable row (`fromStatus` → `toStatus`, operator, time). Detail views load recent acks with `detectionId` + `createdAt` descending. REST ack/reject update that current status on the detection document; they do not append to an array on it.

## Indexes (ensured on startup, idempotent)

| Collection | Index | Purpose |
| --- | --- | --- |
| `sites` | unique `code` | Site switcher and seed upsert-by-code |
| `sensors` | `siteId` | Sensors for a site |
| `sensors` | unique `(siteId, code)` | Code unique per site |
| `detections` | compound `siteId` + `status` + `detectedAt` desc | Queue: open (or any status) for a site, newest first |
| `detections` | `sensorId` | Detections for a sensor |
| `acknowledgements` | `detectionId` + `createdAt` desc | Audit on detail, newest first |

The queue index is the one to defend: equality on `siteId` and `status`, sort on `detectedAt` newest first. That matches “open detections by site, newest first” without a collection scan. Aggregation rollups in a later phase reuse the same compound prefix (`status`, optional `siteId`).

List pagination uses a **keyset** on `(detectedAt, id)`, not `skip`/`limit`. The REST list encodes that pair as an opaque `cursor` query parameter.

## Non-goals here

- Mongo JSON Schema validators on collections
- Multi-document transactions (single-document status update is the ack point)
- Change streams
- Embedding ack history or full site/sensor snapshots on detections
