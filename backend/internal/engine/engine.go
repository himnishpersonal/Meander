package engine

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const Version = "field-3.2.1"

type GeoPoint struct {
	Lat, Lon, Elevation float64
	Time                time.Time
}
type Point struct{ X, Y float64 }
type Line struct {
	Points    []Point
	Width     float64
	Opacity   float64
	Role      string
	Texture   string
	ColorRole int
}
type Context struct {
	LocationLabel, TimeOfDay string
	MusicTempo, MusicEnergy  float64
	Family, Palette          string
}
type Features struct {
	DistanceKM, DisplacementKM, Tortuosity, LoopClosure, MeanCurvature, MaxCurvature, ElevationGainM float64
	DurationMinutes, MeanPaceMinKM                                                                   float64
	HardTurns, SelfIntersections, PointCount, TimedPoints, DwellPoints, PaceChanges                  int
	AspectRatio                                                                                      float64
}
type Score struct {
	Total, Coverage, Balance, NegativeSpace, Hierarchy, RouteLegibility, EdgeSafety, Complexity float64
	ColorStructure, AccentDiscipline, FocalStrength, HeroSupport, AnchorStrength                float64
	RouteVisibility, CoreFlowCoherence, CollisionRate, BundleContinuity, TopologyPreservation   float64
}
type RouteEvent struct {
	Kind     string  `json:"kind"`
	Position Point   `json:"position"`
	Strength float64 `json:"strength"`
}
type CompositionAnchor struct {
	Position Point   `json:"position"`
	Source   string  `json:"source"`
	Strength float64 `json:"strength"`
}
type Composition struct {
	CalibrationProfile string            `json:"calibration_profile"`
	Anchor             CompositionAnchor `json:"anchor"`
	HeroTrails         int               `json:"hero_trails"`
	SupportingTrails   int               `json:"supporting_trails"`
	AmbientTrails      int               `json:"ambient_trails"`
}
type Recipe struct {
	EngineVersion      string      `json:"engine_version"`
	CalibrationProfile string      `json:"calibration_profile"`
	Seed               string      `json:"seed"`
	Palette            string      `json:"palette"`
	Candidate          string      `json:"candidate"`
	Context            Context     `json:"context"`
	Composition        Composition `json:"composition"`
	Score              Score       `json:"score"`
}
type Result struct {
	ID, Title, Subtitle, Palette, Family string
	Features                             Features
	Recipe                               Recipe
	Lines                                []Line  `json:"-"`
	Route                                []Point `json:"-"`
	Events                               []RouteEvent
	Colors                               [6]color.RGBA `json:"-"`
}

type gpxDoc struct {
	Tracks []struct {
		Segments []struct {
			Points []struct {
				Lat       float64 `xml:"lat,attr"`
				Lon       float64 `xml:"lon,attr"`
				Elevation float64 `xml:"ele"`
				Stamp     string  `xml:"time"`
			} `xml:"trkpt"`
		} `xml:"trkseg"`
	} `xml:"trk"`
}
type osmDoc struct {
	Nodes []struct {
		ID  int64   `xml:"id,attr"`
		Lat float64 `xml:"lat,attr"`
		Lon float64 `xml:"lon,attr"`
	} `xml:"node"`
	Ways []struct {
		ID   int64 `xml:"id,attr"`
		Refs []struct {
			Ref int64 `xml:"ref,attr"`
		} `xml:"nd"`
	} `xml:"way"`
	Relations []struct {
		Members []struct {
			Type string `xml:"type,attr"`
			Ref  int64  `xml:"ref,attr"`
			Role string `xml:"role,attr"`
		} `xml:"member"`
	} `xml:"relation"`
}

func Parse(r io.Reader, filename string) ([]GeoPoint, error) {
	b, err := io.ReadAll(io.LimitReader(r, 16<<20))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("route file is empty")
	}
	name := strings.ToLower(filename)
	if strings.HasSuffix(name, ".osm") || strings.Contains(string(b[:min(len(b), 300)]), "<osm") {
		return parseOSM(b)
	}
	return parseGPX(b)
}

