package engine

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"
)

type turbulenceCandidate struct {
	index       int
	lines       []Line
	composition Composition
	score       Score
}

type colorTerritory struct {
	center Point
	role   int
	radius float64
}

func buildTurbulenceCandidates(route []Point, events []RouteEvent, f Features, c Context, seed uint64, profile CalibrationProfile) []turbulenceCandidate {
	out := make([]turbulenceCandidate, 0, 28)
	for i := 0; i < 28; i++ {
		s := seed + uint64(i)*0x9e3779b97f4a7c15
		lines, composition := makeTurbulence(route, events, f, c, s, profile)
		out = append(out, turbulenceCandidate{i, lines, composition, scoreTurbulence(lines, route, composition, profile)})
	}
	return out
}

func chooseCanonical(candidates []turbulenceCandidate) (turbulenceCandidate, bool) {
	if len(candidates) == 0 {
		return turbulenceCandidate{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score.Total > candidates[j].score.Total })
	return candidates[0], true
}

func makeTurbulence(route []Point, events []RouteEvent, f Features, c Context, seed uint64, profile CalibrationProfile) ([]Line, Composition) {
	r := rng{seed}
	energy := clamp(c.MusicEnergy, 0, 1)
	wild := clamp(.48+.18*energy+.16*clamp(f.MeanCurvature/.2, 0, 1)+.12*clamp(float64(f.SelfIntersections)/8, 0, 1), .42, .94)
	territories := makeTerritories(route, events, seed)
	count := 390 + int(170*wild)
	steps := 80 + int(55*wild)
	lines := make([]Line, 0, count+80)
	occ := make([]uint8, 180*180)
	for i := 0; i < count; i++ {
		p := seedPoint(&r, route, events, i, count)
		role := territoryRole(p, territories, &r)
		width := .32 + 1.5*math.Pow(r.next(), 4)
		opacity := .12 + .4*r.next()
		if i%17 == 0 {
			width = 4.5 + 6*r.next()
			opacity = .08 + .15*r.next()
			role = 2 + int(r.next()*2)
		} else if i%9 == 0 {
			width = 1.5 + 2.5*r.next()
			opacity = .13 + .22*r.next()
		}
		ln := Line{Width: width, Opacity: opacity, Role: "field", ColorRole: role}
		fracture := false
		for k := 0; k < steps; k++ {
			if p.X < -0.2 || p.X > 1.2 || p.Y < -0.2 || p.Y > 1.2 {
				break
			}
			if p.X >= 0 && p.X <= 1 && p.Y >= 0 && p.Y <= 1 {
				gx, gy := int(p.X*179), int(p.Y*179)
				idx := gy*180 + gx
				if occ[idx] > 5 && k > 12 && i%17 != 0 {
					break
				}
				if k%3 == 0 && occ[idx] < 255 {
					occ[idx]++
				}
			}
			ln.Points = append(ln.Points, p)
			v, stress := turbulenceVector(p, route, events, wild, seed)
			step := .0034 + .0022*wild
			p.X += v.X * step
			p.Y += v.Y * step
			if stress > .7 && k > 15 && r.next() < .025+.04*wild {
				fracture = true
				break
			}
		}
		if len(ln.Points) > 10 {
			if fracture {
				ln.Opacity = math.Min(.65, ln.Opacity*1.3)
			}
			lines = append(lines, ln)
		}
	}
	// Event fragments create rare chromatic ruptures without redrawing the route.
	for i, e := range events {
		if i > 11 {
			break
		}
		pieces := 3 + int(5*e.Strength)
		for j := 0; j < pieces; j++ {
			a := r.next() * math.Pi * 2
			length := .015 + .07*r.next()*e.Strength
			start := Point{e.Position.X + (r.next()-.5)*.04, e.Position.Y + (r.next()-.5)*.04}
			end := Point{start.X + math.Cos(a)*length, start.Y + math.Sin(a)*length}
			role := 3
			if j%4 == 0 {
				role = 5
			}
			lines = append(lines, Line{Points: []Point{start, end}, Width: .8 + 4*r.next(), Opacity: .28 + .42*r.next(), Role: "event-fragment", Texture: "charcoal", ColorRole: role})
		}
	}
	anchor := chooseCompositionAnchor(route, events, profile)
	composition := classifyTrails(lines, route, anchor, profile)
	// A restrained, interrupted route-memory keeps the source walk discoverable
	// without allowing the GPS trace to dominate the field.
	lines = append(lines, Line{Points: route, Width: .72, Opacity: .22, Role: "route-memory", Texture: "route-memory", ColorRole: 1})
	return lines, composition
}

