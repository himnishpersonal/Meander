package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type generatedCandidate struct {
	index     int
	family    string
	palette   string
	lines     []Line
	score     Score
	signature []float64
}

func buildCandidates(route []Point, events []RouteEvent, f Features, c Context, seed uint64) []generatedCandidate {
	families := []string{"flow-field", "topographic-relief", "movement-weave"}
	if normalized := normalizeFamily(c.Family); normalized != "auto" {
		families = []string{normalized}
	}
	count := 18
	out := make([]generatedCandidate, 0, count)
	for i := 0; i < count; i++ {
		family := families[i%len(families)]
		s := seed + uint64(i)*0x9e3779b97f4a7c15
		lines := makeComposition(family, route, events, f, c, s)
		out = append(out, generatedCandidate{i, family, choosePaletteForFamily(c, seed, family), lines, scoreLines(lines), occupancySignature(lines)})
	}
	return out
}

func selectFinalists(all []generatedCandidate) []generatedCandidate {
	if len(all) == 0 {
		return nil
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].score.Total > all[j].score.Total })
	selected := []generatedCandidate{all[0]}
	used := map[int]bool{all[0].index: true}
	for len(selected) < 3 && len(selected) < len(all) {
		selectedFamilies := map[string]bool{}
		for _, chosen := range selected {
			selectedFamilies[chosen.family] = true
		}
		unusedFamilyAvailable := false
		for _, candidate := range all {
			if !used[candidate.index] && !selectedFamilies[candidate.family] {
				unusedFamilyAvailable = true
				break
			}
		}
		bestIndex, bestValue := -1, -1.0
		for i := range all {
			if used[all[i].index] {
				continue
			}
			if unusedFamilyAvailable && selectedFamilies[all[i].family] {
				continue
			}
			minD := 1.0
			familyBonus := 0.0
			for _, chosen := range selected {
				minD = math.Min(minD, signatureDistance(all[i].signature, chosen.signature))
				if all[i].family != chosen.family {
					familyBonus = .1
				}
			}
			value := .68*all[i].score.Total + .22*minD + familyBonus
			if value > bestValue {
				bestValue, bestIndex = value, i
			}
		}
		if bestIndex < 0 {
			break
		}
		selected = append(selected, all[bestIndex])
		used[all[bestIndex].index] = true
	}
	return selected
}

func candidateID(seed uint64, candidate int, family string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s", seed, candidate, family, Version)))
	return hex.EncodeToString(h[:6])
}

func makeComposition(family string, route []Point, events []RouteEvent, f Features, c Context, seed uint64) []Line {
	switch family {
	case "topographic-relief":
		return makeRelief(route, events, f, c, seed)
	case "movement-weave":
		return makeWeave(route, events, f, c, seed)
	default:
		return makeFlow(route, events, f, c, seed)
	}
}

func makeFlow(route []Point, events []RouteEvent, f Features, c Context, seed uint64) []Line {
	r := rng{seed}
	energy := clamp(c.MusicEnergy, 0, 1)
	tempo := clamp((c.MusicTempo-60)/120, 0, 1)
	count := 250 + int(120*energy)
	steps := 62 + int(30*tempo)
	lines := make([]Line, 0, count+1)
	for i := 0; i < count; i++ {
		var p Point
		if i < count*3/4 {
			q := route[int(r.next()*float64(len(route)-1))]
			p = Point{q.X + (r.next()-.5)*(.08+.05*energy), q.Y + (r.next()-.5)*(.08+.05*energy)}
		} else {
			p = Point{.08 + .84*r.next(), .08 + .84*r.next()}
		}
		ln := Line{Width: .42 + 1.45*math.Pow(r.next(), 3), Opacity: .1 + .38*r.next(), Role: "field"}
		for k := 0; k < steps; k++ {
			if p.X < .035 || p.X > .965 || p.Y < .035 || p.Y > .965 {
				break
			}
			ln.Points = append(ln.Points, p)
			v := fieldWithEvents(p, route, events, f, seed)
			p.X += v.X * (.0041 + .0015*energy)
			p.Y += v.Y * (.0041 + .0015*energy)
		}
		if len(ln.Points) > 9 {
			lines = append(lines, ln)
		}
	}
	return append(lines, Line{Points: route, Width: 2.9, Opacity: .94, Role: "route"})
}