func parseGPX(b []byte) ([]GeoPoint, error) {
	var d gpxDoc
	if err := xml.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("invalid GPX: %w", err)
	}
	var out []GeoPoint
	for _, tr := range d.Tracks {
		for _, seg := range tr.Segments {
			for _, p := range seg.Points {
				t, _ := time.Parse(time.RFC3339, p.Stamp)
				out = append(out, GeoPoint{p.Lat, p.Lon, p.Elevation, t})
			}
		}
	}
	return validateGeo(out)
}

func parseOSM(b []byte) ([]GeoPoint, error) {
	var d osmDoc
	if err := xml.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("invalid OSM XML: %w", err)
	}
	nodes := map[int64]GeoPoint{}
	for _, n := range d.Nodes {
		nodes[n.ID] = GeoPoint{Lat: n.Lat, Lon: n.Lon}
	}
	ways := map[int64][]int64{}
	for _, w := range d.Ways {
		for _, n := range w.Refs {
			ways[w.ID] = append(ways[w.ID], n.Ref)
		}
	}
	var refs []int64
	if len(d.Relations) > 0 {
		for _, m := range d.Relations[0].Members {
			if m.Type != "way" {
				continue
			}
			refs = joinRefs(refs, ways[m.Ref])
		}
	}
	if len(refs) == 0 && len(d.Ways) > 0 {
		best := d.Ways[0]
		for _, w := range d.Ways[1:] {
			if len(w.Refs) > len(best.Refs) {
				best = w
			}
		}
		for _, n := range best.Refs {
			refs = append(refs, n.Ref)
		}
	}
	var out []GeoPoint
	for _, id := range refs {
		if p, ok := nodes[id]; ok {
			out = append(out, p)
		}
	}
	return validateGeo(out)
}

func joinRefs(dst, src []int64) []int64 {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append(dst, src...)
	}
	last := dst[len(dst)-1]
	if src[0] == last {
		return append(dst, src[1:]...)
	}
	if src[len(src)-1] == last {
		for i := len(src) - 2; i >= 0; i-- {
			dst = append(dst, src[i])
		}
		return dst
	}
	return append(dst, src...)
}

func validateGeo(in []GeoPoint) ([]GeoPoint, error) {
	if len(in) < 2 {
		return nil, errors.New("route needs at least two coordinates")
	}
	out := make([]GeoPoint, 0, len(in))
	for _, p := range in {
		if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
			continue
		}
		if len(out) == 0 || math.Abs(p.Lat-out[len(out)-1].Lat) > 1e-9 || math.Abs(p.Lon-out[len(out)-1].Lon) > 1e-9 {
			out = append(out, p)
		}
	}
	if len(out) < 2 {
		return nil, errors.New("route has no usable movement")
	}
	return out, nil
}

func Generate(geo []GeoPoint, ctx Context) (Result, error) {
	geo, err := validateGeo(geo)
	if err != nil {
		return Result{}, err
	}
	pts, meters := project(geo)
	pts = resample(pts, 180)
	pts = smooth(pts, 2)
	route := normalize(pts, 0.12)
	features := measure(geo, pts, meters)
	enrichFeatures(geo, &features)
	events := detectEvents(geo, route)
	if events == nil {
		events = []RouteEvent{}
	}
	ctx.Family = "route-turbulence"
	ctx.Palette = "route-directed"
	seed := canonicalSeed(geo, ctx)
	candidates := buildTurbulenceCandidates(route, events, features, ctx, seed, walkArtV1)
	canonical, ok := chooseCanonical(candidates)
	if !ok {
		return Result{}, errors.New("engine produced no viable candidates")
	}
	id := candidateID(seed, canonical.index, "route-turbulence")
	paletteName, colors := routePalette(ctx, seed)
	title := ctx.LocationLabel
	if title == "" {
		title = "Untitled walk"
	}
	subtitle := fmt.Sprintf("%.2f km · %d turns · %s", features.DistanceKM, features.HardTurns, strings.ToLower(timeLabel(ctx.TimeOfDay)))
	result := Result{ID: id, Title: title, Subtitle: subtitle, Palette: paletteName, Family: "route-turbulence", Features: features, Lines: canonical.lines, Route: route, Events: events, Colors: colors, Recipe: Recipe{EngineVersion: Version, CalibrationProfile: walkArtV1.Version, Seed: fmt.Sprintf("%016x", seed), Palette: paletteName, Candidate: fmt.Sprintf("%02d", canonical.index+1), Context: ctx, Composition: canonical.composition, Score: canonical.score}}
	return result, nil
}

