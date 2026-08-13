"use client";

import { useEffect, useRef, useState } from "react";

type Point = { x: number; y: number };

const sampleRoutes: Record<string, Point[]> = {
  "central-park.osm": [{x:.43,y:.92},{x:.31,y:.78},{x:.33,y:.57},{x:.45,y:.36},{x:.39,y:.17},{x:.57,y:.09},{x:.72,y:.22},{x:.66,y:.43},{x:.78,y:.59},{x:.63,y:.79},{x:.43,y:.92}],
  "high-line.osm": [{x:.2,y:.89},{x:.28,y:.77},{x:.32,y:.63},{x:.44,y:.55},{x:.48,y:.4},{x:.59,y:.31},{x:.66,y:.15},{x:.8,y:.09}],
  "brooklyn-bridge.osm": [{x:.12,y:.78},{x:.28,y:.68},{x:.42,y:.55},{x:.57,y:.45},{x:.73,y:.31},{x:.88,y:.19}],
  "golden-gate.osm": [{x:.12,y:.77},{x:.26,y:.66},{x:.38,y:.58},{x:.5,y:.49},{x:.65,y:.39},{x:.78,y:.26},{x:.9,y:.16}],
};

function normalized(points: Array<{ x: number; y: number }>) {
  if (points.length < 2) return sampleRoutes["central-park.osm"];
  const minX = Math.min(...points.map((point) => point.x));
  const maxX = Math.max(...points.map((point) => point.x));
  const minY = Math.min(...points.map((point) => point.y));
  const maxY = Math.max(...points.map((point) => point.y));
  const width = Math.max(.000001, maxX - minX);
  const height = Math.max(.000001, maxY - minY);
  const scale = .72 / Math.max(width, height);
  const usedWidth = width * scale;
  const usedHeight = height * scale;
  return points.map((point) => ({
    x: .5 - usedWidth / 2 + (point.x - minX) * scale,
    y: .5 + usedHeight / 2 - (point.y - minY) * scale,
  }));
}

async function routeFromFile(file: File | null) {
  if (!file) return null;
  const documentNode = new DOMParser().parseFromString(await file.text(), "application/xml");
  const track = Array.from(documentNode.querySelectorAll("trkpt")).map((node) => ({
    x: Number(node.getAttribute("lon")), y: Number(node.getAttribute("lat")),
  })).filter((point) => Number.isFinite(point.x) && Number.isFinite(point.y));
  if (track.length > 1) return normalized(track);

  const nodes = new Map(Array.from(documentNode.querySelectorAll("node")).map((node) => [node.getAttribute("id") || "", {
    x: Number(node.getAttribute("lon")), y: Number(node.getAttribute("lat")),
  }]));
  const ways = Array.from(documentNode.querySelectorAll("way")).map((way) => Array.from(way.querySelectorAll("nd")).map((nd) => nodes.get(nd.getAttribute("ref") || "")).filter((point): point is Point => Boolean(point)));
  ways.sort((a, b) => b.length - a.length);
  return ways[0]?.length > 1 ? normalized(ways[0]) : null;
}

function strokePath(context: CanvasRenderingContext2D, points: Point[], width: number, height: number, fraction = 1, offset = 0, wave = 0) {
  const count = Math.max(2, Math.floor((points.length - 1) * fraction) + 1);
  context.beginPath();
  for (let index = 0; index < count; index++) {
    const point = points[index];
    const before = points[Math.max(0, index - 1)];
    const after = points[Math.min(points.length - 1, index + 1)];
    const dx = after.x - before.x;
    const dy = after.y - before.y;
    const magnitude = Math.hypot(dx, dy) || 1;
    const pulse = offset + Math.sin(index * .23 + wave) * Math.abs(offset) * .16;
    const x = point.x * width + (-dy / magnitude) * pulse;
    const y = point.y * height + (dx / magnitude) * pulse;
    if (index === 0) context.moveTo(x, y); else context.lineTo(x, y);
  }
  context.stroke();
}

