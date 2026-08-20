package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"walkart/internal/product"
)

func TestGenerateReturnsOneCanonicalArtwork(t *testing.T) {
	out := t.TempDir()
	s := server{output: out, fixtures: "../../fixtures", store: product.NewMemoryStore(), objects: product.LocalObjects{Root: filepath.Join(out, "objects")}, environment: "development"}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	fields := map[string]string{"sample": "central-park.osm", "location_label": "Central Park Test", "time_of_day": "morning", "music_tempo": "108", "music_energy": "0.54"}
	for key, value := range fields {
		if err := form.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/generate", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set(csrfHeader, "browser")
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		ID     string `json:"id"`
		Family string `json:"family"`
		Recipe struct {
			EngineVersion      string `json:"engine_version"`
			CalibrationProfile string `json:"calibration_profile"`
			Composition        struct {
				HeroTrails       int `json:"hero_trails"`
				SupportingTrails int `json:"supporting_trails"`
			} `json:"composition"`
		} `json:"recipe"`
		Score      map[string]float64 `json:"score"`
		ArtworkURL string             `json:"artwork_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Recipe.EngineVersion != "field-3.2.1" || response.Recipe.CalibrationProfile != "walk-art-v1" || response.Recipe.Composition.HeroTrails == 0 || response.Recipe.Composition.SupportingTrails == 0 || response.Family != "route-turbulence" || response.ArtworkURL == "" {
		t.Fatalf("bad v3 contract: %#v", response)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"finalists"`)) {
		t.Fatal("the public v3 response must contain one artwork, not finalists")
	}
	if response.Score["NegativeSpace"] == 0 || response.Score["Hierarchy"] == 0 || response.Score["ColorStructure"] == 0 || response.Score["FocalStrength"] == 0 {
		t.Fatalf("missing scoring metrics: %#v", response.Score)
	}
	path := filepath.Join(out, "objects", "users")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	libraryReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/artworks", nil)
	libraryRec := httptest.NewRecorder()
	s.routes().ServeHTTP(libraryRec, libraryReq)
	if libraryRec.Code != http.StatusOK || !bytes.Contains(libraryRec.Body.Bytes(), []byte(response.ID)) {
		t.Fatalf("saved artwork missing from library: %d %s", libraryRec.Code, libraryRec.Body.String())
	}
	fileReq := httptest.NewRequest(http.MethodGet, response.ArtworkURL, nil)
	fileRec := httptest.NewRecorder()
	s.routes().ServeHTTP(fileRec, fileReq)
	if fileRec.Code != http.StatusOK || !bytes.Contains(fileRec.Body.Bytes(), []byte("<svg")) {
		t.Fatalf("saved SVG unavailable: %d %s", fileRec.Code, fileRec.Body.String())
	}
}

func TestProductionCORSUsesAllowList(t *testing.T) {
	t.Setenv("MEANDER_ENV", "production")
	t.Setenv("MEANDER_ALLOWED_ORIGINS", "https://meander.example, https://www.meander.example")
	if !originAllowed("https://meander.example") || !originAllowed("https://www.meander.example") {
		t.Fatal("configured production origins must be allowed")
	}
	if originAllowed("https://attacker.example") {
		t.Fatal("unconfigured production origins must be rejected")
	}
}