func chooseCompositionAnchor(route []Point, events []RouteEvent, profile CalibrationProfile) CompositionAnchor {
	anchor := CompositionAnchor{Position: route[len(route)/2], Source: "route-curvature", Strength: .35}
	if len(events) > 0 {
		anchor = CompositionAnchor{Position: events[0].Position, Source: events[0].Kind, Strength: events[0].Strength}
	} else {
		best := 0.0
		for i := 3; i < len(route)-3; i++ {
			strength := turn(route[i-3], route[i], route[i+3])
			if strength > best {
				best = strength
				anchor.Position = route[i]
				anchor.Strength = clamp(strength/1.4, .25, 1)
			}
		}
	}
	margin := profile.Anchor.SafeMargin
	anchor.Position.X = clamp(anchor.Position.X, margin, 1-margin)
	anchor.Position.Y = clamp(anchor.Position.Y, margin, 1-margin)
	return anchor
}

type rankedTrail struct {
	index int
	score float64
}

func classifyTrails(lines []Line, route []Point, anchor CompositionAnchor, profile CalibrationProfile) Composition {
	ranked := make([]rankedTrail, 0, len(lines))
	for i := range lines {
		if lines[i].Role != "field" || len(lines[i].Points) < 10 {
			continue
		}
		lines[i].Role = "ambient"
		ranked = append(ranked, rankedTrail{i, trailHeroSignal(lines[i], route, anchor, profile)})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	heroCount := profile.Hierarchy.HeroMin
	if len(ranked) > 300 {
		heroCount++
	}
	heroCount = min(profile.Hierarchy.HeroMax, min(heroCount, len(ranked)))
	remaining := max(0, len(ranked)-heroCount)
	supportCount := int(math.Round(float64(remaining) * profile.Hierarchy.SupportingFraction))
	supportCount = min(remaining, max(profile.Hierarchy.SupportingMin, supportCount))
	for i, item := range ranked {
		switch {
		case i < heroCount:
			lines[item.index].Role = "hero"
			lines[item.index].Texture = "ribbon"
			lines[item.index].Width = clamp(lines[item.index].Width*profile.Hierarchy.HeroWidthMultiplier, 1.8, 12)
			lines[item.index].Opacity = clamp(lines[item.index].Opacity*1.28, .22, .72)
		case i < heroCount+supportCount:
			lines[item.index].Role = "supporting"
			if i%2 == 0 {
				lines[item.index].Texture = "broken"
			} else {
				lines[item.index].Texture = "thread"
			}
			lines[item.index].Width = clamp(lines[item.index].Width*1.1, .45, 5.5)
		default:
			if i%3 == 0 {
				lines[item.index].Texture = "dry-brush"
			} else {
				lines[item.index].Texture = "hairline"
			}
		}
	}
	return Composition{CalibrationProfile: profile.Version, Anchor: anchor, HeroTrails: heroCount, SupportingTrails: supportCount, AmbientTrails: remaining - supportCount}
}

func trailHeroSignal(line Line, route []Point, anchor CompositionAnchor, profile CalibrationProfile) float64 {
	length := 0.0
	minX, maxX, minY, maxY := line.Points[0].X, line.Points[0].X, line.Points[0].Y, line.Points[0].Y
	anchorDistance := math.MaxFloat64
	routeAffinity := 0.0
	samples := 0
	for i, p := range line.Points {
		if i > 0 {
			length += dist(line.Points[i-1], p)
		}
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		anchorDistance = math.Min(anchorDistance, dist(p, anchor.Position))
		if i%8 == 0 {
			near, _ := nearestRoute(p, route)
			routeAffinity += math.Exp(-math.Pow(dist(p, near), 2) / .02)
			samples++
		}
	}
	span := math.Hypot(maxX-minX, maxY-minY)
	anchorAffinity := math.Exp(-math.Pow(anchorDistance/profile.Anchor.InfluenceRadius, 2))
	return .27*clamp(length/.7, 0, 1) + .22*clamp(span/.55, 0, 1) + .22*routeAffinity/math.Max(1, float64(samples)) + .19*anchorAffinity + .1*clamp(line.Width/8, 0, 1)
}

func seedPoint(r *rng, route []Point, events []RouteEvent, i, _ int) Point {
	u := r.next()
	if u < .5 {
		q := route[int(r.next()*float64(len(route)-1))]
		spread := .1 + .24*math.Pow(r.next(), 2)
		return Point{q.X + (r.next()-.5)*spread, q.Y + (r.next()-.5)*spread}
	}
	if u < .76 && len(events) > 0 {
		e := events[int(r.next()*float64(len(events)))]
		return Point{e.Position.X + (r.next()-.5)*.16, e.Position.Y + (r.next()-.5)*.16}
	}
	// Seeds beyond the crop let currents enter and leave the frame.
	side := i % 4
	t := r.next()*1.3 - .15
	switch side {
	case 0:
		return Point{-.12, t}
	case 1:
		return Point{1.12, t}
	case 2:
		return Point{t, -.12}
	default:
		return Point{t, 1.12}
	}
}

func turbulenceVector(p Point, route []Point, events []RouteEvent, wild float64, seed uint64) (Point, float64) {
	near, idx := nearestRoute(p, route)
	a := route[max(0, idx-3)]
	b := route[min(len(route)-1, idx+3)]
	tx, ty := b.X-a.X, b.Y-a.Y
	m := math.Hypot(tx, ty)
	if m > 0 {
		tx /= m
		ty /= m
	}
	start, end := route[0], route[len(route)-1]
	gx, gy := end.X-start.X, end.Y-start.Y
	gm := math.Hypot(gx, gy)
	if gm < .04 {
		gx, gy = -ty, tx
		gm = 1
	}
	gx /= gm
	gy /= gm
	phase := float64(seed%1009) / 1009 * math.Pi * 2
	// Multi-scale analytic curl: large bends, medium eddies, fine nervous motion.
	c1 := math.Sin(p.X*5.7 + p.Y*4.1 + phase)
	c2 := math.Sin(p.X*15.3 - p.Y*11.7 - phase*.7)
	c3 := math.Cos(p.X*39.1 + p.Y*31.3 + phase*1.9)
	angle := c1*.82 + c2*.34 + c3*.1
	d := dist(p, near)
	routePull := math.Exp(-d * d / .028)
	vx := tx*(.18+.78*routePull) + gx*.26 + math.Cos(angle*math.Pi)*wild*.72
	vy := ty*(.18+.78*routePull) + gy*.26 + math.Sin(angle*math.Pi)*wild*.72
	stress := 0.0
	for _, e := range events {
		dx, dy := p.X-e.Position.X, p.Y-e.Position.Y
		d2 := dx*dx + dy*dy
		if d2 > .035 || d2 < 1e-8 {
			continue
		}
		fall := math.Exp(-d2/.006) * e.Strength
		stress = math.Max(stress, fall)
		switch e.Kind {
		case "turn":
			vx += -dy * fall * 12
			vy += dx * fall * 12
		case "pause":
			vx -= dx * fall * 5
			vy -= dy * fall * 5
		case "climb":
			vy -= fall * .8
			vx += -dy * fall * 2
		case "descent":
			vy += fall * .7
		case "pace":
			vx += dx * fall * 3
			vy += dy * fall * 3
		}
	}
	if stress > .55 {
		q := math.Pi / 5
		ang := math.Atan2(vy, vx)
		ang = math.Round(ang/q) * q
		vx, vy = math.Cos(ang), math.Sin(ang)
	}
	m = math.Hypot(vx, vy)
	if m == 0 {
		return Point{1, 0}, stress
	}
	return Point{vx / m, vy / m}, stress
}

func makeTerritories(route []Point, events []RouteEvent, seed uint64) []colorTerritory {
	t := []colorTerritory{{route[len(route)/12], 2, .24}, {route[len(route)*4/12], 3, .2}, {route[len(route)*8/12], 4, .22}, {route[len(route)*11/12], 2, .18}}
	for i, e := range events {
		if i > 5 {
			break
		}
		role := 3 + (i % 2)
		t = append(t, colorTerritory{e.Position, role, .1 + .12*e.Strength})
	}
	return t
}
func territoryRole(p Point, t []colorTerritory, r *rng) int {
	bestRole := 2
	best := math.MaxFloat64
	for _, x := range t {
		d := dist(p, x.center) / x.radius
		if d < best {
			best = d
			bestRole = x.role
		}
	}
	if r.next() < .07 {
		return 1
	}
	if r.next() < .025 {
		return 5
	}
	return bestRole
}

func scoreTurbulence(lines []Line, route []Point, composition Composition, profile CalibrationProfile) Score {
	base := scoreLines(lines)
	visible := 0
	roles := [6]int{}
	alignment := 0.0
	segments := 0
	heroWidth, supportingWidth := 0.0, 0.0
	heroes, supporting := 0, 0
	heroAnchorDistance := math.MaxFloat64
	grid := make([]int, 24*24)
	for _, l := range lines {
		if !isArtworkLine(l) {
			continue
		}
		visible++
		if l.Role == "hero" {
			heroes++
			heroWidth += l.Width
			for _, p := range l.Points {
				heroAnchorDistance = math.Min(heroAnchorDistance, dist(p, composition.Anchor.Position))
			}
		} else if l.Role == "supporting" {
			supporting++
			supportingWidth += l.Width
		}
		if l.ColorRole >= 0 && l.ColorRole < 6 {
			roles[l.ColorRole]++
		}
		if l.Role == "event-fragment" {
			continue
		}
		for i := 1; i < len(l.Points); i += 4 {
			a, b := l.Points[i-1], l.Points[i]
			dx, dy := b.X-a.X, b.Y-a.Y
			m := math.Hypot(dx, dy)
			if m > 0 {
				_, idx := nearestRoute(a, route)
				ra := route[max(0, idx-2)]
				rb := route[min(len(route)-1, idx+2)]
				rx, ry := rb.X-ra.X, rb.Y-ra.Y
				rm := math.Hypot(rx, ry)
				if rm > 0 {
					alignment += math.Abs((dx*rx + dy*ry) / (m * rm))
					segments++
				}
			}
			if a.X >= 0 && a.X < 1 && a.Y >= 0 && a.Y < 1 {
				grid[int(a.Y*24)*24+int(a.X*24)]++
			}
		}
	}
	directional := alignment / math.Max(1, float64(segments))
	directionalQuality := clamp(1-math.Abs(directional-profile.Directional.Target)/profile.Directional.Tolerance, 0, 1)
	body := roles[2] + roles[3] + roles[4]
	topTwo := roles[2] + roles[3]
	colorStructure := clamp(float64(topTwo)/math.Max(1, float64(body))/profile.Color.DominantRatioTarget, 0, 1)
	flare := float64(roles[5]) / math.Max(1, float64(visible))
	accent := clamp(1-math.Abs(flare-profile.Color.FlareTarget)/profile.Color.FlareTolerance, 0, 1)
	mean := 0.0
	maxV := 0
	for _, v := range grid {
		mean += float64(v)
		if v > maxV {
			maxV = v
		}
	}
	mean /= float64(len(grid))
	focal := clamp((float64(maxV)/math.Max(1, mean)-profile.Focal.BaselineRatio)/profile.Focal.Scale, 0, 1)
	averageHero := heroWidth / math.Max(1, float64(heroes))
	averageSupporting := supportingWidth / math.Max(1, float64(supporting))
	heroSupport := clamp((averageHero/math.Max(.1, averageSupporting)-1)/(profile.Hierarchy.HeroWidthMultiplier-1), 0, 1)
	if heroes < profile.Hierarchy.HeroMin || supporting < profile.Hierarchy.SupportingMin {
		heroSupport = 0
	}
	anchorStrength := math.Exp(-math.Pow(heroAnchorDistance/profile.Anchor.InfluenceRadius, 2))
	if heroes == 0 {
		anchorStrength = 0
	}
	base.Hierarchy = round(.55*base.Hierarchy + .45*heroSupport)
	base.RouteLegibility = round(directional)
	base.ColorStructure = round(colorStructure)
	base.AccentDiscipline = round(accent)
	base.FocalStrength = round(focal)
	base.HeroSupport = round(heroSupport)
	base.AnchorStrength = round(anchorStrength)
	w := profile.Weights
	base.Total = round(w.NegativeSpace*base.NegativeSpace + w.Hierarchy*base.Hierarchy + w.Direction*directionalQuality + w.ColorStructure*colorStructure + w.AccentDiscipline*accent + w.FocalStrength*focal + w.HeroSupport*heroSupport + w.AnchorStrength*anchorStrength + w.EdgeSafety*base.EdgeSafety + w.Balance*base.Balance)
	return base
}

func routePalette(c Context, seed uint64) (string, [6]color.RGBA) {
	base := float64(seed % 360)
	switch strings.ToLower(c.TimeOfDay) {
	case "dawn":
		base = 18 + float64(seed%28)
	case "morning":
		base = 52 + float64(seed%38)
	case "afternoon":
		base = 88 + float64(seed%55)
	case "evening":
		base = 258 + float64(seed%44)
	case "night":
		base = 218 + float64(seed%36)
	}
	energy := clamp(c.MusicEnergy, 0, 1)
	colors := [6]color.RGBA{oklch(.94, .025, base+35), oklch(.23, .07, base+185), oklch(.55, .15+.05*energy, base), oklch(.64, .16+.04*energy, base+62), oklch(.77, .1+.035*energy, base+178), oklch(.62, .22+.04*energy, base+22)}
	return fmt.Sprintf("route-spectrum-%03d", int(math.Mod(base, 360))), colors
}

func oklch(l, c, h float64) color.RGBA {
	a := c * math.Cos(h*math.Pi/180)
	b := c * math.Sin(h*math.Pi/180)
	ll := l + 0.3963377774*a + 0.2158037573*b
	mm := l - 0.1055613458*a - 0.0638541728*b
	ss := l - 0.0894841775*a - 1.291485548*b
	ll *= ll * ll
	mm *= mm * mm
	ss *= ss * ss
	rr := 4.0767416621*ll - 3.3077115913*mm + 0.2309699292*ss
	gg := -1.2684380046*ll + 2.6097574011*mm - 0.3413193965*ss
	bb := -0.0041960863*ll - 0.7034186147*mm + 1.707614701*ss
	gamma := func(v float64) float64 {
		if v <= .0031308 {
			return 12.92 * v
		}
		return 1.055*math.Pow(v, 1/2.4) - .055
	}
	return color.RGBA{uint8(clamp(gamma(rr), 0, 1) * 255), uint8(clamp(gamma(gg), 0, 1) * 255), uint8(clamp(gamma(bb), 0, 1) * 255), 255}
}
