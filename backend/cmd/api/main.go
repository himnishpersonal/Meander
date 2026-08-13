package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"walkart/internal/engine"
	"walkart/internal/product"

	"google.golang.org/api/idtoken"
)

type googleIdentity struct{ Email, Name string }
type googleVerifier func(context.Context, string, string) (googleIdentity, error)

type server struct {
	output, fixtures string
	store            product.Store
	objects          product.Objects
	environment      string
	googleClientID   string
	verifyGoogle     googleVerifier
}

func main() {
	loadDevelopmentEnv()
	s, err := newServer(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	addr := envFallback("MEANDER_ADDR", "WALKART_ADDR", ":"+env("PORT", "8080"))
	log.Printf("meander engine %s on %s (%s storage)", engine.Version, addr, s.environment)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}

func newServer(ctx context.Context) (server, error) {
	out := envFallback("MEANDER_OUTPUT", "WALKART_OUTPUT", "output")
	fixtures := envFallback("MEANDER_FIXTURES", "WALKART_FIXTURES", "fixtures")
	s := server{output: out, fixtures: fixtures, environment: env("MEANDER_ENV", "development"), googleClientID: os.Getenv("MEANDER_GOOGLE_CLIENT_ID")}
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		store, err := product.NewPostgresStore(ctx, databaseURL)
		if err != nil {
			return server{}, fmt.Errorf("connect Neon Postgres: %w", err)
		}
		s.store = store
	} else if s.environment == "production" {
		return server{}, errors.New("DATABASE_URL is required in production")
	} else {
		s.store = product.NewMemoryStore()
	}
	if product.R2Enabled() {
		objects, err := product.NewR2Objects(ctx)
		if err != nil {
			return server{}, fmt.Errorf("configure Cloudflare R2: %w", err)
		}
		s.objects = objects
	} else if s.environment == "production" {
		return server{}, errors.New("R2_BUCKET is required in production")
	} else {
		s.objects = product.LocalObjects{Root: filepath.Join(out, "objects")}
	}
	return s, nil
}

func (s server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "engine": engine.Version, "environment": s.environment})
	})
	mux.HandleFunc("GET /api/v1/config", s.publicConfig)
	mux.HandleFunc("GET /api/v1/samples", s.samples)
	mux.HandleFunc("POST /api/v1/auth/google", s.googleSignIn)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/me", s.me)
	mux.HandleFunc("POST /api/v1/generate", s.generate)
	mux.HandleFunc("GET /api/v1/me/artworks", s.library)
	mux.HandleFunc("GET /api/v1/gallery", s.gallery)
	mux.HandleFunc("GET /api/v1/artworks/{id}", s.artwork)
	mux.HandleFunc("PATCH /api/v1/artworks/{id}", s.updateArtwork)
	mux.HandleFunc("DELETE /api/v1/artworks/{id}", s.deleteArtwork)
	mux.HandleFunc("GET /api/v1/artworks/{id}/files/{file}", s.file)
	mux.HandleFunc("GET /api/v1/share/{shareID}", s.sharedArtwork)
	mux.Handle("GET /artifacts/", http.StripPrefix("/artifacts/", http.FileServer(http.Dir(s.output))))
	return cors(mux)
}

func (s server) publicConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"google_sign_in_configured": s.googleClientID != "",
		"google_client_id":          s.googleClientID,
	})
}

func (s server) googleSignIn(w http.ResponseWriter, r *http.Request) {
	if s.googleClientID == "" {
		fail(w, http.StatusServiceUnavailable, errors.New("Google sign-in is not configured"))
		return
	}
	var input struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&input); err != nil || strings.TrimSpace(input.Credential) == "" {
		fail(w, http.StatusBadRequest, errors.New("Google credential is required"))
		return
	}
	verify := s.verifyGoogle
	if verify == nil {
		verify = verifyGoogleCredential
	}
	identity, err := verify(r.Context(), input.Credential, s.googleClientID)
	if err != nil {
		fail(w, http.StatusUnauthorized, errors.New("Google sign-in could not be verified"))
		return
	}
	user, err := s.store.EnsureUser(r.Context(), strings.ToLower(strings.TrimSpace(identity.Email)), clean(identity.Name))
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	s.issueSession(w, r, user)
}

