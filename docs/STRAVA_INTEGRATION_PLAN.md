# Strava Integration and Learning Plan

Status: planning document for the phase after engine `field-3.2.0`.

## Product goal

Let a first-time user connect Strava, choose one recorded walk, and create one canonical artwork without downloading or uploading a GPX file.

The first Strava-generated artwork must use the same `walk-art-v1` global quality gates as a manual upload. Import history and personal statistics are not prerequisites.

## Current state

Meander currently has:

- a local creation studio with GPX/OSM upload and public sample routes;
- a Go API that parses a route and calls the deterministic generation engine;
- one canonical SVG/PNG result with fingerprint and recipe metadata;
- global cold-start calibration, explicit hero/supporting hierarchy, and composition anchoring;
- no accounts, OAuth, token storage, database, activity library, webhooks, background jobs, or personal learning profile.

The engine boundary is already suitable for Strava. A connector only needs to convert Strava streams into the existing `[]GeoPoint` route model.

## First integration release: one activity to one artwork

### User journey

1. The creation studio presents **Connect with Strava** as the primary import action and **Upload GPX** as the permanent fallback.
2. Before redirecting, Meander explains: “Read your activities to choose a walk. We never post to Strava.”
3. OAuth requests the minimum useful read scope. The callback verifies `state`, exchanges the short-lived code, stores the newest rotating refresh token, and records the scopes actually granted.
4. The user returns to a recent-activity picker filtered to walk/hike-compatible activities.
5. Each card shows title, date, distance, duration, elevation, location summary, privacy indicator, and a lightweight route silhouette when available.
6. The user selects exactly one activity and reviews optional time/music context.
7. Meander fetches only that activity's required streams, converts them to the engine route model, generates one canonical result, and discards the raw stream payload.
8. The result screen shows the artwork, movement fingerprint, Strava attribution, and download actions.

### Scope strategy

- Begin with `activity:read` for activities visible to Everyone or Followers.
- Explain when an Only You activity is unavailable instead of requesting broader access silently.
- Offer a separate, explicit upgrade to `activity:read_all` only when the user chooses to include private activities.
- Do not request `activity:write`, profile-write, or unrelated scopes.
- Always verify the scopes returned by the OAuth callback because a user can decline individual scopes.

### Data requested for a selected activity

Required:

- `latlng` stream for route geometry.

Useful when present:

- `time` for pauses and pace changes;
- `altitude` for climb/descent events;
- `distance` and `velocity_smooth` for validation and movement analysis;
- activity title, sport type, start time, distance, duration, elevation gain, and privacy state for the picker and fingerprint.

Heart rate, power, calories, social data, comments, kudos, and write permissions are out of scope.

## Recommended UX states

### Disconnected

- Primary: **Connect with Strava**
- Secondary: **Upload GPX instead**
- Trust copy: read-only, one selected activity, disconnect anytime.

### Connecting

- Full-page redirect rather than an embedded webview.
- Preserve the intended return path and draft context in a signed `state` value.

### Activity picker

- Default to recent eligible walks, not an automatic full-history import.
- Filters: Walk, Hike, date range, distance.
- Fetch summaries in pages and streams only after selection.
- Empty state: explain that activities without GPS cannot generate art and preserve GPX upload.

### Review

- Show the selected activity and route silhouette.
- Let the user change the artwork title, time-of-day interpretation, and music inputs.
- Clearly label which values came from Strava and which are creative additions.

### Generation and result

- Keep the current one-output contract.
- Explain the same four stages shown on the landing page.
- Attribute Strava as the activity source without implying Strava created or endorsed the artwork.

### Failure states

- Access denied or missing scope.
- Expired/revoked authorization.
- No eligible GPS activities.
- Selected activity has no `latlng` stream.
- Private activity requires broader permission.
- Strava rate limit reached.
- Temporary upstream failure.

Every failure state preserves the manual GPX path.

## Backend architecture

### Connector boundary

```text
Strava OAuth + API
        │
        ▼
Strava adapter
        │  activity streams → []GeoPoint
        ▼
existing Generate(points, context)
        │
        ▼
canonical artwork + derived fingerprint
```

The generator must not know about Strava tokens, athlete IDs, or history.

### Proposed endpoints

```text
GET  /auth/strava
GET  /auth/strava/callback
POST /auth/strava/disconnect
GET  /api/v1/strava/activities
POST /api/v1/strava/activities/{id}/generate
GET  /api/v1/artworks
```

### Persistence

Store:

- internal user/connection identifier;
- Strava athlete ID;
- encrypted access and refresh tokens;
- token expiry and granted scopes;
- imported activity ID and minimal source metadata;
- derived route features, recipe, and artwork references;
- explicit product feedback and actions used for later learning.

Do not retain:

