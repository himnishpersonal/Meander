# Meander Art Generation Engine

This document explains how engine v3.2 (`field-3.2.0`) turns one recorded route into one deterministic abstract artwork.

## Product rule

The route must remain the primary source of the composition. It first becomes a force skeleton: its direction, turns, pauses, pace changes, climbs, and descents bend a larger field of marks. A low-opacity, interrupted route-memory layer is then rendered over that field so the source walk remains discoverable without becoming a conventional map trace.

The engine creates 28 private drafts and returns only the highest-scoring one. The user receives one canonical result—not a picker full of alternatives. Every quality decision uses the bundled global `walk-art-v1` calibration profile; no user history is required or accepted.

## Pipeline

```text
GPX / OSM route
      │
      ▼
parse and validate coordinates
      │
      ▼
project → resample → smooth → normalize
      │
      ├── measure route-wide features
      └── detect local movement events
      │
      ▼
derive deterministic seed from route + context + engine version
      │
      ▼
build 28 route-turbulence drafts
      │
      ▼
score composition, direction, color, and focal structure
      │
      ▼
select one canonical draft
      │
      ▼
apply role-specific mark textures + subtle route memory
      │
      ▼
SVG + PNG + fingerprint + recipe
```

## 1. Inputs

### Required

- **Route geometry:** ordered latitude/longitude points from a GPX or OSM XML file.

### Optional

- **Location label:** changes the title and participates in the deterministic seed.
- **Time of day:** shifts the base hue family used by the palette.
- **Music tempo:** participates in the deterministic identity. It is reserved for deeper rhythmic modulation in a later engine version.
- **Music energy:** increases the field's wildness and color chroma.

The raw uploaded route is processed in memory and is not copied into the output bundle.

## 2. Route normalization

Raw GPS coordinates are not suitable drawing coordinates. The engine therefore:

1. rejects invalid or duplicate points;
2. projects latitude and longitude into local planar meters;
3. resamples the route to 180 evenly spaced points;
4. smooths small GPS noise;
5. scales the route into a normalized composition space with padding.

This means a short neighborhood walk and a long trail can use the same generation system while retaining their own proportions and geometry.

## 3. Movement fingerprint

The engine measures route-wide features such as:

- distance and displacement;
- loop closure and tortuosity;
- mean and maximum curvature;
- hard turns and self-intersections;
- aspect ratio;
- elevation gain;
- duration, mean pace, pauses, and pace changes when timestamps exist.

It also creates local events positioned on the normalized route:

- `turn`
- `pause`
- `pace`
- `climb`
- `descent`

Route-wide features set the overall character. Local events create disruptions at specific parts of the walk.

## 4. Deterministic identity

The engine hashes the normalized route data, optional context, and engine version into a stable seed. Every pseudo-random decision is made from that seed.

Therefore:

- the same route, context, and engine version produce the same artwork;
- changing time, music characteristics, or the route can produce a different artwork;
- changing the engine version intentionally creates a new visual identity.

Deterministic does not mean visually simple. It means the apparent randomness is reproducible.

## 5. Route-turbulence field

Each visible mark is a particle trace through a vector field. At a point `p`, the direction is conceptually:

```text
V(p) = local route direction
     + global start-to-end direction
     + multi-scale curl noise
     + nearby event forces
```

The components have different jobs:

- **Local route direction** preserves the sense of where the walk turns. Its influence becomes stronger near the route and weaker farther away.
- **Global direction** gives the composition an overall travel axis. Closed loops fall back to a perpendicular local direction because their start and end nearly coincide.
- **Multi-scale curl** combines large bends, medium eddies, and fine nervous motion. This supplies controlled wildness.
- **Event forces** create vortices, compression, expansion, vertical lift, and fracture around meaningful movement moments.

The route is also retained as a scoring guide. In the final render it becomes a narrow, low-opacity, interrupted route-memory mark. This is intentionally quieter than the hero and supporting currents: it should become visible after a moment of looking, not read as a map at first glance.

## 6. Seeds, collisions, and fractures

A draft begins with hundreds of particles:

- roughly half start near the route;
- roughly one quarter start around detected events;
- the rest begin outside the frame so currents can enter and leave the composition.

Each particle advances through the vector field for a bounded number of steps. A small occupancy grid stops ordinary marks from endlessly piling into the same area. Some thicker structural marks may cross dense areas, creating visual hierarchy.

High-stress event zones can terminate a trace early. Short colored fragments are added around strong events. These breaks make the work feel disrupted and expressive without drawing a literal route.

## 7. Mark texture and path legibility

Engine v3.2 assigns texture by compositional role instead of treating every line as the same material:

