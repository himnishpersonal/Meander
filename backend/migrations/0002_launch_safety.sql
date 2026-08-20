-- Shared rate-limit windows make quotas consistent across Cloud Run instances.
-- Keys are opaque, salted hashes; never store a raw IP address here.
CREATE TABLE IF NOT EXISTS rate_limit_windows (
  rate_key TEXT PRIMARY KEY,
  window_started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  request_count INTEGER NOT NULL DEFAULT 0 CHECK (request_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_rate_limit_windows_started
  ON rate_limit_windows(window_started_at);
