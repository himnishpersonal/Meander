"use client";

import { useEffect, useRef } from "react";
import { BrandMark } from "@/app/brand-mark";

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

export default function Home() {
  return (
    <main id="top">
      <header className="site-header">
        <a className="brand" href="#top" aria-label="Meander home"><BrandMark />Meander</a>
        <nav aria-label="Main navigation"><a href="/how-it-works">How Meander works</a><a href="#fingerprint">The data</a><a href="/gallery">Gallery</a><a href="/library">Library</a></nav>
        <a className="nav-cta" href="/create">Create from a route <span>↗</span></a>
      </header>

      <section className="hero">
        <TerrainCanvas />
        <div className="hero-shade" />
        <div className="hero-copy">
          <p className="eyebrow"><span /> Movement, interpreted</p>
          <h1>Every walk<br />leaves a pattern.</h1>
          <p>Turn the geometry of a real walk into a one-of-one generative print—shaped by every turn, pause, loop, and change in pace.</p>
          <div className="hero-actions"><a className="primary-action" href="/create">Create from a route <span>↗</span></a><a className="quiet-action" href="/how-it-works">See how Meander works</a></div>
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

      <section className="process-preview" id="how-it-works">
        <div className="section-top"><div><p className="section-kicker">How Meander works</p><h2>A route becomes<br />something you keep.</h2></div><p>Your walk stays recognizable while its movement becomes direction, rhythm, color, and flowing structure.</p></div>
        <div className="process-preview-rail" aria-label="Route to artwork process">
          <article><span>01</span><h3>Bring your walk</h3><p>Upload activity data, trace a screenshot, or try a sample.</p></article>
          <article><span>02</span><h3>Read its movement</h3><p>Turns, loops, pauses, rhythm, and direction form a fingerprint.</p></article>
          <article><span>03</span><h3>Build the field</h3><p>The route becomes the quiet spine of a wider composition.</p></article>
          <article><span>04</span><h3>Choose one work</h3><p>The engine scores private drafts and returns one canonical piece.</p></article>
          <article><span>05</span><h3>Keep it your way</h3><p>Save privately, share by link, publish, or download.</p></article>
        </div>
        <a className="process-preview-link" href="/how-it-works">Walk through the complete process <span>↗</span></a>
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

      <footer><a className="brand" href="#top"><BrandMark />Meander</a><p>The geometry of going somewhere.</p><div><a href="/how-it-works">Process</a><a href="#fingerprint">Data</a><a href="/gallery">Gallery</a></div><small>© 2026 MEANDER · RAW ROUTES STAY PRIVATE</small></footer>
    </main>
  );
}
