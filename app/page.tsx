"use client";

import { useEffect, useRef, useState } from "react";

type Stage = "route" | "fingerprint" | "compose" | "canonical";

const transformationSteps: Array<{ id: Stage; eyebrow: string; title: string; detail: string; reads: string; changes: string }> = [
  { id: "route", eyebrow: "01 · ROUTE IN", title: "One real walk enters.", detail: "Upload one GPS track. Time of day, location, and music are optional atmosphere controls; the route is the only required input.", reads: "Ordered GPS points · elevation · timestamps", changes: "Nothing artistic yet. The engine repairs GPS noise and makes the walk reliable enough to interpret." },
  { id: "fingerprint", eyebrow: "02 · MOVEMENT FINGERPRINT", title: "The walk is measured.", detail: "The engine reads how the walk behaved, not just where its line went.", reads: "Distance · turns · curves · loops · pauses · pace · climbing", changes: "Important moments become forces: their position, strength, direction, and rhythm are preserved." },
  { id: "compose", eyebrow: "03 · GLOBAL COMPOSITION", title: "The route disappears.", detail: "WALK-ART-V1 converts the fingerprint into abstract motion. The walk still gives the piece a direction without appearing as a literal map line.", reads: "Route direction · movement events · optional time and music energy", changes: "A composition anchor, directional current, color territories, negative space, and hero/supporting trails emerge." },
  { id: "canonical", eyebrow: "04 · ONE CANONICAL WORK", title: "The engine makes the choice.", detail: "The engine creates 28 private drafts and scores them against one global quality standard—even on a first-ever upload.", reads: "28 drafts · WALK-ART-V1 global calibration profile", changes: "Direction, hierarchy, negative space, color, and focus decide the winner. You receive one finished work, never three finalists." },
];

function sizeCanvas(canvas: HTMLCanvasElement) {
  const rect = canvas.getBoundingClientRect();
  const ratio = Math.min(window.devicePixelRatio || 1, 2);
  const width = Math.max(280, Math.floor(rect.width));
  const height = Math.max(240, Math.floor(rect.height));
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  const context = canvas.getContext("2d");
  if (!context) return null;
  context.scale(ratio, ratio);
  return { context, width, height };
}

function routePoint(index: number, total: number, width: number, height: number) {
  const p = index / Math.max(total - 1, 1);
  return {
    x: width * (0.12 + p * 0.76) + Math.sin(p * 22) * width * 0.055 + Math.sin(p * 7) * width * 0.04,
    y: height * (0.58 - p * 0.18) + Math.sin(p * 13.5) * height * 0.17 + Math.cos(p * 31) * height * 0.025,
  };
}

function drawRoute(context: CanvasRenderingContext2D, width: number, height: number, color: string, lineWidth: number) {
  context.beginPath();
  for (let i = 0; i < 170; i++) {
    const point = routePoint(i, 170, width, height);
    if (i === 0) context.moveTo(point.x, point.y);
    else context.lineTo(point.x, point.y);
  }
  context.strokeStyle = color;
  context.lineWidth = lineWidth;
  context.lineCap = "round";
  context.lineJoin = "round";
  context.stroke();
}

function drawContours(context: CanvasRenderingContext2D, width: number, height: number, opacity = 0.18) {
  context.save();
  context.strokeStyle = `rgba(235, 227, 207, ${opacity})`;
  context.lineWidth = 1;
  for (let ring = 0; ring < 22; ring++) {
    context.beginPath();
    const cx = width * (0.68 + Math.sin(ring) * 0.012);
    const cy = height * (0.42 + Math.cos(ring * 0.8) * 0.018);
    for (let i = 0; i <= 150; i++) {
      const angle = (i / 150) * Math.PI * 2;
      const radius = 44 + ring * 19 + Math.sin(angle * 4 + ring * 0.7) * (7 + ring * 0.8);
      const x = cx + Math.cos(angle) * radius * 1.35;
      const y = cy + Math.sin(angle) * radius * 0.68;
      if (i === 0) context.moveTo(x, y);
      else context.lineTo(x, y);
    }
    context.closePath();
    context.stroke();
  }
  context.restore();
}

function TerrainCanvas() {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const draw = () => {
      if (!ref.current) return;
      const sized = sizeCanvas(ref.current);
      if (!sized) return;
      const { context, width, height } = sized;
      const gradient = context.createLinearGradient(0, 0, width, height);
      gradient.addColorStop(0, "#17372b");
      gradient.addColorStop(0.58, "#254c39");
      gradient.addColorStop(1, "#607a58");
      context.fillStyle = gradient;
      context.fillRect(0, 0, width, height);
      drawContours(context, width, height, 0.18);
      drawRoute(context, width, height, "rgba(244,239,228,.28)", 8);
      drawRoute(context, width, height, "#e46c43", 3.2);
      const start = routePoint(0, 170, width, height);
      const end = routePoint(169, 170, width, height);
      context.fillStyle = "#f4efe4";
      [start, end].forEach((point) => {
        context.beginPath();
        context.arc(point.x, point.y, 6, 0, Math.PI * 2);
        context.fill();
      });
    };
    draw();
    window.addEventListener("resize", draw);
    return () => window.removeEventListener("resize", draw);
  }, []);
  return <canvas ref={ref} className="terrain-canvas" aria-label="Topographic terrain with a traced walking route" />;
}