- raw Strava stream responses;
- a reusable copy of GPS coordinates unless the privacy policy changes and the user explicitly opts in;
- unrelated profile, social, or biometric data.

Token refresh must be concurrency-safe because Strava can rotate the refresh token on every successful refresh. Always persist the newest returned token.

### Rate-limit strategy

- Fetch paginated activity summaries on demand.
- Fetch streams only for the activity the user selected.
- Cache safe summary metadata briefly.
- Read and record Strava rate-limit headers.
- Avoid polling.
- Add webhooks only after the manual import flow is stable.
- Make generation idempotent by athlete ID + activity ID + engine version + creative context.

### Webhooks: later, not MVP

Webhooks can announce created, updated, deleted, and authorization events. The callback must acknowledge quickly and hand work to a queue. A webhook should never automatically generate art without user consent; it should create an “A new walk is ready” inbox item.

Deletion, privacy changes, and deauthorization must revoke access and update or remove locally retained source metadata according to the user's choices.

## Learning patterns without breaking cold start

### Layer 0: global quality, always

Every artwork uses `walk-art-v1` for:

- hero/supporting split requirements;
- negative-space targets;
- composition-anchor safety;
- hierarchy and total-quality thresholds;
- candidate acceptance.

These rules never depend on user history.

### Layer 1: descriptive archive, walks 1–9

Store only derived measurements and show descriptive patterns such as:

- loop-heavy versus directional routes;
- frequent turn or pause structures;
- typical distance and elevation range;
- recurring time-of-day or music context.

Do not alter generation from these early observations. The product can learn enough to explain, but not enough to personalize responsibly.

### Layer 2: optional personal emphasis, 10+ walks

With consent and enough history, build a versioned personal pattern profile from derived features—not raw GPS history. It may influence only bounded emphasis decisions, such as:

- which globally eligible trail becomes a hero;
- which meaningful movement event becomes the preferred anchor;
- tie-breaking between candidates that already pass every global quality gate.

Personal influence should be capped, recorded in the recipe, reversible, and switchable off. It must not change negative-space requirements, remove hierarchy, lower the total-quality threshold, or rescue a globally failing candidate.

Conceptually:

```text
generate candidates
      ↓
apply walk-art-v1 hard quality gates
      ↓
retain globally valid candidates/trails
      ↓
optional bounded personal tie-break
      ↓
one canonical artwork
```

### Signals worth learning

Use deliberate product behavior rather than assuming that every imported walk expresses preference:

- download or share;
- favorite/save;
- explicit “this feels like my walk” feedback;
- which candidate characteristics were present in accepted artworks;
- user corrections to title, time, or music context.

Avoid opaque engagement optimization. A user should be able to see and reset the learned profile.

## Delivery phases

### S1 — Developer and security foundation

- Register the Strava application and configure local/production callback domains.
- Add environment configuration and encrypted token storage.
- Implement OAuth state, callback, scope verification, refresh, disconnect, and revocation handling.

### S2 — Activity picker vertical slice

- List recent eligible activities.
- Build activity cards, filtering, pagination, empty states, and GPX fallback.
- Test public/follower activity access first.

### S3 — Selected-activity generation

- Fetch selected streams, validate `latlng`, adapt to `[]GeoPoint`, and call v3.2.
- Discard raw streams and store only derived output.
- Add idempotency and complete error handling.

### S4 — Artwork library and privacy controls

- Add a user's generated-work archive.
- Add connection status, disconnect, data deletion, and source provenance.

### S5 — Webhook inbox

- Receive new-activity notifications and queue asynchronous enrichment.
- Ask the user before generating.

### S6 — Pattern learning experiment

- Collect explicit feedback and derived features.
- Require at least 10 eligible artworks before calculating personal emphasis.
- Evaluate the bounded tie-breaker offline before enabling it in production.

## MVP acceptance criteria

- A brand-new user can connect Strava and generate from one walk without prior Meander history.
- The OAuth flow requests only documented read access and verifies granted scopes.
- Only the selected activity's streams are fetched.
- An activity with `latlng`, optional time, and optional altitude maps correctly to the existing engine.
- Missing GPS, denied access, revoked tokens, refresh rotation, rate limiting, and upstream errors have recoverable UX.
- Raw streams are discarded after generation.
- The recipe records Strava provenance without exposing athlete or activity identifiers in the artwork.
- The result passes the same `walk-art-v1` cold-start gates as GPX upload.
- Manual GPX upload remains available throughout.

## Official references

- Authentication: https://developers.strava.com/docs/authentication/
- API reference and activity streams: https://developers.strava.com/docs/reference/
- Webhooks: https://developers.strava.com/docs/webhooks/
- Rate limits: https://developers.strava.com/docs/rate-limits/
- Brand guidelines: https://developers.strava.com/guidelines/
- API changelog: https://developers.strava.com/docs/changelog/
