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
	if response.Recipe.EngineVersion != "field-3.2.0" || response.Recipe.CalibrationProfile != "walk-art-v1" || response.Recipe.Composition.HeroTrails == 0 || response.Recipe.Composition.SupportingTrails == 0 || response.Family != "route-turbulence" || response.ArtworkURL == "" {
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
	t.Setenv("MEANDER_ALLOWED_ORIGINS", "https://meander.example, https://www.meander.example")
	if !originAllowed("https://meander.example") || !originAllowed("https://www.meander.example") {
		t.Fatal("configured production origins must be allowed")
	}
	if originAllowed("https://attacker.example") {
		t.Fatal("unconfigured production origins must be rejected")
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
