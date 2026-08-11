package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"walkart/internal/engine"
)

func main() {
	in := flag.String("input", "", "GPX or OSM XML route file")
	out := flag.String("out", "output", "output directory")
	title := flag.String("title", "", "location or route label")
	tod := flag.String("time", "", "optional time of day")
	tempo := flag.Float64("tempo", 0, "optional music tempo")
	energy := flag.Float64("energy", 0, "optional music energy from 0 to 1")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "-input is required")
		os.Exit(2)
	}
	f, err := os.Open(*in)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	pts, err := engine.Parse(f, filepath.Base(*in))
	if err != nil {
		fatal(err)
	}
	res, err := engine.Generate(pts, engine.Context{LocationLabel: *title, TimeOfDay: *tod, MusicTempo: *tempo, MusicEnergy: *energy})
	if err != nil {
		fatal(err)
	}
	dir := filepath.Join(*out, res.ID)
	if err := engine.WriteBundle(dir, res); err != nil {
		fatal(err)
	}
	fmt.Printf("%s\t%s\t%.2f km\t%s\n", res.ID, dir, res.Features.DistanceKM, res.Recipe.Seed)
}
func fatal(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