func project(g []GeoPoint) ([]Point, float64) {
	lat0 := 0.0
	for _, p := range g {
		lat0 += p.Lat
	}
	lat0 /= float64(len(g))
	c := math.Cos(lat0 * math.Pi / 180)
	const earth = 6371000.0
	pts := make([]Point, len(g))
	total := 0.0
	for i, p := range g {
		pts[i] = Point{earth * p.Lon * math.Pi / 180 * c, earth * p.Lat * math.Pi / 180}
		if i > 0 {
			total += dist(pts[i-1], pts[i])
		}
	}
	return pts, total
}
func resample(p []Point, n int) []Point {
	if len(p) < 2 {
		return p
	}
	cum := make([]float64, len(p))
	for i := 1; i < len(p); i++ {
		cum[i] = cum[i-1] + dist(p[i-1], p[i])
	}
	if cum[len(cum)-1] == 0 {
		return p
	}
	out := make([]Point, 0, n)
	j := 1
	for i := 0; i < n; i++ {
		t := cum[len(cum)-1] * float64(i) / float64(n-1)
		for j < len(cum)-1 && cum[j] < t {
			j++
		}
		a, b := p[j-1], p[j]
		q := (t - cum[j-1]) / math.Max(cum[j]-cum[j-1], 1e-9)
		out = append(out, Point{a.X + (b.X-a.X)*q, a.Y + (b.Y-a.Y)*q})
	}
	return out
}
func smooth(p []Point, r int) []Point {
	out := make([]Point, len(p))
	for i := range p {
		var x, y float64
		c := 0
		for j := max(0, i-r); j <= min(len(p)-1, i+r); j++ {
			x += p[j].X
			y += p[j].Y
			c++
		}
		out[i] = Point{x / float64(c), y / float64(c)}
	}
	return out
}
func normalize(p []Point, pad float64) []Point {
	minX, maxX, minY, maxY := p[0].X, p[0].X, p[0].Y, p[0].Y
	for _, q := range p {
		minX = math.Min(minX, q.X)
		maxX = math.Max(maxX, q.X)
		minY = math.Min(minY, q.Y)
		maxY = math.Max(maxY, q.Y)
	}
	w, h := maxX-minX, maxY-minY
	s := (1 - 2*pad) / math.Max(w, h)
	out := make([]Point, len(p))
	for i, q := range p {
		out[i] = Point{.5 + (q.X-(minX+maxX)/2)*s, .5 - (q.Y-(minY+maxY)/2)*s}
	}
	return out
}

func measure(g []GeoPoint, p []Point, total float64) Features {
	f := Features{DistanceKM: total / 1000, PointCount: len(g)}
	f.DisplacementKM = dist(p[0], p[len(p)-1]) / 1000
	f.Tortuosity = total / math.Max(dist(p[0], p[len(p)-1]), 1)
	f.LoopClosure = 1 - math.Min(1, dist(p[0], p[len(p)-1])/math.Max(total, 1))
	minX, maxX, minY, maxY := p[0].X, p[0].X, p[0].Y, p[0].Y
	for i := 1; i < len(p)-1; i++ {
		a := turn(p[i-1], p[i], p[i+1])
		f.MeanCurvature += a
		f.MaxCurvature = math.Max(f.MaxCurvature, a)
		if a > .65 {
			f.HardTurns++
		}
		minX = math.Min(minX, p[i].X)
		maxX = math.Max(maxX, p[i].X)
		minY = math.Min(minY, p[i].Y)
		maxY = math.Max(maxY, p[i].Y)
	}
	f.MeanCurvature /= float64(max(1, len(p)-2))
	f.AspectRatio = (maxX - minX) / math.Max(maxY-minY, 1)
	for i := 1; i < len(g); i++ {
		if g[i].Elevation > g[i-1].Elevation {
			f.ElevationGainM += g[i].Elevation - g[i-1].Elevation
		}
	}
	step := max(1, len(p)/80)
	for i := 0; i < len(p)-3*step; i += step {
		for j := i + 2*step; j < len(p)-step; j += step {
			if segmentsCross(p[i], p[i+step], p[j], p[j+step]) {
				f.SelfIntersections++
			}
		}
	}
	return f
}
func turn(a, b, c Point) float64 {
	u := Point{a.X - b.X, a.Y - b.Y}
	v := Point{c.X - b.X, c.Y - b.Y}
	d := math.Sqrt((u.X*u.X + u.Y*u.Y) * (v.X*v.X + v.Y*v.Y))
	if d == 0 {
		return 0
	}
	x := (u.X*v.X + u.Y*v.Y) / d
	x = math.Max(-1, math.Min(1, x))
	return math.Pi - math.Acos(x)
}
func segmentsCross(a, b, c, d Point) bool {
	cross := func(p, q, r Point) float64 { return (q.X-p.X)*(r.Y-p.Y) - (q.Y-p.Y)*(r.X-p.X) }
	return cross(a, b, c)*cross(a, b, d) < 0 && cross(c, d, a)*cross(c, d, b) < 0
}

