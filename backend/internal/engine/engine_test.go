package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

const tinyGPX = `<?xml version="1.0"?><gpx><trk><trkseg><trkpt lat="40.0" lon="-73.0"><ele>2</ele><time>2026-08-09T10:00:00Z</time></trkpt><trkpt lat="40.001" lon="-73.0"><ele>7</ele><time>2026-08-09T10:01:00Z</time></trkpt><trkpt lat="40.001" lon="-73.001"><ele>4</ele><time>2026-08-09T10:05:00Z</time></trkpt><trkpt lat="40.0" lon="-73.001"><ele>12</ele><time>2026-08-09T10:06:00Z</time></trkpt><trkpt lat="40.0" lon="-73.0"><ele>2</ele><time>2026-08-09T10:07:00Z</time></trkpt></trkseg></trk></gpx>`

func TestDeterministic(t *testing.T) {
	p, e := Parse(bytes.NewBufferString(tinyGPX), "x.gpx")
	if e != nil {
		t.Fatal(e)
	}
	c := Context{LocationLabel: "Test Loop", TimeOfDay: "dawn", MusicTempo: 112, MusicEnergy: .6}
	a, e := Generate(p, c)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Generate(p, c)
	if e != nil {
		t.Fatal(e)
	}
	if a.ID != b.ID || a.Recipe.Seed != b.Recipe.Seed || a.Recipe.Candidate != b.Recipe.Candidate {
		t.Fatalf("not deterministic: %#v %#v", a.Recipe, b.Recipe)
	}
	if len(a.Lines) < 100 {
		t.Fatalf("expected field lines, got %d", len(a.Lines))
	}
	if a.Family != "route-turbulence" || a.Recipe.EngineVersion != "field-3.2.1" {
		t.Fatalf("wrong v3 identity: %#v", a.Recipe)
	}
	score := a.Recipe.Score
	if score.NegativeSpace == 0 || score.Hierarchy == 0 || score.RouteLegibility == 0 || score.ColorStructure == 0 || score.AccentDiscipline == 0 || score.FocalStrength == 0 || score.HeroSupport == 0 || score.AnchorStrength == 0 || score.RouteVisibility == 0 || score.CoreFlowCoherence == 0 || score.BundleContinuity == 0 || score.TopologyPreservation != 1 {
		t.Fatalf("missing named quality metrics: %#v", score)
	}
	textures := map[string]bool{}
	routeMemory := false
	for _, line := range a.Lines {
		textures[line.Texture] = true
		if line.Role == "route" {
			t.Fatal("v3 must not render a literal route line")
		}
		if line.Role == "route-memory" {
			routeMemory = line.Opacity >= .3 && line.Opacity <= .4 && line.Width >= 1.4 && line.ColorRole != 1 && line.Texture == "route-memory"
		}
	}
	if !routeMemory {
		t.Fatal("expected a strengthened visible route-memory layer")
	}
	if score.RouteVisibility < .72 || score.CoreFlowCoherence < .68 || score.CollisionRate > .32 {
		t.Fatalf("route corridor quality gate failed: %#v", score)
	}
	for _, texture := range []string{"ribbon", "broken", "thread", "dry-brush", "charcoal"} {
		if !textures[texture] {
			t.Fatalf("missing texture family %q", texture)
		}
	}
	if a.Features.TimedPoints != 5 || a.Features.DwellPoints == 0 || a.Features.DurationMinutes != 7 {
		t.Fatalf("timestamp fingerprint missing: %#v", a.Features)
	}
	if len(a.Events) == 0 {
		t.Fatal("expected locally positioned route events")
	}
}

func TestColdStartSingleUploadUsesGlobalCalibration(t *testing.T) {
	file, err := os.Open("../../fixtures/cold-start-single.gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	points, err := Parse(file, "cold-start-single.gpx")
	if err != nil {
		t.Fatal(err)
	}
	// This is deliberately the complete input: one route and no user/history object.
	result, err := Generate(points, Context{LocationLabel: "First Walk"})
	if err != nil {
		t.Fatal(err)
	}
	profile := GlobalCalibrationProfile()
	composition := result.Recipe.Composition
	if result.Recipe.CalibrationProfile != profile.Version || composition.CalibrationProfile != profile.Version || profile.Scope != "global-corpus" {
		t.Fatalf("cold start did not use global calibration: %#v", result.Recipe)
	}
	if composition.HeroTrails < profile.Hierarchy.HeroMin || composition.HeroTrails > profile.Hierarchy.HeroMax || composition.SupportingTrails < profile.Hierarchy.SupportingMin {
		t.Fatalf("hero/supporting split failed: %#v", composition)
	}
	margin := profile.Anchor.SafeMargin
	anchor := composition.Anchor.Position
	if composition.Anchor.Source == "" || anchor.X < margin || anchor.X > 1-margin || anchor.Y < margin || anchor.Y > 1-margin || result.Recipe.Score.AnchorStrength < profile.Anchor.MinimumScore {
		t.Fatalf("composition anchor failed: anchor=%#v score=%#v", composition.Anchor, result.Recipe.Score)
	}
	if result.Recipe.Score.NegativeSpace < profile.NegativeSpace.MinimumScore {
		t.Fatalf("global negative-space check failed: %#v", result.Recipe.Score)
	}
	if result.Recipe.Score.Hierarchy < profile.Hierarchy.MinimumScore || result.Recipe.Score.HeroSupport == 0 || result.Recipe.Score.Total < profile.MinimumTotalScore {
		t.Fatalf("cold-start quality gate failed: %#v", result.Recipe.Score)
	}
	heroes, supporting := 0, 0
	for _, line := range result.Lines {
		switch line.Role {
		case "hero":
			heroes++
		case "supporting":
			supporting++
		}
	}
	if heroes != composition.HeroTrails || supporting != composition.SupportingTrails {
		t.Fatalf("recipe hierarchy does not match rendered trails: heroes=%d supporting=%d composition=%#v", heroes, supporting, composition)
	}
}

func TestRejectsEmpty(t *testing.T) {
	if _, e := Parse(bytes.NewBufferString("<gpx/>"), "x.gpx"); e == nil {
		t.Fatal("wanted error")
	}
}

func TestEventlessRouteSerializesEmptyArray(t *testing.T) {
	result, err := Generate([]GeoPoint{{Lat: 40, Lon: -73}, {Lat: 40, Lon: -72.999}}, Context{LocationLabel: "Straight Walk"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Events == nil {
		t.Fatal("eventless routes must return an initialized empty event collection")
	}
	encoded, err := json.Marshal(result.Events)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("expected JSON array, got %s", encoded)
	}
}
