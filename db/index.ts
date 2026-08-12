import type { drizzle } from "drizzle-orm/d1";

/**
 * Kept as a clear boundary for an unused starter-template module.
 *
 * Meander's product data lives in Neon and is accessed by the Go API. The
 * browser app must never open a database connection directly. Leaving the
 * former Cloudflare D1 import here made a Vercel/Next production build depend
 * on Cloudflare-only types, even though no application code uses this module.
 */
export function getDb(): ReturnType<typeof drizzle> {
  throw new Error(
    "The frontend does not access a database directly. Use the Meander Go API."
  );
}