func makeRelief(route []Point, events []RouteEvent, f Features, c Context, seed uint64) []Line {
	r := rng{seed}
	energy := clamp(c.MusicEnergy, 0, 1)
	lines := make([]Line, 0, 180)
	for band := -22; band <= 22; band++ {
		offset := float64(band) * (.0032 + .0008*energy)
		ln := Line{Width: .5 + math.Abs(float64(band%5))*.08, Opacity: .16 + .22*(1-math.Abs(float64(band))/24), Role: "field"}
		for i := 1; i < len(route)-1; i++ {
			a, b := route[i-1], route[i+1]
			dx, dy := b.X-a.X, b.Y-a.Y
			m := math.Hypot(dx, dy)
			if m == 0 {
				continue
			}
			eventLift := 0.0
			for _, e := range events {
				d := dist(route[i], e.Position)
				if d < .12 {
					eventLift += math.Exp(-d*d/.004) * e.Strength * .008
				}
			}
			wav := math.Sin(float64(i)*.19+float64(seed%37)) * eventLift
			ln.Points = append(ln.Points, Point{route[i].X - dy/m*(offset+wav), route[i].Y + dx/m*(offset+wav)})
		}
		if len(ln.Points) > 8 {
			lines = append(lines, ln)
		}
	}
	for _, e := range events {
		rings := 3 + int(e.Strength*3)
		for k := 1; k <= rings; k++ {
			ln := Line{Width: .65, Opacity: .2 + .1*e.Strength, Role: "accent"}
			rad := float64(k) * (.009 + .004*r.next())
			for j := 0; j <= 40; j++ {
				a := float64(j) / 40 * math.Pi * 2
				ln.Points = append(ln.Points, Point{e.Position.X + math.Cos(a)*rad, e.Position.Y + math.Sin(a)*rad*.72})
			}
			lines = append(lines, ln)
		}
	}
	return append(lines, Line{Points: route, Width: 2.7, Opacity: .95, Role: "route"})
}

func makeWeave(route []Point, events []RouteEvent, f Features, c Context, seed uint64) []Line {
	r := rng{seed}
	energy := clamp(c.MusicEnergy, 0, 1)
	lines := make([]Line, 0, 190)
	rows := 78 + int(34*energy)
	for row := 0; row < rows; row++ {
		y := .08 + .84*float64(row)/float64(rows-1)
		ln := Line{Width: .38 + 1.1*math.Pow(r.next(), 4), Opacity: .11 + .28*r.next(), Role: "field"}
		for step := 0; step <= 110; step++ {
			x := .06 + .88*float64(step)/110
			p := Point{x, y}
			near, idx := nearestRoute(p, route)
			pull := math.Exp(-dist(p, near) * dist(p, near) / .008)
			warp := (near.Y - y) * pull * (.35 + .25*energy)
			warp += math.Sin(x*20+float64(row)*.13+float64(seed%31)) * .003
			for _, e := range events {
				d := dist(p, e.Position)
				if d < .1 {
					warp += (e.Position.Y - y) * e.Strength * .12 * (1 - d/.1)
				}
			}
			_ = idx
			ln.Points = append(ln.Points, Point{x, y + warp})
		}
		lines = append(lines, ln)
	}
	for strand := 0; strand < 36; strand++ {
		base := route[int(float64(strand)/35*float64(len(route)-1))]
		ln := Line{Width: .45, Opacity: .16 + .18*r.next(), Role: "accent"}
		for j := 0; j <= 70; j++ {
			t := float64(j) / 70
			ln.Points = append(ln.Points, Point{base.X + (t-.5)*.12 + math.Sin(t*14+float64(strand))*.004, .08 + t*.84})
		}
		lines = append(lines, ln)
	}
	return append(lines, Line{Points: route, Width: 3.1, Opacity: .96, Role: "route"})
}

func fieldWithEvents(p Point, route []Point, events []RouteEvent, f Features, seed uint64) Point {
	base := fieldAt(p, route, f, seed)
	vx, vy := base.X, base.Y
	for _, e := range events {
		dx, dy := p.X-e.Position.X, p.Y-e.Position.Y
		d2 := dx*dx + dy*dy
		if d2 > .025 || d2 < 1e-8 {
			continue
		}
		fall := math.Exp(-d2/.006) * e.Strength
		switch e.Kind {
		case "turn":
			vx += -dy * fall * 8
			vy += dx * fall * 8
		case "pause":
			vx += -dx * fall * 3
			vy += -dy * fall * 3
		case "climb":
			vy -= fall * .6
		case "descent":
			vy += fall * .5
		case "pace":
			vx += dx * fall * 2
			vy += dy * fall * 2
		}
	}
	m := math.Hypot(vx, vy)
	if m == 0 {
		return Point{1, 0}
	}
	return Point{vx / m, vy / m}
}

