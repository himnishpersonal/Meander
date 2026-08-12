# Meander Deployment and CI/CD

This is the locked deployment plan for Meander. The repository contains the pipeline, but releases remain intentional: production runs only when a GitHub Release is published or the deployment workflow is started manually.

## Architecture

```text
Browser
  │
  ▼
Cloudflare frontend
  │  NEXT_PUBLIC_MEANDER_API_URL
  ▼
Google Cloud Run · Go API + field engine
  ├── Neon Postgres · accounts and product records
  └── Cloudflare R2 · permanent artwork objects

GitHub Actions
  ├── pull request / main → lint, build, Go tests, vet, container build
  └── release / manual → verify → Cloud Run → health check → Cloudflare
```

The API now uses a storage interface. Development falls back to local files and an in-memory record store; production refuses to start unless Neon and R2 configuration are present. SVG, PNG, recipe, and fingerprint files are stored in R2, while Neon stores ownership, visibility, and searchable artwork metadata.

## CI behavior

`.github/workflows/ci.yml` runs on every pull request and push to `main`:

- installs dependencies from the lockfile;
- lints, builds, and renders the frontend tests;
- tests and vets the Go API and engine;
- builds the exact production container used by Cloud Run.

A failed check prevents the release pipeline from being considered healthy.

## CD behavior

`.github/workflows/deploy.yml` is triggered by a published GitHub Release or a manual run. It:

1. repeats the complete release verification;
2. authenticates to Google Cloud using short-lived OIDC credentials;
3. deploys the `backend/Dockerfile` source to Cloud Run;
4. calls `/healthz` and stops if the engine is unhealthy;
5. rebuilds the frontend with the deployed Cloud Run URL;
6. deploys the validated frontend to Cloudflare.

The production environment uses a concurrency lock, so two releases cannot deploy over each other. Configure GitHub's `production` environment with an approval rule if the repository plan supports it.

## Required GitHub configuration

Create a GitHub environment named `production`.

Environment secrets:

- `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_SERVICE_ACCOUNT`
- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_ACCOUNT_ID`

Environment variables:

- `GCP_PROJECT_ID`
- `GCP_REGION` — defaults to `us-east1`
- `CLOUD_RUN_SERVICE` — defaults to `meander-api`
- `MEANDER_WEB_ORIGIN` — exact production frontend origin used by API CORS, e.g. `https://meander.app`
- `MEANDER_API_ORIGIN` — exact public API origin, e.g. `https://api.meander.app`; use a sibling subdomain of the frontend so browser sessions remain first-party
- `MEANDER_GOOGLE_CLIENT_ID` — Google Identity Services web client ID, used by both the API and frontend
- `R2_ACCOUNT_ID`
- `R2_BUCKET`

Create these Google Secret Manager secrets before releasing: `meander-database-url`, `meander-r2-access-key-id`, and `meander-r2-secret-access-key`. Google authentication uses Workload Identity Federation. Do not create or store a long-lived Google service-account JSON key in GitHub. The Google Sign-In client ID is a public identifier, so it belongs in the `MEANDER_GOOGLE_CLIENT_ID` environment variable—not in Secret Manager. Google client secrets are not used by Meander's identity-token sign-in flow.

Before the first release, apply [`backend/migrations/0001_product.sql`](../backend/migrations/0001_product.sql) to the production Neon database. The API intentionally does not run schema migrations at startup, which keeps a bad deployment from mutating production data automatically.

## Local parity

Copy `.env.example` to an ignored `.env.local` for local development. The creation studio automatically uses `http://localhost:8080` when `NEXT_PUBLIC_MEANDER_API_URL` is absent. The Go API reads this local file when it is run from the repository or `backend` directory. The API accepts localhost origins automatically and uses `MEANDER_ALLOWED_ORIGINS` for exact production origins.

## Release order

The current safe sequence is:

1. merge only after CI passes;
2. publish a GitHub Release or manually start `Deploy production`;
3. approve the `production` environment if protection is configured;
4. verify the Cloud Run health check and Cloudflare deployment in the workflow summary;
5. sign in with Google, generate one real route, confirm it appears in the private library, then test an unlisted share link.