func (s server) issueSession(w http.ResponseWriter, r *http.Request, user product.User) {
	session, err := secureToken()
	if err != nil {
		fail(w, 500, err)
		return
	}
	if err = s.store.CreateSession(r.Context(), tokenHash(session), user.ID, time.Now().Add(30*24*time.Hour)); err != nil {
		fail(w, 500, err)
		return
	}
	sameSite := http.SameSiteLaxMode
	if s.environment == "production" {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{Name: "meander_session", Value: session, Path: "/", HttpOnly: true, Secure: s.environment == "production", SameSite: sameSite, MaxAge: 30 * 24 * 60 * 60})
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (s server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("meander_session"); err == nil {
		_ = s.store.RevokeSession(r.Context(), tokenHash(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "meander_session", Value: "", Path: "/", HttpOnly: true, Secure: s.environment == "production", MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}
func (s server) me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	user, err := s.currentUser(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	writeJSON(w, 200, user)
}

func (s server) samples(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []string{"central-park.osm", "high-line.osm", "brooklyn-bridge.osm", "golden-gate.osm"})
}

func (s server) generate(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		fail(w, http.StatusUnauthorized, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		fail(w, 400, fmt.Errorf("invalid upload: %w", err))
		return
	}
	file, head, err := s.routeFile(r)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	points, err := engine.Parse(file, head.Filename)
	if err != nil {
		fail(w, 422, err)
		return
	}
	result, err := engine.Generate(points, engine.Context{LocationLabel: clean(r.FormValue("location_label")), TimeOfDay: clean(r.FormValue("time_of_day")), MusicTempo: number(r.FormValue("music_tempo")), MusicEnergy: number(r.FormValue("music_energy"))})
	if err != nil {
		fail(w, 422, err)
		return
	}
	if r.FormValue("geometry_only") == "true" {
		result.Features.DistanceKM = 0
		result.Features.DisplacementKM = 0
		result.Features.DurationMinutes = 0
		result.Features.MeanPaceMinKM = 0
		result.Features.ElevationGainM = 0
		result.Features.TimedPoints = 0
		result.Subtitle = "Geometry-only route · " + strings.ToLower(clean(r.FormValue("time_of_day")))
	}
	artwork, err := s.persistArtwork(r.Context(), user, result)
	if err != nil {
		fail(w, 500, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.artworkResponse(artwork, result.Features, result.Events, result.Recipe))
}

func (s server) routeFile(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	file, head, err := r.FormFile("route")
	if err == nil {
		return file, head, nil
	}
	sample := filepath.Base(r.FormValue("sample"))
	if sample == "." || sample == "" {
		return nil, nil, errors.New("route file is required")
	}
	opened, err := os.Open(filepath.Join(s.fixtures, sample))
	if err != nil {
		return nil, nil, fmt.Errorf("sample route not found")
	}
	return opened, &multipart.FileHeader{Filename: sample}, nil
}

func (s server) persistArtwork(ctx context.Context, user product.User, result engine.Result) (product.Artwork, error) {
	dir, err := os.MkdirTemp("", "meander-artwork-")
	if err != nil {
		return product.Artwork{}, err
	}
	defer os.RemoveAll(dir)
	if err = engine.WriteBundle(dir, result); err != nil {
		return product.Artwork{}, err
	}
	id, shareID := product.NewID("art_"), product.NewID("")
	prefix := filepath.ToSlash(filepath.Join("users", user.ID, "artworks", id))
	files := []struct{ name, typ string }{{"artwork.svg", "image/svg+xml"}, {"preview.png", "image/png"}, {"fingerprint.json", "application/json"}, {"recipe.json", "application/json"}}
	for _, file := range files {
		b, readErr := os.ReadFile(filepath.Join(dir, file.name))
		if readErr != nil {
			return product.Artwork{}, readErr
		}
		if putErr := s.objects.Put(ctx, prefix+"/"+file.name, file.typ, b); putErr != nil {
			return product.Artwork{}, putErr
		}
	}
	features, _ := json.Marshal(result.Features)
	events, _ := json.Marshal(result.Events)
	recipe, _ := json.Marshal(result.Recipe)
	score, _ := json.Marshal(result.Recipe.Score)
	a := product.Artwork{ID: id, UserID: user.ID, ShareID: shareID, Title: result.Title, Subtitle: result.Subtitle, Palette: result.Palette, EngineVersion: result.Recipe.EngineVersion, Visibility: product.Private, SVGKey: prefix + "/artwork.svg", PNGKey: prefix + "/preview.png", FingerprintKey: prefix + "/fingerprint.json", RecipeKey: prefix + "/recipe.json", FeaturesJSON: features, EventsJSON: events, RecipeJSON: recipe, ScoreJSON: score, CreatedAt: time.Now().UTC()}
	if err = s.store.CreateArtwork(ctx, a); err != nil {
		return product.Artwork{}, err
	}
	return a, nil
}

func (s server) library(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	items, err := s.store.ListArtworks(r.Context(), user.ID)
	if err != nil {
		fail(w, 500, err)
		return
	}
	summaries := make([]map[string]any, 0, len(items))
	for _, artwork := range items {
		summaries = append(summaries, artworkSummary(artwork))
	}
	writeJSON(w, 200, map[string]any{"user": user, "artworks": summaries})
}
func (s server) artwork(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	a, err := s.store.GetArtwork(r.Context(), r.PathValue("id"))
	if err != nil || a.UserID != user.ID {
		fail(w, 404, product.ErrNotFound)
		return
	}
	writeJSON(w, 200, artworkSummary(a))
}
func (s server) gallery(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListPublicArtworks(r.Context(), 48)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	summaries := make([]map[string]any, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, artworkSummary(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"artworks": summaries})
}
func (s server) updateArtwork(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	var input struct {
		Visibility product.Visibility `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || (input.Visibility != product.Private && input.Visibility != product.Unlisted && input.Visibility != product.Public) {
		fail(w, 400, errors.New("visibility must be private, unlisted, or public"))
		return
	}
	if err = s.store.SetVisibility(r.Context(), r.PathValue("id"), user.ID, input.Visibility); err != nil {
		fail(w, 404, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}
func (s server) deleteArtwork(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	a, err := s.store.DeleteArtwork(r.Context(), r.PathValue("id"), user.ID)
	if err != nil {
		fail(w, 404, err)
		return
	}
	for _, key := range []string{a.SVGKey, a.PNGKey, a.FingerprintKey, a.RecipeKey} {
		_ = s.objects.Delete(r.Context(), key)
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s server) sharedArtwork(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetArtworkByShareID(r.Context(), r.PathValue("shareID"))
	if err != nil || a.Visibility == product.Private {
		fail(w, 404, product.ErrNotFound)
		return
	}
	writeJSON(w, 200, artworkSummary(a))
}
func (s server) file(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetArtwork(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, 404, err)
		return
	}
	user, userErr := s.currentUser(r)
	if a.Visibility == product.Private && (userErr != nil || user.ID != a.UserID) {
		fail(w, 404, product.ErrNotFound)
		return
	}
	key := map[string]string{"svg": a.SVGKey, "png": a.PNGKey, "fingerprint": a.FingerprintKey, "recipe": a.RecipeKey}[r.PathValue("file")]
	if key == "" {
		fail(w, 404, product.ErrNotFound)
		return
	}
	b, typ, err := s.objects.Get(r.Context(), key)
	if err != nil {
		fail(w, 404, err)
		return
	}
	w.Header().Set("Content-Type", typ)
	w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	_, _ = w.Write(b)
}

func (s server) artworkResponse(a product.Artwork, features engine.Features, events []engine.RouteEvent, recipe engine.Recipe) map[string]any {
	return map[string]any{"id": a.ID, "share_id": a.ShareID, "share_url": "/m/" + a.ShareID, "title": a.Title, "subtitle": a.Subtitle, "palette": a.Palette, "family": "route-turbulence", "features": features, "events": events, "recipe": recipe, "score": recipe.Score, "visibility": a.Visibility, "artwork_url": "/api/v1/artworks/" + a.ID + "/files/svg", "preview_url": "/api/v1/artworks/" + a.ID + "/files/png"}
}

func artworkSummary(a product.Artwork) map[string]any {
	return map[string]any{"id": a.ID, "share_id": a.ShareID, "title": a.Title, "subtitle": a.Subtitle, "palette": a.Palette, "engine_version": a.EngineVersion, "visibility": a.Visibility, "created_at": a.CreatedAt, "features": json.RawMessage(a.FeaturesJSON), "events": json.RawMessage(a.EventsJSON), "recipe": json.RawMessage(a.RecipeJSON), "score": json.RawMessage(a.ScoreJSON), "artwork_url": "/api/v1/artworks/" + a.ID + "/files/svg", "preview_url": "/api/v1/artworks/" + a.ID + "/files/png", "share_url": "/m/" + a.ShareID}
}

func (s server) currentUser(r *http.Request) (product.User, error) {
	if cookie, err := r.Cookie("meander_session"); err == nil {
		if user, sessionErr := s.store.UserForSession(r.Context(), tokenHash(cookie.Value)); sessionErr == nil {
			return user, nil
		}
	}
	if s.environment == "production" || s.googleClientID != "" {
		return product.User{}, errors.New("sign in is required")
	}
	return s.store.EnsureUser(r.Context(), env("MEANDER_DEV_USER_EMAIL", "local@meander.local"), "Local creator")
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Idempotency-Key")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func originAllowed(origin string) bool {
	if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
		return true
	}
	for _, allowed := range strings.Split(os.Getenv("MEANDER_ALLOWED_ORIGINS"), ",") {
		if strings.TrimSpace(allowed) == origin && origin != "" {
			return true
		}
	}
	return false
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func number(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
func clean(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envFallback(primary, legacy, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return env(legacy, fallback)
}

func secureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func verifyGoogleCredential(ctx context.Context, credential, audience string) (googleIdentity, error) {
	payload, err := idtoken.Validate(ctx, credential, audience)
	if err != nil {
		return googleIdentity{}, err
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	verified, ok := payload.Claims["email_verified"].(bool)
	if !ok {
		if value, stringOK := payload.Claims["email_verified"].(string); stringOK {
			verified = value == "true"
		}
	}
	if email == "" || !verified {
		return googleIdentity{}, errors.New("Google account must have a verified email")
	}
	return googleIdentity{Email: email, Name: name}, nil
}

// loadDevelopmentEnv makes the ignored root .env.local available to a locally
// launched Go API. Hosted environments supply values directly and take priority.
func loadDevelopmentEnv() {
	if os.Getenv("MEANDER_ENV") == "production" {
		return
	}
	for _, path := range []string{".env.local", "../.env.local"} {
		contents, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(contents), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			key = strings.TrimSpace(key)
			if !ok || key == "" {
				continue
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
			value = strings.Trim(strings.TrimSpace(value), "\"'")
			_ = os.Setenv(key, value)
		}
		return
	}
}