func canonicalSeed(g []GeoPoint, c Context) uint64 {
	h := sha256.New()
	w := bufio.NewWriter(h)
	fmt.Fprint(w, Version, "|", walkArtV1.Version, "|", strings.ToLower(c.LocationLabel), "|", strings.ToLower(c.TimeOfDay), "|", math.Round(c.MusicTempo), "|", math.Round(c.MusicEnergy*100), "|", normalizeFamily(c.Family), "|", strings.ToLower(c.Palette))
	for _, p := range g {
		fmt.Fprintf(w, "|%.6f,%.6f,%.1f", p.Lat, p.Lon, p.Elevation)
	}
	w.Flush()
	return binary.LittleEndian.Uint64(h.Sum(nil)[:8])
}

type rng struct{ x uint64 }

func (r *rng) next() float64 {
	r.x ^= r.x << 13
	r.x ^= r.x >> 7
	r.x ^= r.x << 17
	return float64(r.x>>11) / float64(uint64(1)<<53)
}

func makeField(route []Point, f Features, c Context, seed uint64) []Line {
	r := rng{seed}
	energy := clamp(c.MusicEnergy, 0, 1)
	tempo := clamp((c.MusicTempo-60)/120, 0, 1)
	count := 260 + int(120*energy)
	steps := 62 + int(30*tempo)
	lines := make([]Line, 0, count+3)
	for i := 0; i < count; i++ {
		var p Point
		if i < count*2/3 {
			q := route[int(r.next()*float64(len(route)-1))]
			p = Point{q.X + (r.next()-.5)*(.09+.04*energy), q.Y + (r.next()-.5)*(.09+.04*energy)}
		} else {
			p = Point{.08 + .84*r.next(), .08 + .84*r.next()}
		}
		ln := Line{Width: .45 + 1.35*math.Pow(r.next(), 3), Opacity: .12 + .34*r.next(), Role: "field"}
		for k := 0; k < steps; k++ {
			if p.X < .035 || p.X > .965 || p.Y < .035 || p.Y > .965 {
				break
			}
			ln.Points = append(ln.Points, p)
			v := fieldAt(p, route, f, seed)
			p.X += v.X * (.0042 + .0015*energy)
			p.Y += v.Y * (.0042 + .0015*energy)
		}
		if len(ln.Points) > 9 {
			lines = append(lines, ln)
		}
	}
	lines = append(lines, Line{Points: route, Width: 2.8, Opacity: .92, Role: "route"})
	return lines
}
func fieldAt(p Point, route []Point, f Features, seed uint64) Point {
	best := 0
	bd := math.MaxFloat64
	for i := 0; i < len(route); i += 3 {
		d := (p.X-route[i].X)*(p.X-route[i].X) + (p.Y-route[i].Y)*(p.Y-route[i].Y)
		if d < bd {
			bd = d
			best = i
		}
	}
	a := route[max(0, best-2)]
	b := route[min(len(route)-1, best+2)]
	tx, ty := b.X-a.X, b.Y-a.Y
	mag := math.Hypot(tx, ty)
	if mag > 0 {
		tx /= mag
		ty /= mag
	}
	dx, dy := p.X-.5, p.Y-.5
	swirl := .22 + .28*f.LoopClosure
	phase := float64(seed%997) / 997 * math.Pi * 2
	noise := math.Sin(p.X*17+p.Y*11+phase) * .22
	vx := tx*.72 - dy*swirl + noise
	vy := ty*.72 + dx*swirl + math.Cos(p.Y*15-p.X*9+phase)*.22
	m := math.Hypot(vx, vy)
	if m == 0 {
		return Point{1, 0}
	}
	return Point{vx / m, vy / m}
}

