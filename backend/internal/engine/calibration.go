package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// calibrationJSON is a build-time global profile. Generation never queries a
// user account, personal history, or per-user percentiles.
//
//go:embed calibration/walk-art-v1.json
var calibrationJSON []byte

type targetRange struct {
	Target       float64 `json:"target"`
	Tolerance    float64 `json:"tolerance"`
	MinimumScore float64 `json:"minimum_score"`
}

type CalibrationProfile struct {
	Version          string      `json:"version"`
	Scope            string      `json:"scope"`
	CoverageTarget   float64     `json:"coverage_target"`
	ComplexityTarget float64     `json:"complexity_target"`
	NegativeSpace    targetRange `json:"negative_space"`
	Directional      targetRange `json:"directional"`
	Hierarchy        struct {
		DenseRatioTarget    float64 `json:"dense_ratio_target"`
		WidthSpreadTarget   float64 `json:"width_spread_target"`
		HeroMin             int     `json:"hero_min"`
		HeroMax             int     `json:"hero_max"`
		SupportingMin       int     `json:"supporting_min"`
		SupportingFraction  float64 `json:"supporting_fraction"`
		HeroWidthMultiplier float64 `json:"hero_width_multiplier"`
		MinimumScore        float64 `json:"minimum_score"`
	} `json:"hierarchy"`
	Anchor struct {
		SafeMargin      float64 `json:"safe_margin"`
		InfluenceRadius float64 `json:"influence_radius"`
		MinimumScore    float64 `json:"minimum_score"`
	} `json:"anchor"`
	Color struct {
		DominantRatioTarget float64 `json:"dominant_ratio_target"`
		FlareTarget         float64 `json:"flare_target"`
		FlareTolerance      float64 `json:"flare_tolerance"`
	} `json:"color"`
	Focal struct {
		BaselineRatio float64 `json:"baseline_ratio"`
		Scale         float64 `json:"scale"`
	} `json:"focal"`
	Weights struct {
		NegativeSpace    float64 `json:"negative_space"`
		Hierarchy        float64 `json:"hierarchy"`
		Direction        float64 `json:"direction"`
		ColorStructure   float64 `json:"color_structure"`
		AccentDiscipline float64 `json:"accent_discipline"`
		FocalStrength    float64 `json:"focal_strength"`
		HeroSupport      float64 `json:"hero_support"`
		AnchorStrength   float64 `json:"anchor_strength"`
		EdgeSafety       float64 `json:"edge_safety"`
		Balance          float64 `json:"balance"`
	} `json:"weights"`
	MinimumTotalScore float64 `json:"minimum_total_score"`
}

var walkArtV1 = mustCalibration()

func mustCalibration() CalibrationProfile {
	var profile CalibrationProfile
	if err := json.Unmarshal(calibrationJSON, &profile); err != nil {
		panic(fmt.Sprintf("invalid embedded calibration profile: %v", err))
	}
	if profile.Version == "" || profile.Scope != "global-corpus" || profile.NegativeSpace.Tolerance <= 0 || profile.Directional.Tolerance <= 0 || profile.Hierarchy.HeroMin < 1 {
		panic("invalid embedded calibration profile values")
	}
	return profile
}

func GlobalCalibrationProfile() CalibrationProfile { return walkArtV1 }
