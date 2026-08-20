# Meander public-release checklist

This is the release gate for Meander's Vercel frontend, Cloud Run API, Neon
database, and Cloudflare R2 artwork storage. A checked code item is not a
substitute for the owner-controlled release steps at the end.

## Implemented in the application

- [x] Google ID tokens are verified server-side; session tokens are random,
  hashed at rest, HTTP-only, secure in production, and `SameSite=Lax`.
- [x] State-changing browser requests require an allowed origin and the
  `X-Meander-Request: browser` header.
- [x] Production security headers and a restrictive browser Content Security
  Policy are emitted.
- [x] Raw uploads are capped at 16 MB; XML complexity and point counts are
  bounded; multipart temporary files are removed after handling.
- [x] Shared generation quota is persisted in Neon. Short-window limits use a
  shared Neon-backed window keyed by salted hashes rather than a Cloud Run
  instance's memory.
- [x] New work is private by default. Changing private work to shareable
  rotates its share identifier. Revoked links cannot retrieve artwork files.
- [x] Public endpoints return only intentional display fields, not route event
  streams, engine recipes, seeds, or raw uploads.
- [x] Users can delete individual work or permanently delete their account,
  stored artwork files, sessions, and account records.
- [x] Privacy, terms, and copyright-reporting pages are linked from the site.
- [x] CI runs frontend checks, production dependency audit, Go tests/vet/build,
  container build, and secret scanning.

## Required before deploying this change

1. Apply `backend/migrations/0002_launch_safety.sql` to the production Neon
   database. This creates the shared rate-limit table.
2. Create a Secret Manager secret named `meander-rate-limit-salt` with a random
   32-byte-or-longer value, for example `openssl rand -hex 32`. Grant the Cloud
   Run service account access. Do not put this value in Vercel or Git.
3. Confirm `MEANDER_ALLOWED_ORIGINS` contains only the production Vercel origin
   (and separately configured preview origins if intentionally supported).
4. Confirm the R2 API token is limited to the Meander bucket and only object
   read/write/delete actions it needs. Confirm the Cloud Run service account is
   limited to Secret Manager access and deployment-required permissions.
5. Deploy to a private beta first. Verify `/healthz`, Google sign-in, create,
   private library access, sharing/revocation, account deletion, and the policy
   links on the real domains.

## Required owner/legal decisions before an unrestricted public gallery

- Supply Meander's legal business name, postal address, governing jurisdiction,
  support/privacy email, and effective dates. Replace the draft notices with
  counsel-reviewed policy text.
- Decide the age policy. The current draft uses 16+ / local digital-consent age.
  Do not knowingly collect precise-route data from children without an
  appropriate parental-consent program.
- Appoint and publish a copyright contact. If launching public user-generated
  content in the United States, register a DMCA designated agent with the U.S.
  Copyright Office and place the registered contact on `/copyright`.
- Document a takedown and counter-notice workflow and assign an owner who can
  respond quickly.
- Keep OpenStreetMap attribution wherever fixtures or OSM-derived examples are
  displayed. Do not use third-party map screenshots in marketing without
  permission. The upload flow must remain limited to material the user has the
  right to use.
- Review vendor agreements, backups, incident response, retention behavior,
  and privacy-rights obligations for the locations where you offer Meander.

## Post-release operating checklist

- Watch Cloud Run errors and latency without logging raw routes, screenshots,
  Google credentials, session values, or location coordinates.
- Review failed-generation, quota, abuse, and copyright reports weekly during
  beta.
- Test Neon restore and R2 object recovery before relying on backups.
- Re-run the access-control test set after any changes to sharing, login,
  uploads, or Strava integration.
