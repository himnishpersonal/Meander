# Meander

Meander turns the geometry of a recorded walk into one deterministic abstract artwork. The route remains the primary creative signal: it shapes direction, turbulence, color territories, hierarchy, and focal structure, then reappears as a restrained broken “route-memory” line that can be discovered without dominating the abstraction.

Current engine: **`field-3.2.1`**
Global calibration: **`walk-art-v1`**
Product contract: **one walk → one canonical artwork**

Deployment and release plumbing is documented in [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md). Pull requests run the complete frontend, Go, and container checks; production releases are deliberately gated.

## Current capabilities

- GPX and OSM XML route import
- PNG, JPEG, and WebP screenshot tracing as a geometry-only route source
- Public sample routes for local testing
- Route-wide feature extraction and local movement-event detection
- Deterministic route-turbulence generation in Go
- Twenty-eight private drafts per request
- Explicit hero, supporting, ambient, and event-fragment trails
- Role-specific ribbon, broken, thread, dry-brush, and charcoal textures
- A subtle route-memory layer that keeps the source path legible
- Guaranteed composition-anchor selection
- Route-derived six-role OKLCH color system
- Global cold-start calibration from the first upload
- One canonical SVG and PNG result
- Explainable fingerprint, recipe, and named quality scores
- Google accounts, private artwork storage, link sharing, intentional publishing, and an artwork library
- Public gallery containing only artworks their creators explicitly publish
- Dedicated visual walkthrough explaining route interpretation and the artwork lifecycle
- Production creation studio with staged progress and recovery states

Meander does not currently have Strava OAuth, webhooks, or personal learning. Those remain separate future product phases.

## How the engine works

```text
GPX / OSM route
      │
      ▼
validate and clean coordinates
      │
      ▼
project → resample → smooth → normalize
      │
      ├── measure route-wide features
      └── detect local movement events
      │
      ▼
derive deterministic seed
      │
      ▼
build 28 route-turbulence drafts
      │
      ├── select composition anchor
      ├── classify hero/supporting/ambient trails
      └── assign route-derived color territories
      │
      ▼
score every draft with walk-art-v1
      │
      ▼
return one canonical SVG + PNG
```

### 1. Route normalization

The engine rejects invalid and duplicate coordinates, projects latitude/longitude into local planar meters, resamples the route to 180 evenly spaced points, smooths GPS noise, and scales the route into normalized composition space.

### 2. Movement fingerprint

The engine measures:

- distance, displacement, tortuosity, and loop closure;
- mean/max curvature, hard turns, and self-intersections;
- aspect ratio and elevation gain;
- duration, mean pace, pauses, and pace changes when timestamps are available.

It also locates `turn`, `pause`, `pace`, `climb`, and `descent` events along the route.

### 3. Invisible route field

Each visible mark is a particle trace through a vector field:

```text
visible direction = local route direction
                  + global journey direction
                  + multi-scale curl
                  + nearby movement-event forces
```

The local route direction is strongest near the walk and weaker farther away. Turns create vortices, pauses compress the field, pace changes expand it, and elevation events create vertical forces. After composition, the route is rendered as a narrow, low-opacity, interrupted route-memory layer so it can be found without taking over the work.

### 4. Controlled wildness

Wildness is bounded and grows with music energy, route curvature, and self-intersections. It influences particle count, trace length, curl strength, and fracture probability without allowing the composition to collapse into uniform noise.

### 5. Composition hierarchy

Every draft receives a composition anchor from its strongest movement event, with a curvature fallback for simple routes. Trails are ranked using globally calibrated length, frame span, route affinity, anchor affinity, and width signals.

They become:

- `hero` — primary structural movement;
- `supporting` — secondary directional framework;
- `ambient` — surrounding atmosphere;
- `event-fragment` — rare disruptions around meaningful events.

### 6. Route-derived color

The engine creates six semantic colors in OKLCH space: background, anchor ink, primary territory, secondary territory, light counterpoint, and rare high-chroma flare. Time of day shifts the hue region; music energy changes chroma. Color territories attach to positions and events along the walk instead of being assigned randomly per line.

### 7. Private candidate search

One request creates 28 deterministic drafts. The engine scores all of them and returns only the winner. Drafts are never presented as user choices.

## Global cold-start guarantee

Every upload—including a person's first—uses the bundled [`walk-art-v1`](backend/internal/engine/calibration/walk-art-v1.json) global profile.

Hero selection, negative-space targets, anchor safety, hierarchy, and minimum composition quality do **not** depend on:

- user history;
- personal baselines;
- per-user percentiles;
- previous artworks;
- account age.

The calibration corpus contains a dedicated [`cold-start-single-upload`](backend/fixtures/cold-start-single.gpx) case with zero prior history. Its acceptance test verifies hero/supporting separation, negative space, composition anchoring, overall quality, and determinism using only global normalization.

Future personalization may only break ties among globally valid trails or candidates. It may never weaken these quality gates.

## Candidate scoring

Scoring weights are loaded from `walk-art-v1`:

| Metric | Weight | Purpose |
|---|---:|---|
| Negative space | 15% | Prevent uniform density |
| Hierarchy | 12% | Reward structural contrast |
| Direction | 15% | Preserve the walk without tracing it literally |
| Color structure | 14% | Maintain coherent territories |
| Accent discipline | 8% | Keep the flare rare but present |
| Focal strength | 9% | Create meaningful concentration |
| Hero/support split | 10% | Guarantee visual leadership |
| Anchor strength | 9% | Connect hero trails to the composition anchor |
| Edge safety | 4% | Avoid accidental clipping |
| Balance | 4% | Distribute visual mass intentionally |

Directional quality targets a middle alignment band. Maximum alignment is not ideal because an artwork that follows the route too closely becomes a disguised GPS trace.

## Inputs and influence

| Input | Required | Influence |
|---|---|---|
| Route geometry | Yes | Structure, direction, events, fingerprint, deterministic identity |
| Screenshot trace | Alternative | Geometry-only structure and direction; no invented distance, elevation, pace, or time data |
| Location label | No | Display title and deterministic identity |
| Time of day | No | Base hue region |
| Music tempo | No | Deterministic identity; deeper rhythmic modulation is future work |
| Music energy | No | Wildness and palette chroma |

## Run locally

Prerequisites: Node.js 22+ and Go. This workspace can use the ignored local Go toolchain under `.tools/go`.

Terminal 1 — Go API:

```bash
cd backend
../.tools/go/bin/go run ./cmd/api
```

Terminal 2 — website:

```bash
npm install
npm run dev
```

Open:

- Landing page: [http://localhost:3000](http://localhost:3000)
- Creation studio: [http://localhost:3000/create](http://localhost:3000/create)
- How Meander works: [http://localhost:3000/how-it-works](http://localhost:3000/how-it-works)
- Public gallery: [http://localhost:3000/gallery](http://localhost:3000/gallery)
- Engine health: [http://localhost:8080/healthz](http://localhost:8080/healthz)

Nothing needs to be deployed.

When `NEXT_PUBLIC_GOOGLE_CLIENT_ID` and `MEANDER_GOOGLE_CLIENT_ID` are present in the root `.env.local`, the local site uses Google Sign-In and the Go API verifies each identity token before creating a Meander session. A Google client secret is not used for this flow.

## API contract

`POST http://localhost:8080/api/v1/generate` accepts multipart form data:

- `route`: GPX/OSM XML upload, required unless `sample` is supplied;
- `sample`: bundled fixture filename;
- `location_label`: optional display context;
- `time_of_day`: optional atmosphere input;
- `music_tempo`: optional BPM;
- `music_energy`: optional value from `0` to `1`.

Example:

```bash
curl -X POST http://localhost:8080/api/v1/generate \
  -F route=@backend/fixtures/cold-start-single.gpx \
  -F "location_label=First Walk" \
  -F time_of_day=morning \
  -F music_tempo=108 \
  -F music_energy=0.54
```

The response contains one artwork URL, one preview URL, route features, local events, the canonical recipe, and named quality scores. It does not contain finalists.

New artworks are private by default. `PATCH /api/v1/artworks/{id}` can make an owned work `unlisted` for link-only sharing or `public` for gallery discovery. `GET /api/v1/gallery` returns only explicitly public, non-deleted artworks; raw route uploads are never part of that response.

## Output bundle

- `artwork.svg` — resolution-independent canonical artwork
- `preview.png` — raster preview
- `fingerprint.json` — derived route measurements and local events
- `recipe.json` — engine/calibration versions, deterministic seed, palette, winning draft, context, composition anchor, hierarchy counts, and quality scores

Raw uploaded routes are processed in memory and are not copied into the output bundle.

## Validation

```bash
cd backend
env GOCACHE=/tmp/walkart-go-cache ../.tools/go/bin/go test ./...
env GOCACHE=/tmp/walkart-go-cache ../.tools/go/bin/go vet ./...

cd ..
npm run lint
npm test
```

The public validation fixtures represent Central Park, the High Line, Brooklyn Bridge, and Golden Gate Bridge geometry. See [`backend/fixtures/README.md`](backend/fixtures/README.md) for OpenStreetMap attribution. They are engine fixtures, not navigation data.

## Repository map

```text
app/                              website and creation studio
backend/cmd/api/                  local HTTP API
backend/cmd/walkart/              command-line generator
backend/internal/engine/          parsing, generation, scoring, rendering
backend/internal/engine/calibration/
                                  embedded global profile
backend/calibration/corpus/       calibration manifest and cold-start case
backend/fixtures/                 GPX/OSM validation routes
docs/ART_GENERATION_ENGINE.md     full engine explanation
docs/STRAVA_INTEGRATION_PLAN.md   next-phase product and architecture plan
public/generated/                 v3.2 public sample artwork
```

## Next phase

The next major product phase is Strava integration:

```text
Connect Strava → choose one walk → review context → generate one artwork
```

The first release should fetch only the selected activity, adapt its route streams to the existing engine input, discard raw streams after generation, and retain GPX upload as a fallback. Personal learning remains a later, optional layer after enough derived history exists.

See [Strava Integration and Learning Plan](docs/STRAVA_INTEGRATION_PLAN.md).

## Detailed documentation

- [Art Generation Engine](docs/ART_GENERATION_ENGINE.md)
- [Strava Integration and Learning Plan](docs/STRAVA_INTEGRATION_PLAN.md)
- [Backend Engine Guide](backend/README.md)
- [Calibration Corpus](backend/calibration/README.md)