func TestAccountDeletionRemovesStoredArtworkAndSession(t *testing.T) {
	store := product.NewMemoryStore()
	user, _ := store.EnsureUser(context.Background(), "delete@example.com", "Delete")
	a := product.Artwork{ID: "delete-art", UserID: user.ID, ShareID: "delete-share", Title: "Delete", Visibility: product.Private, SVGKey: "users/delete-art/artwork.svg", PNGKey: "users/delete-art/preview.png", FingerprintKey: "users/delete-art/fingerprint.json", RecipeKey: "users/delete-art/recipe.json", FeaturesJSON: []byte(`{}`), EventsJSON: []byte(`[]`), RecipeJSON: []byte(`{}`), ScoreJSON: []byte(`{}`), CreatedAt: time.Now()}
	if err := store.CreateArtwork(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	objects := product.LocalObjects{Root: t.TempDir()}
	if err := objects.Put(context.Background(), a.PNGKey, "image/png", []byte("png")); err != nil {
		t.Fatal(err)
	}
	session := "delete-session"
	if err := store.CreateSession(context.Background(), tokenHash(session), user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	s := server{store: store, objects: objects, environment: "production"}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/me", bytes.NewBufferString(`{"confirmation":"DELETE"}`))
	req.Header.Set(csrfHeader, "browser")
	req.AddCookie(&http.Cookie{Name: "__Host-meander_session", Value: session})
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete account: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := store.UserForSession(context.Background(), tokenHash(session)); !errors.Is(err, product.ErrNotFound) {
		t.Fatalf("session remained after deletion: %v", err)
	}
	if _, _, err := objects.Get(context.Background(), a.PNGKey); err == nil {
		t.Fatal("artwork object remained after deletion")
	}
}

func TestGalleryOnlyReturnsPublishedArtwork(t *testing.T) {
	store := product.NewMemoryStore()
	user, err := store.EnsureUser(context.Background(), "walker@example.com", "Walker")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	private := product.Artwork{ID: "private-work", UserID: user.ID, ShareID: "private-share", Title: "Private", Visibility: product.Private, FeaturesJSON: []byte(`{}`), EventsJSON: []byte(`[]`), RecipeJSON: []byte(`{}`), ScoreJSON: []byte(`{}`), CreatedAt: now}
	public := product.Artwork{ID: "public-work", UserID: user.ID, ShareID: "public-share", Title: "Published", Visibility: product.Public, FeaturesJSON: []byte(`{}`), EventsJSON: []byte(`[]`), RecipeJSON: []byte(`{}`), ScoreJSON: []byte(`{}`), CreatedAt: now.Add(time.Second)}
	if err = store.CreateArtwork(context.Background(), private); err != nil {
		t.Fatal(err)
	}
	if err = store.CreateArtwork(context.Background(), public); err != nil {
		t.Fatal(err)
	}
	s := server{store: store, environment: "development"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gallery", nil)
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("public-work")) || bytes.Contains(rec.Body.Bytes(), []byte("private-work")) {
		t.Fatalf("gallery visibility leak: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPublicConfigReportsGoogleSignIn(t *testing.T) {
	s := server{googleClientID: "public-client.apps.googleusercontent.com"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"google_sign_in_configured":true`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"google_client_id":"public-client.apps.googleusercontent.com"`)) {
		t.Fatalf("unexpected public config: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProductionGenerateRequiresSignIn(t *testing.T) {
	s := server{store: product.NewMemoryStore(), environment: "production"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/generate", nil)
	req.Header.Set(csrfHeader, "browser")
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !bytes.Contains(rec.Body.Bytes(), []byte("sign in is required")) {
		t.Fatalf("expected authentication response, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestGoogleSignInCreatesSession(t *testing.T) {
	s := server{store: product.NewMemoryStore(), objects: product.LocalObjects{Root: t.TempDir()}, environment: "development", googleClientID: "google-client", verifyGoogle: func(_ context.Context, credential, audience string) (googleIdentity, error) {
		if credential != "trusted-google-token" || audience != "google-client" {
			return googleIdentity{}, errors.New("invalid token")
		}
		return googleIdentity{Email: "walker@example.com", Name: "Trail Walker"}, nil
	}}
	start := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewBufferString(`{"credential":"trusted-google-token"}`))
	start.Header.Set("Content-Type", "application/json")
	start.Header.Set(csrfHeader, "browser")
	startRec := httptest.NewRecorder()
	s.routes().ServeHTTP(startRec, start)
	if startRec.Code != http.StatusCreated {
		t.Fatalf("Google sign-in: %d %s", startRec.Code, startRec.Body.String())
	}
	cookies := startRec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected session cookie, got %#v", cookies)
	}
	me := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	me.AddCookie(cookies[0])
	meRec := httptest.NewRecorder()
	s.routes().ServeHTTP(meRec, me)
	if meRec.Code != http.StatusOK || !bytes.Contains(meRec.Body.Bytes(), []byte("walker@example.com")) {
		t.Fatalf("Google session did not identify user: %d %s", meRec.Code, meRec.Body.String())
	}
}

func TestProductionGoogleSessionCookieSupportsFrontendProxy(t *testing.T) {
	s := server{store: product.NewMemoryStore(), objects: product.LocalObjects{Root: t.TempDir()}, environment: "production", googleClientID: "google-client", verifyGoogle: func(_ context.Context, _, _ string) (googleIdentity, error) {
		return googleIdentity{Email: "walker@example.com", Name: "Trail Walker"}, nil
	}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewBufferString(`{"credential":"trusted-google-token"}`))
	req.Header.Set(csrfHeader, "browser")
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	cookies := rec.Result().Cookies()
	if rec.Code != http.StatusCreated || len(cookies) != 1 {
		t.Fatalf("expected production session cookie, got %d %#v", rec.Code, cookies)
	}
	if cookies[0].Name != "__Host-meander_session" || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode || !cookies[0].HttpOnly || cookies[0].Path != "/" {
		t.Fatalf("production cookie must be host-bound and protected: %#v", cookies[0])
	}
}

func TestStateChangingRequestsRequireCSRFHeaderAndAllowedOrigin(t *testing.T) {
	s := server{store: product.NewMemoryStore(), environment: "production", googleClientID: "google-client"}
	missing := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewBufferString(`{"credential":"token"}`))
	missingRec := httptest.NewRecorder()
	s.routes().ServeHTTP(missingRec, missing)
	if missingRec.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF header returned %d", missingRec.Code)
	}
	t.Setenv("MEANDER_ALLOWED_ORIGINS", "https://meander.example")
	hostile := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewBufferString(`{"credential":"token"}`))
	hostile.Header.Set(csrfHeader, "browser")
	hostile.Header.Set("Origin", "https://attacker.example")
	hostileRec := httptest.NewRecorder()
	s.routes().ServeHTTP(hostileRec, hostile)
	if hostileRec.Code != http.StatusForbidden {
		t.Fatalf("hostile origin returned %d", hostileRec.Code)
	}
}

func TestPublicGalleryUsesMinimizedPayload(t *testing.T) {
	store := product.NewMemoryStore()
	user, _ := store.EnsureUser(context.Background(), "public@example.com", "Public")
	a := product.Artwork{ID: "art-public", UserID: user.ID, ShareID: "share-secret", Title: "Public work", Subtitle: "A walk", Visibility: product.Public, FeaturesJSON: []byte(`{"DistanceKM":4.2,"HardTurns":7}`), EventsJSON: []byte(`[{"kind":"pause"}]`), RecipeJSON: []byte(`{"seed":"sensitive"}`), ScoreJSON: []byte(`{"Total":0.91}`), CreatedAt: time.Now()}
	if err := store.CreateArtwork(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	s := server{store: store, environment: "production"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gallery", nil)
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || bytes.Contains(rec.Body.Bytes(), []byte(`"events"`)) || bytes.Contains(rec.Body.Bytes(), []byte(`"recipe"`)) || bytes.Contains(rec.Body.Bytes(), []byte(`"features"`)) || bytes.Contains(rec.Body.Bytes(), []byte(`"share_id"`)) {
		t.Fatalf("public payload exposes private generation data: %d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"metrics"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`/m/share-secret`)) {
		t.Fatalf("public payload is missing approved fields: %s", rec.Body.String())
	}
}

func TestPrivateToSharedTransitionRotatesShareID(t *testing.T) {
	store := product.NewMemoryStore()
	user, _ := store.EnsureUser(context.Background(), "owner@example.com", "Owner")
	a := product.Artwork{ID: "art-owned", UserID: user.ID, ShareID: "old-share", Title: "Owned", Visibility: product.Private, FeaturesJSON: []byte(`{}`), EventsJSON: []byte(`[]`), RecipeJSON: []byte(`{}`), ScoreJSON: []byte(`{}`), CreatedAt: time.Now()}
	if err := store.CreateArtwork(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	session := "session-token"
	if err := store.CreateSession(context.Background(), tokenHash(session), user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	s := server{store: store, environment: "production"}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/artworks/art-owned", bytes.NewBufferString(`{"visibility":"unlisted"}`))
	req.SetPathValue("id", "art-owned")
	req.Header.Set(csrfHeader, "browser")
	req.AddCookie(&http.Cookie{Name: "__Host-meander_session", Value: session})
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || bytes.Contains(rec.Body.Bytes(), []byte("old-share")) {
		t.Fatalf("share ID was not rotated: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := store.GetArtworkByShareID(context.Background(), "old-share"); !errors.Is(err, product.ErrNotFound) {
		t.Fatal("old share link still resolves")
	}
}

func TestRateLimiterAndDailyQuota(t *testing.T) {
	limiter := newRateLimiter()
	if ok, _ := limiter.allow("client", 1, time.Minute); !ok {
		t.Fatal("first request should pass")
	}
	if ok, _ := limiter.allow("client", 1, time.Minute); ok {
		t.Fatal("second request should be rate limited")
	}
	store := product.NewMemoryStore()
	user, _ := store.EnsureUser(context.Background(), "quota@example.com", "Quota")
	if err := store.StartGeneration(context.Background(), "job-1", user.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.StartGeneration(context.Background(), "job-2", user.ID, 1); !errors.Is(err, product.ErrQuotaExceeded) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if allowed, _, err := store.AllowRateLimit(context.Background(), "opaque-key", 1, time.Minute); err != nil || !allowed {
		t.Fatalf("first shared limit request: allowed=%v err=%v", allowed, err)
	}
	if allowed, _, err := store.AllowRateLimit(context.Background(), "opaque-key", 1, time.Minute); err != nil || allowed {
		t.Fatalf("second shared limit request: allowed=%v err=%v", allowed, err)
	}
}