func scoreLines(lines []Line) Score {
	profile := walkArtV1
	const n = 72
	cells := make([]int, n*n)
	edge := 0
	visible := 0
	minWidth, maxWidth := math.MaxFloat64, 0.0
	for _, l := range lines {
		if !isArtworkLine(l) {
			continue
		}
		visible++
		minWidth = math.Min(minWidth, l.Width)
		maxWidth = math.Max(maxWidth, l.Width)
		for _, p := range l.Points {
			x := int(p.X * n)
			y := int(p.Y * n)
			if x < 0 || x >= n || y < 0 || y >= n {
				continue
			}
			cells[y*n+x]++
			if x < 3 || x >= n-3 || y < 3 || y >= n-3 {
				edge++
			}
		}
	}
	used := 0
	left, right, top, bottom := 0, 0, 0, 0
	dense := 0
	veryDense := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := cells[y*n+x]
			if v > 0 {
				used++
			}
			if v > 4 {
				dense++
			}
			if v > 12 {
				veryDense++
			}
			if x < n/2 {
				left += v
			} else {
				right += v
			}
			if y < n/2 {
				top += v
			} else {
				bottom += v
			}
		}
	}
	coverage := float64(used) / float64(n*n)
	balance := 1 - (math.Abs(float64(left-right))/math.Max(1, float64(left+right))+math.Abs(float64(top-bottom))/math.Max(1, float64(top+bottom)))/2
	rawNegative := 1 - float64(dense)/float64(n*n)
	negativeTarget := clamp(1-math.Abs(rawNegative-profile.NegativeSpace.Target)/profile.NegativeSpace.Tolerance, 0, 1)
	safe := 1 - float64(edge)/math.Max(1, float64(left+right))
	complexity := clamp(float64(visible)/profile.ComplexityTarget, 0, 1)
	densityHierarchy := clamp(float64(veryDense)/math.Max(1, float64(dense))/profile.Hierarchy.DenseRatioTarget, 0, 1)
	widthHierarchy := clamp((maxWidth-minWidth)/profile.Hierarchy.WidthSpreadTarget, 0, 1)
	hierarchy := .55*densityHierarchy + .45*widthHierarchy
	routePoints, routeInside := 0, 0
	for _, l := range lines {
		if !strings.HasPrefix(l.Role, "route") {
			continue
		}
		routePoints += len(l.Points)
		for _, p := range l.Points {
			if p.X > .04 && p.X < .96 && p.Y > .04 && p.Y < .96 {
				routeInside++
			}
		}
	}
	legibility := float64(routeInside) / math.Max(1, float64(routePoints))
	total := .2*clamp(coverage/profile.CoverageTarget, 0, 1) + .14*balance + .18*negativeTarget + .17*hierarchy + .16*legibility + .09*safe + .06*complexity
	return Score{Total: round(total), Coverage: round(coverage), Balance: round(balance), NegativeSpace: round(negativeTarget), Hierarchy: round(hierarchy), RouteLegibility: round(legibility), EdgeSafety: round(safe), Complexity: round(complexity)}
}

func isArtworkLine(l Line) bool {
	return len(l.Points) > 1 && !strings.HasPrefix(l.Role, "route")
}

type palette struct {
	BG, Ink, Route, Accent color.RGBA
	Name                   string
}