function TransformationCanvas({ stage }: { stage: Stage }) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const draw = () => {
      if (!ref.current) return;
      const sized = sizeCanvas(ref.current);
      if (!sized) return;
      const { context, width, height } = sized;
      context.fillStyle = "#e8e5db";
      context.fillRect(0, 0, width, height);
      if (stage === "route") {
        context.strokeStyle = "rgba(23,55,43,.12)";
        context.lineWidth = 1;
        for (let x = 0; x < width; x += 34) { context.beginPath(); context.moveTo(x, 0); context.lineTo(x, height); context.stroke(); }
        for (let y = 0; y < height; y += 34) { context.beginPath(); context.moveTo(0, y); context.lineTo(width, y); context.stroke(); }
        drawRoute(context, width, height, "#e46c43", 3.5);
        for (let i = 0; i < 170; i += 8) {
          const point = routePoint(i, 170, width, height);
          context.fillStyle = "#17372b";
          context.beginPath();
          context.arc(point.x, point.y, 2.2, 0, Math.PI * 2);
          context.fill();
        }
      }
      if (stage === "fingerprint") {
        drawContours(context, width, height, 0.1);
        drawRoute(context, width, height, "rgba(23,55,43,.14)", 10);
        drawRoute(context, width, height, "#17372b", 2.3);
        for (let i = 0; i < 170; i += 17) {
          const point = routePoint(i, 170, width, height);
          context.fillStyle = i % 34 === 0 ? "#e46c43" : "#d5b66f";
          context.beginPath();
          context.arc(point.x, point.y, 5, 0, Math.PI * 2);
          context.fill();
        }
      }
    };
    draw();
    window.addEventListener("resize", draw);
    return () => window.removeEventListener("resize", draw);
  }, [stage]);
  return <canvas ref={ref} className="transformation-canvas" aria-label={`${stage} stage of a walk becoming artwork`} />;
}

function EngineArtwork({ annotated = false }: { annotated?: boolean }) {
  return (
    <div className={`engine-artwork${annotated ? " is-annotated" : ""}`}>
      <img src="/generated/central-park.svg" alt={annotated ? "Current engine artwork with its composition roles identified" : "Actual canonical artwork generated from Central Park by the current Go engine"} />
      {annotated && <div className="composition-overlay" aria-hidden="true"><span className="role-anchor">ANCHOR</span><span className="role-hero">HERO CURRENT</span><span className="role-support">SUPPORTING TRAILS</span><span className="role-space">NEGATIVE SPACE</span></div>}
    </div>
  );
}