export function GenerationTheatre({ stage, routeFile, sample, title, timeOfDay, energy, revealing }: {
  stage: number; routeFile: File | null; sample: string; title: string; timeOfDay: string; energy: number; revealing: boolean;
}) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const shell = useRef<HTMLDivElement>(null);
  const [points, setPoints] = useState<Point[]>(sampleRoutes[sample] || sampleRoutes["central-park.osm"]);
  const stageStart = useRef(0);

  useEffect(() => { stageStart.current = performance.now(); }, [stage]);
  useEffect(() => {
    let alive = true;
    routeFromFile(routeFile).then((next) => { if (alive && next) setPoints(next); });
    return () => { alive = false; };
  }, [routeFile]);
  useEffect(() => {
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    shell.current?.focus();
    return () => { document.body.style.overflow = previous; };
  }, []);

  useEffect(() => {
    const element = canvas.current;
    if (!element) return;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let frame = 0;
    const draw = (now: number) => {
      const rect = element.getBoundingClientRect();
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.round(rect.width * ratio));
      const height = Math.max(1, Math.round(rect.height * ratio));
      if (element.width !== width || element.height !== height) { element.width = width; element.height = height; }
      const context = element.getContext("2d");
      if (!context) return;
      const elapsed = reduced ? 2000 : now - stageStart.current;
      context.clearRect(0, 0, width, height);
      context.fillStyle = "#eeeade";
      context.fillRect(0, 0, width, height);

      const routeProgress = stage === 0 ? Math.min(1, elapsed / 850) : 1;
      if (stage >= 2) {
        const fieldProgress = stage === 2 ? Math.min(1, elapsed / 1200) : 1;
        const colors = ["#6c8462", "#df754b", "#9db8ae", "#c8aa69"];
        const lineCount = Math.floor((20 + energy * 18) * fieldProgress);
        for (let index = lineCount; index >= 0; index--) {
          const side = index % 2 ? 1 : -1;
          const distance = side * (8 + Math.ceil(index / 2) * (3.5 + energy * 2.5)) * ratio;
          context.strokeStyle = colors[index % colors.length];
          context.globalAlpha = .12 + (1 - index / Math.max(1, lineCount)) * .2;
          context.lineWidth = (index % 7 === 0 ? 2.1 : .9) * ratio;
          context.setLineDash(index % 5 === 0 ? [8 * ratio, 5 * ratio] : []);
          strokePath(context, points, width, height, 1, distance, now / 900 + index);
        }
      }

      context.setLineDash(stage >= 2 ? [12 * ratio, 7 * ratio, 2 * ratio, 6 * ratio] : []);
      context.strokeStyle = stage >= 2 ? "#547987" : "#df754b";
      context.globalAlpha = stage >= 2 ? .72 : .96;
      context.lineWidth = (stage >= 2 ? 2 : 3) * ratio;
      strokePath(context, points, width, height, routeProgress);

      if (stage === 1) {
        context.setLineDash([]);
        points.filter((_, index) => index % Math.max(1, Math.floor(points.length / 8)) === 0).forEach((point, index) => {
          const pulse = (5 + Math.sin(now / 220 + index) * 2) * ratio;
          context.beginPath();
          context.arc(point.x * width, point.y * height, pulse, 0, Math.PI * 2);
          context.strokeStyle = index % 2 ? "#df754b" : "#6c8462";
          context.globalAlpha = .65;
          context.lineWidth = ratio;
          context.stroke();
        });
      }
      context.globalAlpha = 1;
      frame = window.requestAnimationFrame(draw);
    };
    frame = window.requestAnimationFrame(draw);
    return () => window.cancelAnimationFrame(frame);
  }, [energy, points, stage]);

  const acts = [
    ["01 · Recovering the walk", "Reading its shape", "Preserving the turns, loops, and direction that make this route yours."],
    ["02 · Finding movement", "Listening to the geometry", "Curvature, rhythm, and meaningful changes become forces in the field."],
    ["03 · Building the field", "Letting the walk expand", "Color and surrounding currents grow from the original direction of travel."],
    ["04 · Choosing the work", "One composition remains", "Checking hierarchy, negative space, and route memory before the reveal."],
  ][stage] || ["04 · Choosing the work", "One composition remains", "Preparing your canonical work."];

  return <div ref={shell} className={`generation-theatre ${revealing ? "is-revealing" : ""}`} role="dialog" aria-modal="true" aria-labelledby="generation-title" tabIndex={-1}>
    <div className="theatre-brand"><span className="brand-line" /> Meander <b>TRANSFORMATION IN PROGRESS</b></div>
    <div className="theatre-stage">
      <div className="theatre-canvas"><canvas ref={canvas} aria-label="Your route transforming into a generative field" /><span>LIVE INTERPRETATION · {timeOfDay.toUpperCase()}</span></div>
      <div className="theatre-copy" aria-live="polite">
        <p>{acts[0]}</p><h2 id="generation-title">{acts[1]}</h2><span>{acts[2]}</span>
        <div className="theatre-route-name">{title || "Untitled walk"}</div>
        <ol>{[0,1,2,3].map((item) => <li className={item < stage ? "complete" : item === stage ? "active" : ""} key={item}><i /> <span>{["Route", "Movement", "Field", "Canonical"][item]}</span></li>)}</ol>
      </div>
    </div>
    <p className="theatre-note">One route enters. One work leaves. Your source file is not stored.</p>
  </div>;
}
