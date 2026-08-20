"use client";

import { PointerEvent, useEffect, useRef, useState } from "react";

type TracePoint = { x: number; y: number };
type SuggestedTrace = { points: TracePoint[]; confidence: number };

function routeFile(points: TracePoint[]) {
  const lat0 = 40.75;
  const lon0 = -73.98;
  const body = points.map((point, index) => `<trkpt lat="${(lat0 + (1 - point.y) * .02).toFixed(7)}" lon="${(lon0 + point.x * .02).toFixed(7)}"><time>${new Date(Date.UTC(2026, 0, 1, 12, 0, index * 8)).toISOString()}</time></trkpt>`).join("");
  return new File([`<?xml version="1.0"?><gpx version="1.1" creator="Meander screenshot tracer"><trk><name>Screenshot route</name><trkseg>${body}</trkseg></trk></gpx>`], "screenshot-route.gpx", { type: "application/gpx+xml" });
}

// A deliberately conservative, browser-only route suggestion. It looks for a
// long connected run of high-saturation pixels—the common styling for an
// activity route—rather than uploading map imagery to a recognition service.
function routeSignal(red: number, green: number, blue: number) {
  const high = Math.max(red, green, blue), low = Math.min(red, green, blue);
  const saturation = high === 0 ? 0 : (high - low) / high;
  if (saturation < .35 || high < 105) return 0;
  const hue = ((red === high ? (green - blue) / (high - low || 1) : green === high ? 2 + (blue - red) / (high - low || 1) : 4 + (red - green) / (high - low || 1)) * 60 + 360) % 360;
  // Orange/red/purple/blue are all common route colors. Green scores lower so
  // parks and vegetation are less likely to be mistaken for a walk.
  const hueWeight = (hue <= 55 || hue >= 325) ? 1 : hue >= 190 && hue <= 285 ? .88 : hue >= 70 && hue <= 165 ? .48 : .58;
  return saturation * (high / 255) * hueWeight;
}

function suggestedRoute(source: HTMLImageElement): SuggestedTrace | null {
  const longest = 560;
  const scale = Math.min(1, longest / Math.max(source.naturalWidth, source.naturalHeight));
  const width = Math.max(80, Math.round(source.naturalWidth * scale));
  const height = Math.max(80, Math.round(source.naturalHeight * scale));
  const working = document.createElement("canvas");
  working.width = width; working.height = height;
  const context = working.getContext("2d", { willReadFrequently: true });
  if (!context) return null;
  context.drawImage(source, 0, 0, width, height);
  const pixels = context.getImageData(0, 0, width, height).data;
  const mask = new Uint8Array(width * height);
  const scores = new Float32Array(width * height);
  for (let index = 0; index < mask.length; index++) {
    const signal = routeSignal(pixels[index * 4], pixels[index * 4 + 1], pixels[index * 4 + 2]);
    scores[index] = signal;
    if (signal > .34) mask[index] = 1;
  }
  const visited = new Uint8Array(mask.length);
  let winner: number[] = [], winnerScore = 0;
  const neighbors = [-1, 0, 1];
  for (let start = 0; start < mask.length; start++) {
    if (!mask[start] || visited[start]) continue;
    const queue = [start]; visited[start] = 1;
    const component: number[] = []; let score = 0;
    for (let cursor = 0; cursor < queue.length; cursor++) {
      const current = queue[cursor]; component.push(current); score += scores[current];
      const x = current % width, y = Math.floor(current / width);
      for (const dy of neighbors) for (const dx of neighbors) {
        if (dx === 0 && dy === 0) continue;
        const nx = x + dx, ny = y + dy;
        if (nx < 0 || ny < 0 || nx >= width || ny >= height) continue;
        const next = ny * width + nx;
        if (mask[next] && !visited[next]) { visited[next] = 1; queue.push(next); }
      }
    }
    // Long highlighted paths are preferred over small colourful labels/icons.
    const weighted = score * Math.min(2.2, component.length / 140);
    if (component.length >= 36 && weighted > winnerScore) { winner = component; winnerScore = weighted; }
  }
  if (winner.length < 36) return null;

  const cells = new Map<string, { x: number; y: number; count: number }>();
  const cellSize = Math.max(2, Math.round(Math.min(width, height) / 170));
  for (const pixel of winner) {
    const x = pixel % width, y = Math.floor(pixel / width), key = `${Math.floor(x / cellSize)}:${Math.floor(y / cellSize)}`;
    const cell = cells.get(key) || { x: 0, y: 0, count: 0 };
    cell.x += x; cell.y += y; cell.count++; cells.set(key, cell);
  }
  const candidates = [...cells.values()].map((cell) => ({ x: cell.x / cell.count, y: cell.y / cell.count }));
  if (candidates.length < 6) return null;
  const center = candidates.reduce((sum, point) => ({ x: sum.x + point.x / candidates.length, y: sum.y + point.y / candidates.length }), { x: 0, y: 0 });
  // Use the point farthest from the component center as a stable endpoint,
  // then walk to nearest neighbors. Direction is intentionally not inferred.
  let start = 0, farthest = -1;
  candidates.forEach((point, index) => { const distance = (point.x - center.x) ** 2 + (point.y - center.y) ** 2; if (distance > farthest) { farthest = distance; start = index; } });
  const ordered = [candidates[start]]; const remaining = candidates.filter((_, index) => index !== start);
  while (remaining.length && ordered.length < 150) {
    const previous = ordered[ordered.length - 1]; let nearest = 0, distance = Infinity;
    remaining.forEach((point, index) => { const next = (point.x - previous.x) ** 2 + (point.y - previous.y) ** 2; if (next < distance) { distance = next; nearest = index; } });
    ordered.push(remaining.splice(nearest, 1)[0]);
  }
  const points = ordered.filter((point, index) => index === 0 || index === ordered.length - 1 || Math.hypot(point.x - ordered[index - 1].x, point.y - ordered[index - 1].y) >= cellSize * 1.2).map((point) => ({ x: point.x / width, y: point.y / height }));
  return points.length >= 6 ? { points, confidence: Math.min(1, winnerScore / 500) } : null;
}

