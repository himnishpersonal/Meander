"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { AccountNav } from "@/app/account-nav";
import { API, MEANDER_REQUEST_HEADERS } from "@/app/api";
import { BrandMark } from "@/app/brand-mark";
import { ScreenshotTracer } from "@/app/create/screenshot-tracer";
import { GenerationTheatre } from "@/app/create/generation-theatre";

type QualityScore = { Total: number; NegativeSpace: number; Hierarchy: number; RouteLegibility: number; ColorStructure: number; AccentDiscipline: number; FocalStrength: number; HeroSupport: number; AnchorStrength: number };
type Generation = { id: string; share_id: string; share_url: string; title: string; subtitle: string; palette: string; artwork_url: string; preview_url: string; features: Record<string, number>; family: string; events: Array<{ kind: string; strength: number }>; recipe: { candidate: string; seed: string; score: QualityScore } };
type InputKind = "file" | "screenshot" | "sample";

const samples = [
  { file: "central-park.osm", name: "Central Park", place: "New York" },
  { file: "high-line.osm", name: "The High Line", place: "New York" },
  { file: "brooklyn-bridge.osm", name: "Brooklyn Bridge", place: "New York" },
  { file: "golden-gate.osm", name: "Golden Gate", place: "San Francisco" },
];
const progressSteps = ["Recovering your walk", "Finding its movement", "Building the field", "Choosing the work"];
const minimumRevealTime = 5200;

