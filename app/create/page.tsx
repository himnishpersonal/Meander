"use client";

import { FormEvent, useState } from "react";
import Link from "next/link";
import { API } from "@/app/api";

type Generation = {
  id: string;
  title: string;
  subtitle: string;
  palette: string;
  artwork_url: string;
  preview_url: string;
  features: Record<string, number>;
  family: string;
  events: Array<{ kind: string; strength: number }>;
  recipe: { candidate: string; seed: string; score: QualityScore };
};
type QualityScore = { Total: number; NegativeSpace: number; Hierarchy: number; RouteLegibility: number; ColorStructure: number; AccentDiscipline: number; FocalStrength: number; HeroSupport: number; AnchorStrength: number };

const samples = [
  { file: "central-park.osm", name: "Central Park", place: "New York" },
  { file: "high-line.osm", name: "The High Line", place: "New York" },
  { file: "brooklyn-bridge.osm", name: "Brooklyn Bridge", place: "New York" },
  { file: "golden-gate.osm", name: "Golden Gate Bridge", place: "San Francisco" },
];

export default function CreatePage() {
  const [route, setRoute] = useState<File | null>(null);
  const [sample, setSample] = useState(samples[0].file);
  const [location, setLocation] = useState(samples[0].name);
  const [time, setTime] = useState("morning");
  const [tempo, setTempo] = useState("108");
  const [energy, setEnergy] = useState("0.54");
  const [result, setResult] = useState<Generation | null>(null);
  const [status, setStatus] = useState<"idle" | "working" | "error">("idle");
  const [error, setError] = useState("");
  const [authRequired, setAuthRequired] = useState(false);
  const artwork = result ? `${API}${result.artwork_url}` : "/generated/central-park.svg";

  async function submit(event: FormEvent) {
    event.preventDefault(); setStatus("working"); setError(""); setAuthRequired(false);
    const data = new FormData();
    if (route) data.append("route", route); else data.append("sample", sample);
    data.append("location_label", location || route?.name.replace(/\.[^.]+$/, "") || "Untitled walk");
    data.append("time_of_day", time); data.append("music_tempo", tempo); data.append("music_energy", energy);
    try {
      const response = await fetch(`${API}/api/v1/generate`, { method: "POST", credentials: "include", body: data });
      const body = await response.json().catch(() => ({}));
      if (response.status === 401) {
        setAuthRequired(true);
        setStatus("idle");
        return;
      }
      if (!response.ok) throw new Error(body.error || "The engine could not read that route.");
      setResult({ ...body, events: Array.isArray(body.events) ? body.events : [], features: body.features || {} }); setStatus("idle");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Meander could not reach the generation service."); setStatus("error");
    }
  }

  function chooseSample(file: string) {
    const item = samples.find((entry) => entry.file === file)!;
    setSample(file); setLocation(item.name); setRoute(null); setResult(null);
  }

  return <main className="studio-page">
    <header className="site-header studio-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><span>CREATION STUDIO · LIVE GO ENGINE</span><Link className="nav-cta" href="/library">My library <span>↗</span></Link></header>
    <section className="studio-intro"><p className="section-kicker">Route in. Turbulence out.</p><h1>Your movement<br />makes the system.</h1><p>Upload a GPX track or test public geometry. The route drives the entire field, while a restrained broken path keeps the walk discoverable inside the abstraction.</p></section>
    <section className="studio-workspace">
      <form className="studio-controls" onSubmit={submit}>
        <div className="control-block"><span className="control-number">01</span><label>Choose the route <small>Required</small></label><div className="sample-grid">{samples.map((item) => <button type="button" className={!route && sample === item.file ? "selected" : ""} onClick={() => chooseSample(item.file)} key={item.file}><strong>{item.name}</strong><span>{item.place} · OSM</span></button>)}</div><label className="upload-field"><input type="file" accept=".gpx,.osm,application/gpx+xml,application/xml,text/xml" onChange={(event) => { const next=event.target.files?.[0] || null; setRoute(next); if(next)setLocation(next.name.replace(/\.[^.]+$/, "")); }} /><span>{route ? route.name : "Or upload GPX / OSM XML"}</span><b>Browse ↗</b></label></div>
        <div className="control-block"><span className="control-number">02</span><label htmlFor="location">Name the walk <small>Optional</small></label><input id="location" value={location} onChange={(event) => setLocation(event.target.value)} placeholder="Sunday loop" /></div>
        <div className="control-block context-grid"><div><span className="control-number">03</span><label htmlFor="time">Time of day <small>Optional</small></label><select id="time" value={time} onChange={(event) => setTime(event.target.value)}><option>morning</option><option>afternoon</option><option>evening</option><option>night</option><option>dawn</option></select></div><div><span className="control-number">04</span><label htmlFor="tempo">Music tempo <small>BPM</small></label><input id="tempo" type="number" min="40" max="220" value={tempo} onChange={(event) => setTempo(event.target.value)} /></div></div>
        <div className="control-block"><span className="control-number">05</span><label htmlFor="energy">Music energy <small>{Math.round(Number(energy) * 100)}%</small></label><input id="energy" type="range" min="0" max="1" step="0.01" value={energy} onChange={(event) => setEnergy(event.target.value)} /></div>
        <button className="generate-button" disabled={status === "working"}>{status === "working" ? "Searching 28 private drafts…" : "Generate the artwork"}<span>→</span></button>
        {authRequired && <div className="auth-required"><strong>Sign in to generate your artwork.</strong><p>Your route is ready. Sign in with Google, then return here to create and save the result privately.</p><Link href="/sign-in?returnTo=/create">Continue with Google <span>↗</span></Link></div>}
        {error && <p className="engine-error"><strong>Generation paused.</strong> {error}<br />Please try again in a moment.</p>}
        <p className="privacy-note">Your uploaded file is processed in memory. The local engine stores only the resulting art and fingerprint—not the raw route.</p>
      </form>
      <div className="studio-output">
        <div className="output-frame"><img src={artwork} alt={result ? `Generated route field for ${result.title}` : "Central Park sample generated by the Meander engine"} /><span>{status === "working" ? "RUNNING CANONICAL QUALITY SEARCH" : result ? "ONE WALK · ONE OUTPUT" : "LIVE ENGINE SAMPLE"}</span></div>
        <div className="output-copy"><div><p>{result ? "Canonical artwork" : "Public route sample"}</p><h2>{result?.title || "Central Park"}</h2><span>{result ? `${result.subtitle} · directional abstraction` : "9.83 km · deterministic turbulence · morning"}</span></div>{result && <div className="output-actions"><a href={`${API}${result.artwork_url}`} download>SVG ↗</a><a href={`${API}${result.preview_url}`} download>PNG ↗</a></div>}</div>
        {result && <><div className="result-metrics"><div><span>Direction</span><strong>{result.recipe.score.RouteLegibility.toFixed(3)}</strong></div><div><span>Color structure</span><strong>{result.recipe.score.ColorStructure.toFixed(3)}</strong></div><div><span>Focal strength</span><strong>{result.recipe.score.FocalStrength.toFixed(3)}</strong></div><div><span>Quality</span><strong>{result.recipe.score.Total.toFixed(3)}</strong></div></div><div className="movement-readout"><span>ROUTE FINGERPRINT</span><p>{result.events?.length ?? 0} local movement events detected · {result.features.DwellPoints || 0} pauses · {result.features.PaceChanges || 0} pace changes · {Number(result.features.ElevationGainM || 0).toFixed(0)} m climbing</p></div></>}
      </div>
    </section>
    <footer className="studio-footer"><p>One walk · one canonical artwork · globally calibrated from upload one</p><small>ENGINE field-3.2.0 · WALK-ART-V1 · CLOUD RUN</small></footer>
  </main>;
}