export function ScreenshotTracer({ image, onConfirm, onCancel }: { image: File; onConfirm: (file: File) => void; onCancel: () => void }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const bitmap = useRef<HTMLImageElement | null>(null);
  const [points, setPoints] = useState<TracePoint[]>([]);
  const [suggestion, setSuggestion] = useState<SuggestedTrace | null>(null);

  useEffect(() => {
    const url = URL.createObjectURL(image);
    const next = new Image();
    next.onload = () => {
      bitmap.current = next;
      const detected = suggestedRoute(next);
      setSuggestion(detected); setPoints(detected?.points || []); draw(detected?.points || []);
    };
    next.src = url;
    return () => URL.revokeObjectURL(url);
  }, [image]);

  function draw(route: TracePoint[]) {
    const element = canvas.current, source = bitmap.current;
    if (!element || !source) return;
    const width = 760, height = Math.max(320, Math.round(width * source.height / source.width));
    element.width = width; element.height = height;
    const context = element.getContext("2d"); if (!context) return;
    context.drawImage(source, 0, 0, width, height); context.fillStyle = "rgba(11, 35, 24, .18)"; context.fillRect(0, 0, width, height);
    if (!route.length) return;
    context.strokeStyle = "#ff7448"; context.lineWidth = 5; context.lineCap = "round"; context.lineJoin = "round"; context.beginPath();
    route.forEach((point, index) => { const x = point.x * width, y = point.y * height; if (index === 0) context.moveTo(x, y); else context.lineTo(x, y); });
    context.stroke(); route.forEach((point, index) => { context.beginPath(); context.fillStyle = index === 0 ? "#d5b66f" : index === route.length - 1 ? "#f3f0e7" : "#ff7448"; context.arc(point.x * width, point.y * height, index === 0 || index === route.length - 1 ? 7 : 2, 0, Math.PI * 2); context.fill(); });
  }

  function addPoint(event: PointerEvent<HTMLCanvasElement>) {
    const rect = event.currentTarget.getBoundingClientRect();
    const point = { x: Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width)), y: Math.max(0, Math.min(1, (event.clientY - rect.top) / rect.height)) };
    setSuggestion(null); setPoints((current) => { const next = [...current, point]; draw(next); return next; });
  }
  function undo() { setSuggestion(null); setPoints((current) => { const next = current.slice(0, -1); draw(next); return next; }); }
  function clear() { setSuggestion(null); setPoints([]); draw([]); }

  const detected = Boolean(suggestion && points.length >= 6);
  return <div className="screenshot-tracer">
    <div className="tracer-head"><div><span>GEOMETRY-ONLY IMPORT</span><h3>{detected ? "Route found in your screenshot." : "Trace the highlighted route."}</h3><p>{detected ? "Meander found a likely highlighted path locally in your browser. Check the orange overlay, then use it or clear it and trace manually." : "No clear highlighted path was found. Click along the walk from start to finish; add more points around bends."}</p></div><button type="button" onClick={onCancel}>Close</button></div>
    <canvas ref={canvas} onPointerDown={addPoint} aria-label="Route screenshot tracing canvas" />
    <div className="tracer-actions"><span>{points.length < 6 ? `${6 - points.length} more points needed` : detected ? `${points.length} suggested route points · ready` : `${points.length} route points · ready`}</span><div><button type="button" onClick={undo} disabled={!points.length}>Undo</button><button type="button" onClick={clear} disabled={!points.length}>Trace manually</button><button className="confirm-trace" type="button" disabled={points.length < 6} onClick={() => onConfirm(routeFile(points))}>{detected ? "Use suggested route →" : "Use this route →"}</button></div></div>
    <small>The screenshot stays on this device. Meander extracts only a geometry-only path; it does not infer distance, elevation, pace, or timestamps.</small>
  </div>;
}