export default function CreatePage() {
  const [route, setRoute] = useState<File | null>(null);
  const [screenshot, setScreenshot] = useState<File | null>(null);
  const [inputKind, setInputKind] = useState<InputKind>("file");
  const [sample, setSample] = useState(samples[0].file);
  const [location, setLocation] = useState("");
  const [time, setTime] = useState("morning");
  const [tempo, setTempo] = useState("108");
  const [energy, setEnergy] = useState("0.54");
  const [result, setResult] = useState<Generation | null>(null);
  const [status, setStatus] = useState<"idle" | "working" | "revealing" | "error">("idle");
  const [error, setError] = useState("");
  const [signedIn, setSignedIn] = useState<boolean | null>(null);
  const [progress, setProgress] = useState(0);
  const [shareMessage, setShareMessage] = useState("");

  useEffect(() => { fetch(`${API}/api/v1/me`, { credentials: "include", cache: "no-store" }).then((response) => setSignedIn(response.ok)).catch(() => setSignedIn(false)); }, []);
  useEffect(() => {
    if (status !== "working") return;
    const timers = [900, 2200, 3700].map((delay, index) => window.setTimeout(() => setProgress(index + 1), delay));
    return () => timers.forEach(window.clearTimeout);
  }, [status]);

  function chooseRoute(next: File) {
    setRoute(next); setScreenshot(null); setInputKind(next.name === "screenshot-route.gpx" ? "screenshot" : "file"); setLocation((current) => current || next.name.replace(/\.[^.]+$/, "")); setResult(null); setError("");
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!signedIn) return;
    if (inputKind !== "sample" && !route) { setError("Choose a GPX/OSM file or trace a route screenshot first."); return; }
    const started = performance.now();
    setProgress(0); setStatus("working"); setError(""); setShareMessage("");
    const data = new FormData();
    if (inputKind === "sample") data.append("sample", sample); else if (route) data.append("route", route);
    if (inputKind === "screenshot") data.append("geometry_only", "true");
    data.append("location_label", location || (inputKind === "sample" ? samples.find((item) => item.file === sample)?.name || "Untitled walk" : "Untitled walk"));
    data.append("time_of_day", time); data.append("music_tempo", tempo); data.append("music_energy", energy);
    try {
      const response = await fetch(`${API}/api/v1/generate`, { method: "POST", credentials: "include", headers: MEANDER_REQUEST_HEADERS, body: data });
      const body = await response.json().catch(() => ({}));
      if (response.status === 401) { setSignedIn(false); setStatus("idle"); return; }
      if (!response.ok) throw new Error(body.error || "The engine could not read that route.");
      const wait = Math.max(0, minimumRevealTime - (performance.now() - started));
      if (wait) await new Promise((resolve) => window.setTimeout(resolve, wait));
      setProgress(3);
      setResult({ ...body, events: Array.isArray(body.events) ? body.events : [], features: body.features || {} });
      setStatus("revealing");
      await new Promise((resolve) => window.setTimeout(resolve, 700));
      setStatus("idle");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Meander could not reach the generation service."); setStatus("error"); }
  }

  async function shareArtwork() {
    if (!result) return;
    const response = await fetch(`${API}/api/v1/artworks/${result.id}`, { method: "PATCH", credentials: "include", headers: { "Content-Type": "application/json", ...MEANDER_REQUEST_HEADERS }, body: JSON.stringify({ visibility: "unlisted" }) });
    if (!response.ok) { setShareMessage("Sharing could not be enabled."); return; }
    const updated = await response.json().catch(() => ({})) as Partial<Generation>;
    const next = { ...result, ...updated };
    setResult(next);
    const url = `${window.location.origin}${next.share_url}`;
    try { await navigator.clipboard.writeText(url); setShareMessage("Private share link copied"); } catch { setShareMessage(url); }
  }

  const sourceReady = inputKind === "sample" || Boolean(route);
  const interpretation = result ? `${result.events.length || "A few"} meaningful movement moments shaped a ${time} composition with ${Math.round(Number(energy) * 100)}% atmospheric energy.` : "The route remains visible as direction—not as a literal map.";

  const transforming = status === "working" || status === "revealing";

  return <main className={`studio-page ${result ? "has-result" : ""}`}>
    {transforming && <GenerationTheatre stage={progress} routeFile={inputKind === "sample" ? null : route} sample={sample} title={location || samples.find((item) => item.file === sample)?.name || "Untitled walk"} timeOfDay={time} energy={Number(energy)} revealing={status === "revealing"} />}
    <header className="site-header studio-header"><Link className="brand" href="/"><BrandMark />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link href="/gallery">Gallery</Link><Link href="/library">Library</Link></nav><AccountNav /></header>
    {!result && <section className="studio-intro"><p className="section-kicker">One walk. One work.</p><h1>Bring a route.<br />Leave with art.</h1><p>Upload recorded activity data or a route screenshot—Meander will find the highlighted path for you. Your path supplies the direction; optional atmosphere controls shape how it feels.</p></section>}
    <section className="studio-workspace">
      <form className="studio-controls" onSubmit={submit}>
        {!signedIn && signedIn !== null && <div className="sign-in-first"><span>01 · BEFORE YOU BEGIN</span><h2>Sign in before choosing a private route.</h2><p>This keeps your file from being lost during the Google redirect. Meander never stores the raw route.</p><Link href="/sign-in?returnTo=/create">Continue with Google <b>↗</b></Link></div>}
        <fieldset disabled={!signedIn || status === "working"}>
          <div className="control-block route-source"><span className="control-number">01</span><label>Choose your route <small>Required</small></label>
            <div className="source-tabs"><button type="button" aria-pressed={inputKind === "file"} onClick={() => { setInputKind("file"); setRoute(null); setScreenshot(null); }}>Activity file</button><button type="button" aria-pressed={inputKind === "screenshot"} onClick={() => { setInputKind("screenshot"); setRoute(null); setScreenshot(null); }}>Screenshot</button><button type="button" aria-pressed={inputKind === "sample"} onClick={() => { setInputKind("sample"); setRoute(null); setScreenshot(null); if (!location) setLocation(samples[0].name); }}>Try a sample</button></div>
            {inputKind === "file" && <label className="upload-drop"><input type="file" accept=".gpx,.osm,application/gpx+xml,application/xml,text/xml" onChange={(event) => { const next = event.target.files?.[0]; if (next) chooseRoute(next); }} /><strong>{route ? route.name : "Choose a GPX or OSM file"}</strong><span>{route ? "Ready to interpret" : "click to browse · up to 16 MB"}</span><b>{route ? "✓" : "Browse ↗"}</b></label>}
            {inputKind === "screenshot" && !screenshot && !route && <label className="upload-drop screenshot-drop"><input type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => { const next = event.target.files?.[0]; if (next) setScreenshot(next); }} /><strong>Upload a route screenshot</strong><span>PNG, JPEG or WebP · Meander finds the highlighted route locally</span><b>Choose image ↗</b></label>}
            {inputKind === "screenshot" && route && <div className="screenshot-ready"><span>✓</span><div><strong>Screenshot route traced</strong><small>Geometry-only path ready for the engine</small></div><button type="button" onClick={() => { setRoute(null); setScreenshot(null); }}>Trace again</button></div>}
            {inputKind === "sample" && <div className="sample-grid">{samples.map((item) => <button type="button" className={sample === item.file ? "selected" : ""} onClick={() => { setSample(item.file); setLocation(item.name); }} key={item.file}><strong>{item.name}</strong><span>{item.place} · public geometry</span></button>)}</div>}
          </div>
          {screenshot && <ScreenshotTracer image={screenshot} onCancel={() => { setScreenshot(null); setRoute(null); }} onConfirm={(file) => { chooseRoute(file); setScreenshot(null); setLocation((current) => current || "Screenshot walk"); }} />}
          <div className="control-block"><span className="control-number">02</span><label htmlFor="location">Name the walk <small>Optional</small></label><input id="location" value={location} onChange={(event) => setLocation(event.target.value)} placeholder="Sunday loop" /></div>
          <details className="atmosphere-controls"><summary><span>Shape the interpretation</span><small>Optional · time and music</small></summary><div className="context-grid"><div><label htmlFor="time">Time of day</label><select id="time" value={time} onChange={(event) => setTime(event.target.value)}><option>morning</option><option>afternoon</option><option>evening</option><option>night</option><option>dawn</option></select></div><div><label htmlFor="tempo">Music tempo</label><input id="tempo" type="number" min="40" max="220" value={tempo} onChange={(event) => setTempo(event.target.value)} /></div></div><label htmlFor="energy">Music energy <small>{Math.round(Number(energy) * 100)}%</small></label><input id="energy" type="range" min="0" max="1" step="0.01" value={energy} onChange={(event) => setEnergy(event.target.value)} /></details>
          <button className="generate-button" disabled={!signedIn || !sourceReady || status === "working"}>{status === "working" ? progressSteps[progress] : "Create my artwork"}<span>→</span></button>
        </fieldset>
        {error && <p className="engine-error"><strong>Generation paused.</strong> {error} Your choices are still here—adjust the route or try again.</p>}
        <p className="privacy-note">Raw route files and screenshots are processed for this creation only. Meander stores the finished work and its abstract movement fingerprint—not the source upload.</p>
      </form>
      <div className="studio-output">
        {result && <div className="result-nav"><Link href="/library">← Back to library</Link><span>Saved privately</span></div>}
        <div className="output-frame"><img src={result ? `${API}${result.preview_url}` : "/generated/central-park.svg"} alt={result ? `Generated artwork for ${result.title}` : "Sample artwork generated by Meander"} /><span>{status === "working" ? progressSteps[progress].toUpperCase() : result ? "YOUR CANONICAL WORK" : "LIVE ENGINE SAMPLE"}</span></div>
        <div className="output-copy"><div><p>{result ? "A walk, interpreted" : "What Meander makes"}</p><h2>{result?.title || "Direction without a map"}</h2><span>{result ? result.subtitle : "A single globally calibrated work from your route."}</span></div></div>
        {result && <div className="result-story"><p>{interpretation}</p><div className="result-access-note"><span>Saved privately</span><p>Only you can open this work until you create a share link or publish it from your library.</p></div><div className="output-actions"><a className="primary-download" href={`${API}${result.preview_url}`} download>Download PNG ↓</a><a href={`${API}${result.artwork_url}`} download>SVG ↓</a><button type="button" onClick={shareArtwork}>Copy private share link ↗</button><Link href="/library">Manage in library ↗</Link></div>{shareMessage && <span className="share-feedback">{shareMessage}</span>}<details><summary>View generation details</summary><div className="result-metrics"><div><span>Direction</span><strong>{result.recipe.score.RouteLegibility.toFixed(3)}</strong></div><div><span>Color structure</span><strong>{result.recipe.score.ColorStructure.toFixed(3)}</strong></div><div><span>Focus</span><strong>{result.recipe.score.FocalStrength.toFixed(3)}</strong></div><div><span>Composition</span><strong>{result.recipe.score.Total.toFixed(3)}</strong></div></div><div className="movement-readout"><span>ROUTE FINGERPRINT</span><p>{result.events.length} movement events · {result.features.DwellPoints || 0} pauses · {result.features.PaceChanges || 0} pace changes · {Number(result.features.ElevationGainM || 0).toFixed(0)} m climbing</p></div></details><button type="button" className="create-another" onClick={() => { setResult(null); setShareMessage(""); window.scrollTo({ top: 0, behavior: "smooth" }); }}>Create another walk</button></div>}
      </div>
    </section>
    <footer className="studio-footer"><p>One walk · one canonical artwork · globally calibrated from upload one</p><small>ENGINE field-3.2.1 · WALK-ART-V1</small></footer>
  </main>;
}
