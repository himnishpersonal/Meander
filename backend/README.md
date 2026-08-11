# Meander Go Engine

The Go backend parses recorded routes, extracts a movement fingerprint, builds and scores deterministic route-turbulence compositions, renders one canonical SVG/PNG result, and exposes the process through a local HTTP API.

Current engine: `field-3.2.0`  
Calibration profile: `walk-art-v1`

## Design invariants

1. The route is the primary structural signal.
2. The route is never rendered as a visible GPS trace.
3. Every request produces exactly one canonical artwork.
4. The same route, context, engine version, and calibration version produce the same result.
5. Global composition quality applies from the first upload.
6. Generation accepts no user history or personal baseline.

## Generation pipeline

```text
Parse GPX/OSM
  → validate coordinates
  → project to local meters
  → resample to 180 points
  → smooth and normalize
  → measure route features
  → detect local events
  → derive deterministic seed
  → generate 28 turbulence drafts
  → choose anchor and classify trails
  → score with walk-art-v1
  → select winner
  → render SVG + PNG + JSON metadata
```

## Core packages

- `internal/engine/engine.go` — parsing, normalization, features, seeding, bundle writing, SVG/PNG rendering
- `internal/engine/phase3.go` — turbulence field, anchors, trail hierarchy, color territories, candidate scoring and selection
- `internal/engine/calibration.go` — embedded profile loading and validation
- `internal/engine/calibration/walk-art-v1.json` — global scoring and hierarchy targets
- `cmd/api/main.go` — local HTTP server
- `cmd/walkart/main.go` — command-line generator

`phase2.go` retains legacy composition helpers used by shared scoring utilities; v3.2 generation enters through `buildTurbulenceCandidates` in `phase3.go`.

## Global calibration

`walk-art-v1` supplies:

- coverage and complexity targets;
- negative-space target/tolerance and minimum score;
- directional target/tolerance;
- density and width hierarchy targets;
- hero/supporting counts and width contrast;
- composition-anchor safety and influence radius;
- color dominance and flare targets;
- focal-density targets;
- final scoring weights and minimum total score.

The profile is embedded at build time. No database, network request, account, or user history is needed to score a first walk.

The calibration corpus manifest is in `calibration/corpus/manifest.json`. The `cold-start-single-upload` case points to `fixtures/cold-start-single.gpx` and explicitly records `user_history_count: 0`.

## Route and event model

`GeoPoint` stores latitude, longitude, elevation, and optional timestamp. After normalization, `Point` uses composition-space coordinates.

The fingerprint includes distance, displacement, tortuosity, loop closure, curvature, elevation, timing, pace, turn, and intersection measurements. Local `RouteEvent` values are one of:

- `turn`
- `pause`
- `pace`
- `climb`
- `descent`

These events alter the vector field and provide composition-anchor candidates.

## Composition model

Each visible `Line` receives a semantic role:

- `hero`
- `supporting`
- `ambient`
- `event-fragment`

The hidden `route-guide` has zero opacity and is skipped by both renderers.

The winning recipe records:

- engine and calibration versions;
- deterministic seed and winning draft;
- optional context and palette identity;
- anchor position, source, and strength;
- hero/supporting/ambient trail counts;
- all named quality scores.

## Run

From `backend/`:

```bash
../.tools/go/bin/go run ./cmd/api
```

The API listens on `http://localhost:8080` by default.

Generate from the command line:

```bash
../.tools/go/bin/go run ./cmd/walkart \
  -input fixtures/central-park.osm \
  -out output \
  -title "Central Park" \
  -time morning \
  -tempo 108 \
  -energy 0.54
```

## HTTP API

### Health

```http
GET /healthz
```

### Generate

```http
POST /api/v1/generate
Content-Type: multipart/form-data
```

Accepted form fields:

- `route` — GPX/OSM upload;
- `sample` — fixture filename when no upload is supplied;
- `location_label` — optional;
- `time_of_day` — optional;
- `music_tempo` — optional;
- `music_energy` — optional `0–1` value.

### Artifacts

```http
GET /artifacts/{result-id}/artwork.svg
GET /artifacts/{result-id}/preview.png
```

### Generated samples

```http
GET /api/v1/samples
```

## Output bundle

```text
output/{result-id}/
├── artwork.svg
├── preview.png
├── fingerprint.json
└── recipe.json
```

The raw input route is not included.

## Test

```bash
env GOCACHE=/tmp/walkart-go-cache ../.tools/go/bin/go test ./...
env GOCACHE=/tmp/walkart-go-cache ../.tools/go/bin/go vet ./...
```

Key coverage includes:

- deterministic generation;
- rejection of empty routes;
- one canonical API result with no finalists;
- required named scoring metrics;
- invisible route guide;
- global cold-start calibration;
- hero/supporting split;
- negative-space threshold;
- safe composition anchor;
- minimum total quality.

## Privacy boundary

Uploads are parsed from memory. Generated artifacts and derived metadata are written to the configured output directory; raw uploaded GPX/OSM files are not retained.

Future Strava support belongs in an adapter outside the engine package. The adapter will convert selected activity streams into `[]GeoPoint`; the generator will not receive OAuth tokens, athlete IDs, account history, or Strava-specific types.

For the full conceptual explanation, see [`../docs/ART_GENERATION_ENGINE.md`](../docs/ART_GENERATION_ENGINE.md).