func paletteFor(name string) palette {
	switch name {
	case "alpine-dawn":
		return palette{hexColor("E9F0DF"), hexColor("17352B"), hexColor("EF6A3A"), hexColor("799A64"), name}
	case "blue-hour":
		return palette{hexColor("DCE7E3"), hexColor("17313D"), hexColor("E7834B"), hexColor("557C8A"), name}
	case "high-desert":
		return palette{hexColor("F0E4CD"), hexColor("3A3528"), hexColor("D7572B"), hexColor("87945E"), name}
	case "canyon-light":
		return palette{hexColor("F4E3CB"), hexColor("493428"), hexColor("C95135"), hexColor("B88955"), name}
	case "lichen-signal":
		return palette{hexColor("E4E8D5"), hexColor("25372C"), hexColor("D95C39"), hexColor("7D984E"), name}
	case "storm-path":
		return palette{hexColor("D9E0DC"), hexColor("24343B"), hexColor("F07B4E"), hexColor("587584"), name}
	default:
		return palette{hexColor("E7E2D2"), hexColor("193B2C"), hexColor("E86238"), hexColor("708866"), "trail-moss"}
	}
}
func choosePalette(c Context, s uint64) string {
	t := strings.ToLower(c.TimeOfDay)
	if strings.Contains(t, "night") || strings.Contains(t, "evening") {
		return "blue-hour"
	}
	if strings.Contains(t, "dawn") || strings.Contains(t, "morning") {
		return "alpine-dawn"
	}
	if c.MusicEnergy > .72 {
		return "high-desert"
	}
	names := []string{"trail-moss", "alpine-dawn", "blue-hour"}
	return names[int(s%uint64(len(names)))]
}
func hexColor(s string) color.RGBA {
	v, _ := strconv.ParseUint(s, 16, 32)
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func WriteBundle(dir string, r Result) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := writeSVG(filepath.Join(dir, "artwork.svg"), r); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(dir, "preview.png"), r); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "fingerprint.json"), struct {
		ID       string       `json:"id"`
		Title    string       `json:"title"`
		Subtitle string       `json:"subtitle"`
		Features Features     `json:"features"`
		Events   []RouteEvent `json:"events"`
	}{r.ID, r.Title, r.Subtitle, r.Features, r.Events}); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "recipe.json"), r.Recipe); err != nil {
		return err
	}
	return nil
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
func writeSVG(path string, r Result) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	colors := resultColors(r)
	bg, ink := colors[0], colors[1]
	fmt.Fprintf(f, "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 1200 1500\" role=\"img\" aria-label=\"%s generative route artwork\">\n<rect width=\"1200\" height=\"1500\" fill=\"#%02x%02x%02x\"/>\n", xmlEscape(r.Title), bg.R, bg.G, bg.B)
	fmt.Fprintf(f, "<g fill=\"none\" stroke-linecap=\"round\" stroke-linejoin=\"round\">\n")
	for _, l := range r.Lines {
		if len(l.Points) < 2 {
			continue
		}
		role := max(1, min(5, l.ColorRole))
		col := colors[role]
		var d strings.Builder
		for i, q := range l.Points {
			x := 120 + q.X*960
			y := 180 + q.Y*1050
			if i == 0 {
				fmt.Fprintf(&d, "M%.1f %.1f", x, y)
			} else {
				fmt.Fprintf(&d, " L%.1f %.1f", x, y)
			}
		}
		if l.Texture == "ribbon" {
			fmt.Fprintf(f, "<path d=\"%s\" stroke=\"#%02x%02x%02x\" stroke-width=\"%.2f\" opacity=\"%.3f\"/>\n", d.String(), col.R, col.G, col.B, l.Width*3.8, l.Opacity*.2)
		}
		fmt.Fprintf(f, "<path d=\"%s\" stroke=\"#%02x%02x%02x\" stroke-width=\"%.2f\" opacity=\"%.3f\"%s/>\n", d.String(), col.R, col.G, col.B, l.Width*2, l.Opacity, svgTextureAttributes(l.Texture))
	}
	fmt.Fprintln(f, "</g>")
	fmt.Fprintf(f, "<text x=\"84\" y=\"1360\" fill=\"#%02x%02x%02x\" font-family=\"ui-monospace,monospace\" font-size=\"23\" letter-spacing=\"3\">%s</text>\n", ink.R, ink.G, ink.B, strings.ToUpper(xmlEscape(r.Title)))
	fmt.Fprintf(f, "<text x=\"84\" y=\"1405\" fill=\"#%02x%02x%02x\" font-family=\"ui-monospace,monospace\" font-size=\"15\" letter-spacing=\"2\" opacity=\".7\">%s · %s</text>\n</svg>\n", ink.R, ink.G, ink.B, strings.ToUpper(xmlEscape(r.Subtitle)), strings.ToUpper(r.Recipe.Seed[:8]))
	return nil
}