func detectEvents(g []GeoPoint, route []Point) []RouteEvent {
	var events []RouteEvent
	for i := 3; i < len(route)-3; i += 2 {
		a := turn(route[i-3], route[i], route[i+3])
		if a > .24 {
			events = append(events, RouteEvent{"turn", route[i], clamp(a/1.4, .25, 1)})
		}
	}
	if len(g) > 2 {
		for i := 1; i < len(g); i++ {
			pos := route[min(len(route)-1, int(float64(i)/float64(len(g)-1)*float64(len(route)-1)))]
			if !g[i].Time.IsZero() && !g[i-1].Time.IsZero() {
				dt := g[i].Time.Sub(g[i-1].Time).Seconds()
				if dt > 75 {
					events = append(events, RouteEvent{"pause", pos, clamp(dt/300, .3, 1)})
				}
				if i > 1 && !g[i-2].Time.IsZero() {
					prev := g[i-1].Time.Sub(g[i-2].Time).Seconds()
					if prev > 0 && dt > 0 && (dt/prev > 2.2 || prev/dt > 2.2) {
						events = append(events, RouteEvent{"pace", pos, .55})
					}
				}
			}
			de := g[i].Elevation - g[i-1].Elevation
			if math.Abs(de) > 4 {
				kind := "climb"
				if de < 0 {
					kind = "descent"
				}
				events = append(events, RouteEvent{kind, pos, clamp(math.Abs(de)/18, .25, 1)})
			}
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Strength > events[j].Strength })
	if len(events) > 18 {
		events = events[:18]
	}
	return events
}

func enrichFeatures(g []GeoPoint, f *Features) {
	var first, last time.Time
	var speeds []float64
	for i, p := range g {
		if p.Time.IsZero() {
			continue
		}
		f.TimedPoints++
		if first.IsZero() {
			first = p.Time
		}
		last = p.Time
		if i > 0 && !g[i-1].Time.IsZero() {
			dt := p.Time.Sub(g[i-1].Time).Seconds()
			if dt > 0 {
				a, _ := project([]GeoPoint{g[i-1], p})
				speeds = append(speeds, dist(a[0], a[1])/dt)
				if dt > 75 {
					f.DwellPoints++
				}
			}
		}
	}
	if !first.IsZero() && last.After(first) {
		f.DurationMinutes = last.Sub(first).Minutes()
		if f.DistanceKM > 0 {
			f.MeanPaceMinKM = f.DurationMinutes / f.DistanceKM
		}
	}
	for i := 1; i < len(speeds); i++ {
		if speeds[i] > .15 && speeds[i-1] > .15 && (speeds[i]/speeds[i-1] > 1.8 || speeds[i-1]/speeds[i] > 1.8) {
			f.PaceChanges++
		}
	}
}

func nearestRoute(p Point, route []Point) (Point, int) {
	best := route[0]
	idx := 0
	bd := math.MaxFloat64
	for i := 0; i < len(route); i += 2 {
		d := dist(p, route[i])
		if d < bd {
			bd = d
			best = route[i]
			idx = i
		}
	}
	return best, idx
}
func normalizeFamily(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "flow", "flow-field":
		return "flow-field"
	case "relief", "topographic-relief":
		return "topographic-relief"
	case "weave", "movement-weave":
		return "movement-weave"
	default:
		return "auto"
	}
}

func occupancySignature(lines []Line) []float64 {
	const n = 12
	s := make([]float64, n*n)
	for _, l := range lines {
		if l.Role == "route" {
			continue
		}
		for _, p := range l.Points {
			x, y := int(p.X*n), int(p.Y*n)
			if x >= 0 && x < n && y >= 0 && y < n {
				s[y*n+x]++
			}
		}
	}
	maxV := 0.0
	for _, v := range s {
		maxV = math.Max(maxV, v)
	}
	if maxV > 0 {
		for i := range s {
			s[i] /= maxV
		}
	}
	return s
}
func signatureDistance(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 1
	}
	sum := 0.0
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return clamp(sum/float64(len(a))/.5, 0, 1)
}

func choosePaletteForFamily(c Context, seed uint64, family string) string {
	if c.Palette != "" && c.Palette != "auto" {
		return c.Palette
	}
	p := choosePalette(c, seed)
	if family == "topographic-relief" && strings.ToLower(c.TimeOfDay) == "afternoon" {
		return "canyon-light"
	}
	if family == "movement-weave" && c.MusicEnergy > .65 {
		return "lichen-signal"
	}
	return p
}