| Role | Texture | Visual job |
|---|---|---|
| Hero trails | Layered ribbon | Establish the dominant current and visual hierarchy |
| Supporting trails | Broken marks and dotted threads | Reinforce direction without competing with the hero |
| Ambient trails | Hairlines and dry-brush gaps | Supply complexity, atmosphere, and depth |
| Event fragments | Charcoal-like interrupted strokes | Mark turns, pauses, pace changes, climbs, and descents |
| Route memory | Fine long-short dash | Reveal the original walk quietly inside the field |

The SVG renderer creates these materials with layered strokes, different caps, and deterministic dash patterns. The PNG renderer applies corresponding segment masks and layered widths so downloaded formats preserve the same hierarchy.

Texture is deterministic. The same route, context, and engine version always receives the same texture assignments.

## 8. Wildness

Wildness is bounded between `0.42` and `0.94`. It currently grows with:

- music energy;
- mean route curvature;
- self-intersections.

Wildness increases particle count, trace length, curl strength, and fracture probability. The bounds prevent a calm route from becoming empty and a chaotic route from becoming unreadable noise.

## 9. Route-derived color system

The engine generates six semantic color roles in OKLCH color space:

1. background;
2. dark anchor/ink;
3. primary territory;
4. secondary territory;
5. light counterpoint;
6. rare high-chroma flare.

Time of day shifts the base hue region. Music energy increases chroma. Territory centers are attached to several positions and events along the route, so color changes across the composition instead of being assigned independently to every line.

The flare color is deliberately rare. Candidate scoring penalizes both missing accents and excessive accent coverage.

## 10. Private candidate search

One request builds 28 deterministic drafts. Each draft uses a derived seed, producing different particle placement, mark widths, opacity, fractures, and color assignment while keeping the same route forces.

The public API does not return these drafts. It sorts them by total quality score and returns only the winner.

This distinction is important: the engine explores; the product decides.

Before scoring, each draft selects a composition anchor from its strongest movement event, with a curvature fallback for routes without event data. Its trails are then ranked against globally calibrated length, frame span, route affinity, anchor affinity, and width signals. The strongest eligible trails become `hero`; the next layer becomes `supporting`; the remaining trails become `ambient`. These roles exist on the first upload and are recorded in the recipe.

## 11. Candidate scoring

The current total score is a weighted sum loaded from `walk-art-v1`:

| Metric | Weight | Purpose |
|---|---:|---|
| Negative space | 15% | Prevents the artwork from becoming uniformly dense |
| Hierarchy | 12% | Rewards a mix of fine, medium, and structural marks |
| Directional quality | 15% | Preserves motion without rewarding a literal trace |
| Color structure | 14% | Rewards coherent dominant territories |
| Accent discipline | 8% | Keeps the flare color rare but present |
| Focal strength | 9% | Creates at least one meaningful concentration of energy |
| Hero/supporting split | 10% | Requires explicit visual leadership and contrast |
| Anchor strength | 9% | Connects hero trails to a safe composition anchor |
| Edge safety | 4% | Avoids accidental clipping and weak framing |
| Balance | 4% | Prevents all visual mass from collapsing to one side |

Directional quality targets a middle band around `0.68` alignment. Maximum route alignment is not considered ideal: a draft that follows the GPS path too closely should lose to one that preserves direction more abstractly.

The recipe records both the final score and its named submetrics, making the selection explainable.

## 12. Output contract

Each canonical result contains:

- `artwork.svg` — resolution-independent artwork;
- `preview.png` — raster preview;
- `fingerprint.json` — measured route features and local events;
- `recipe.json` — engine and calibration versions, seed, palette identity, winning draft number, hero/supporting counts, composition anchor, context, and scores.

The API also returns artwork URLs and the same fingerprint/recipe metadata. There is no public `finalists` collection.

## 13. Current limitations and next engine work

- Music tempo affects deterministic identity but does not yet directly control spacing or rhythmic grouping.
- GPX timestamp quality varies, so pause and pace events are only as reliable as the source recording.
- Candidate scoring uses engineered visual proxies rather than human preference data.
- Marks remain path-based. Filled regions, translucent washes, grain masks, and stamped primitives could add another level of material contrast.
- The next useful evaluation layer is a route corpus with visual review notes, allowing scoring thresholds to be tuned against real human judgments.

## Code map

- `backend/internal/engine/engine.go` — parsing, normalization, feature extraction, deterministic identity, output rendering, and orchestration.
- `backend/internal/engine/phase3.go` — v3 turbulence generation, color territories, palette construction, candidate scoring, and canonical selection.
- `backend/cmd/api/main.go` — local HTTP API.
- `backend/cmd/walkart/main.go` — command-line generator.
