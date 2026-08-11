# Global calibration corpus

`walk-art-v1` is the engine's global normalization profile. It is embedded in the Go binary from `internal/engine/calibration/walk-art-v1.json` and applies to every generation, including a person's first upload.

The corpus manifest includes a dedicated `cold-start-single-upload` acceptance case with `user_history_count: 0`. No generation or scoring API accepts personal history. Future personalization must remain downstream of these global quality gates.
