const configuredBase = process.env.NEXT_PUBLIC_MEANDER_API_URL;

// Vercel proxies production API calls through the frontend origin. Local
// development keeps talking directly to the Go API on port 8080.
export const API = (configuredBase === "same-origin" ? "" : configuredBase || "http://localhost:8080").replace(/\/$/, "");