func svgTextureAttributes(texture string) string {
	switch texture {
	case "broken":
		return ` stroke-dasharray="14 8 3 7"`
	case "dry-brush":
		return ` stroke-dasharray="7 2 1 3" stroke-linecap="butt"`
	case "thread":
		return ` stroke-dasharray="1.5 4.5"`
	case "charcoal":
		return ` stroke-dasharray="5 2" stroke-linecap="square"`
	case "route-memory":
		return ` stroke-dasharray="18 10 2 8"`
	default:
		return ""
	}
}
func writePNG(path string, r Result) error {
	const w, h = 600, 750
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	colors := resultColors(r)
	bg := colors[0]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	for _, l := range r.Lines {
		role := max(1, min(5, l.ColorRole))
		col := colors[role]
		col.A = uint8(clamp(l.Opacity, 0, 1) * 255)
		drawTexturedLine(img, l, col)
	}
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawTexturedLine(img *image.RGBA, l Line, col color.RGBA) {
	if l.Texture == "ribbon" {
		under := col
		under.A = uint8(float64(col.A) * .2)
		for i := 1; i < len(l.Points); i++ {
			drawLine(img, mapPixel(l.Points[i-1]), mapPixel(l.Points[i]), under, math.Max(1, l.Width*1.9))
		}
	}
	for i := 1; i < len(l.Points); i++ {
		if !textureSegmentVisible(l.Texture, i) {
			continue
		}
		drawLine(img, mapPixel(l.Points[i-1]), mapPixel(l.Points[i]), col, math.Max(1, l.Width))
	}
}

func textureSegmentVisible(texture string, segment int) bool {
	switch texture {
	case "broken":
		return segment%16 < 10
	case "dry-brush":
		return segment%9 != 0 && segment%9 != 4
	case "thread":
		return segment%5 < 2
	case "charcoal":
		return segment%7 < 5
	case "route-memory":
		return segment%20 < 13 || segment%20 == 16
	default:
		return true
	}
}
func resultColors(r Result) [6]color.RGBA {
	if r.Colors[0].A != 0 {
		return r.Colors
	}
	p := paletteFor(r.Palette)
	return [6]color.RGBA{p.BG, p.Ink, p.Route, p.Accent, p.Ink, p.Route}
}
func mapPixel(p Point) image.Point { return image.Pt(int(60+p.X*480), int(90+p.Y*525)) }
func drawLine(img *image.RGBA, a, b image.Point, c color.RGBA, width float64) {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	n := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if n < 1 {
		n = 1
	}
	rad := int(math.Ceil(width / 2))
	for i := 0; i <= n; i++ {
		x := a.X + int(dx*float64(i)/float64(n))
		y := a.Y + int(dy*float64(i)/float64(n))
		for oy := -rad; oy <= rad; oy++ {
			for ox := -rad; ox <= rad; ox++ {
				if ox*ox+oy*oy > rad*rad {
					continue
				}
				blend(img, x+ox, y+oy, c)
			}
		}
	}
}
func blend(img *image.RGBA, x, y int, c color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	d := img.RGBAAt(x, y)
	a := float64(c.A) / 255
	img.SetRGBA(x, y, color.RGBA{uint8(float64(c.R)*a + float64(d.R)*(1-a)), uint8(float64(c.G)*a + float64(d.G)*(1-a)), uint8(float64(c.B)*a + float64(d.B)*(1-a)), 255})
}
func timeLabel(v string) string {
	if v == "" {
		return "time unrecorded"
	}
	return v
}
func xmlEscape(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
func dist(a, b Point) float64       { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func clamp(v, a, b float64) float64 { return math.Max(a, math.Min(b, v)) }
func round(v float64) float64       { return math.Round(v*1000) / 1000 }

func ListArtifacts(root string) ([]map[string]any, error) {
	ents, e := os.ReadDir(root)
	if e != nil {
		return nil, e
	}
	var out []map[string]any
	for _, x := range ents {
		if !x.IsDir() {
			continue
		}
		b, e := os.ReadFile(filepath.Join(root, x.Name(), "fingerprint.json"))
		if e != nil {
			continue
		}
		var v map[string]any
		if json.Unmarshal(b, &v) == nil {
			v["id"] = x.Name()
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]["title"]) < fmt.Sprint(out[j]["title"]) })
	return out, nil
}