export default function Home() {
  const [stage, setStage] = useState<Stage>("canonical");
  const activeStep = transformationSteps.find((step) => step.id === stage) ?? transformationSteps[3];

  return (
    <main id="top">
      <header className="site-header">
        <a className="brand" href="#top" aria-label="Meander home"><span className="brand-line" />Meander</a>
        <nav aria-label="Main navigation"><a href="#how-it-works">How it works</a><a href="#fingerprint">The data</a><a href="#gallery">Prints</a><a href="/library">Library</a></nav>
        <a className="nav-cta" href="/create">Create from a route <span>↗</span></a>
      </header>

      <section className="hero">
        <TerrainCanvas />
        <div className="hero-shade" />
        <div className="hero-copy">
          <p className="eyebrow"><span /> Movement, interpreted</p>
          <h1>Every walk<br />leaves a pattern.</h1>
          <p>Turn the geometry of a real walk into a one-of-one generative print—shaped by every turn, pause, loop, and change in pace.</p>
          <div className="hero-actions"><a className="primary-action" href="/create">Create from a GPX <span>↗</span></a><a className="quiet-action" href="#how-it-works">See how it works</a></div>
        </div>
        <div className="hero-stats" aria-label="Sample walk summary">
          <div><span>Distance</span><strong>8.42 km</strong></div>
          <div><span>Elevation</span><strong>+124 m</strong></div>
          <div><span>Duration</span><strong>01:47:18</strong></div>
          <div><span>Location</span><strong>Brooklyn, NY</strong></div>
        </div>
        <div className="map-scale" aria-hidden="true"><i /><span>1 km</span></div>
        <p className="coordinates">40.6782° N · 73.9442° W</p>
      </section>

      <section className="intro">
        <p className="section-kicker">From GPS to generative art</p>
        <h2>Your route is more than a line.<br />It is a record of <span>how you moved.</span></h2>
        <p className="intro-copy">Meander turns a GPS track into a force field. Direction survives in the motion, ruptures, color territories, and a quiet broken trace of the original path.</p>
      </section>

      <section className="transformation" id="how-it-works">
        <div className="section-top"><div><p className="section-kicker">How it works</p><h2>One walk.<br />Four decisions.</h2></div><p>Follow one real route from raw GPS to the actual canonical engine output shown here. No personal history is required.</p></div>
        <div className="transformation-shell">
          <div className="canvas-panel">{stage === "canonical" ? <EngineArtwork /> : stage === "compose" ? <EngineArtwork annotated /> : <TransformationCanvas stage={stage} />}<div className="canvas-meta"><span>LIVE ENGINE EXAMPLE · CENTRAL PARK</span><span>{stage.toUpperCase()} VIEW</span></div></div>
          <div className="stage-panel">
            <div className="stage-buttons" role="tablist" aria-label="Transformation stages">
              {transformationSteps.map((step) => <button key={step.id} role="tab" aria-selected={stage === step.id} onClick={() => setStage(step.id)}><span>{step.eyebrow.slice(0, 2)}</span>{step.id}</button>)}
            </div>
            <div className="stage-copy" aria-live="polite"><p>{activeStep.eyebrow}</p><h3>{activeStep.title}</h3><span>{activeStep.detail}</span><div className="stage-facts"><div><b>What it reads</b><span>{activeStep.reads}</span></div><div><b>What changes in the art</b><span>{activeStep.changes}</span></div></div></div>
            <div className="stage-progress"><i style={{ width: `${(transformationSteps.findIndex((item) => item.id === stage) + 1) * 25}%` }} /></div>
          </div>
        </div>
      </section>

      <section className="fingerprint" id="fingerprint">
        <div className="fingerprint-copy"><p className="section-kicker">Movement fingerprint</p><h2>The walk<br />behind the work.</h2><p>Each print ships with a readable fingerprint showing exactly which characteristics shaped it.</p></div>
        <div className="fingerprint-card">
          <div className="fingerprint-head"><div><span>PUBLIC ROUTE / ENGINE 3.1</span><strong>Central Park</strong></div><span className="verified">Globally calibrated</span></div>
          <div className="fingerprint-route" aria-hidden="true"><i /><b>START</b><em>END</em></div>
          <div className="metric-strip"><div><span>Distance</span><strong>9.83<small> km</small></strong></div><div><span>Loop closure</span><strong>.991</strong></div><div><span>Hero trails</span><strong>02</strong></div><div><span>Draft</span><strong>26</strong></div></div>
          <div className="elevation"><div><span>COMPOSITION SCORE · WALK-ART-V1</span><span>0.903 / 1</span></div><i /></div>
        </div>
      </section>

      <section className="gallery" id="gallery">
        <div className="section-top"><div><p className="section-kicker">Generated by the Go engine</p><h2>Three routes.<br />Three distinct fields.</h2></div><p>These are real engine outputs from attributed OpenStreetMap geometry—not illustrative canvas mocks.</p></div>
        <div className="gallery-grid">
          <article><div className="print-frame"><img src="/generated/central-park.svg" alt="Field artwork generated from Central Park geometry" /></div><div><h3>Central Park</h3><p>NEW YORK · 9.83 KM · LOOP .991</p></div></article>
          <article><div className="print-frame"><img src="/generated/high-line.svg" alt="Field artwork generated from the High Line geometry" /></div><div><h3>The High Line</h3><p>NEW YORK · 5.43 KM · 6 TURNS</p></div></article>
          <article><div className="print-frame"><img src="/generated/golden-gate.svg" alt="Field artwork generated from Golden Gate Bridge geometry" /></div><div><h3>Golden Gate</h3><p>SAN FRANCISCO · 2.74 KM · DAWN</p></div></article>
        </div>
        <p className="data-credit">Validation geometry © OpenStreetMap contributors, ODbL. Route snapshots are for engine testing, not navigation.</p>
      </section>

      <section className="principles">
        <p className="section-kicker">Built on a simple promise</p>
        <div className="principle-grid"><article><span>01</span><h3>Real movement</h3><p>The source is a walk that happened—not a prompt describing one.</p></article><article><span>02</span><h3>Explainable math</h3><p>Every visible decision can be traced to a measurable route feature.</p></article><article><span>03</span><h3>Repeatable output</h3><p>The same walk and engine version always create the same composition.</p></article><article><span>04</span><h3>Private by default</h3><p>Raw location data is processed temporarily and never placed in the print.</p></article></div>
      </section>

      <section className="closing">
        <div className="closing-contours" aria-hidden="true" />
        <p className="section-kicker">The creation studio is live locally</p>
        <h2>Turn somewhere<br />you went into<br /><span>something you keep.</span></h2>
        <a href="/create">Open the creation studio <span>↗</span></a>
        <p className="next-view">READY NOW · GPX UPLOAD · ENGINE ANALYSIS · SVG + PNG OUTPUT</p>
      </section>

      <footer><a className="brand" href="#top"><span className="brand-line" />Meander</a><p>The geometry of going somewhere.</p><div><a href="#how-it-works">Process</a><a href="#fingerprint">Data</a><a href="#gallery">Prints</a></div><small>© 2026 MEANDER · LOCATION DATA STAYS PRIVATE</small></footer>
    </main>
  );
}
