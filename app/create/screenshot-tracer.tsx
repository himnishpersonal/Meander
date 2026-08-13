"use client";

import { PointerEvent, useEffect, useRef, useState } from "react";

type TracePoint = { x: number; y: number };

function routeFile(points: TracePoint[]) {
  const lat0 = 40.75;
  const lon0 = -73.98;
  const body = points.map((point, index) => `<trkpt lat="${(lat0 + (1 - point.y) * .02).toFixed(7)}" lon="${(lon0 + point.x * .02).toFixed(7)}"><time>${new Date(Date.UTC(2026, 0, 1, 12, 0, index * 8)).toISOString()}</time></trkpt>`).join("");
  return new File([`<?xml version="1.0"?><gpx version="1.1" creator="Meander screenshot tracer"><trk><name>Screenshot route</name><trkseg>${body}</trkseg></trk></gpx>`], "screenshot-route.gpx", { type: "application/gpx+xml" });
}

export function ScreenshotTracer({ image, onConfirm, onCancel }: { image: File; onConfirm: (file: File) => void; onCancel: () => void }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const bitmap = useRef<HTMLImageElement | null>(null);
  const [points, setPoints] = useState<TracePoint[]>([]);

  useEffect(() => {
    const url = URL.createObjectURL(image);
    const next = new Image();
    next.onload = () => { bitmap.current = next; draw([]); };
    next.src = url;
    return () => URL.revokeObjectURL(url);
  }, [image]);

  function draw(route: TracePoint[]) {
    const element = canvas.current;
    const source = bitmap.current;
    if (!element || !source) return;
    const width = 760;
    const height = Math.max(320, Math.round(width * source.height / source.width));
    element.width = width;
    element.height = height;
    const context = element.getContext("2d");
    if (!context) return;
    context.drawImage(source, 0, 0, width, height);
    context.fillStyle = "rgba(11, 35, 24, .18)";
    context.fillRect(0, 0, width, height);
    if (route.length) {
      context.strokeStyle = "#ff7448";
      context.lineWidth = 5;
      context.lineCap = "round";
      context.lineJoin = "round";
      context.beginPath();
      route.forEach((point, index) => {
        const x = point.x * width, y = point.y * height;
        if (index === 0) context.moveTo(x, y); else context.lineTo(x, y);
      });
      context.stroke();
      route.forEach((point, index) => {
        context.beginPath();
        context.fillStyle = index === 0 ? "#d5b66f" : index === route.length - 1 ? "#f3f0e7" : "#ff7448";
        context.arc(point.x * width, point.y * height, index === 0 || index === route.length - 1 ? 7 : 3, 0, Math.PI * 2);
        context.fill();
      });
    }
  }

  function addPoint(event: PointerEvent<HTMLCanvasElement>) {
    const rect = event.currentTarget.getBoundingClientRect();
    const point = { x: Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width)), y: Math.max(0, Math.min(1, (event.clientY - rect.top) / rect.height)) };
    setPoints((current) => { const next = [...current, point]; draw(next); return next; });
  }

  function undo() { setPoints((current) => { const next = current.slice(0, -1); draw(next); return next; }); }
  function clear() { setPoints([]); draw([]); }

  return <div className="screenshot-tracer">
    <div className="tracer-head"><div><span>GEOMETRY-ONLY IMPORT</span><h3>Trace the highlighted route.</h3><p>Click along the walk from start to finish. Add more points around bends; the orange preview is the exact geometry Meander will interpret.</p></div><button type="button" onClick={onCancel}>Close</button></div>
    <canvas ref={canvas} onPointerDown={addPoint} aria-label="Route screenshot tracing canvas" />
    <div className="tracer-actions"><span>{points.length < 6 ? `${6 - points.length} more points needed` : `${points.length} route points · ready`}</span><div><button type="button" onClick={undo} disabled={!points.length}>Undo</button><button type="button" onClick={clear} disabled={!points.length}>Clear</button><button className="confirm-trace" type="button" disabled={points.length < 6} onClick={() => onConfirm(routeFile(points))}>Use this route →</button></div></div>
    <small>Screenshots do not contain reliable distance, elevation, pace, or timestamps. Meander uses only the path shape and never invents missing activity data.</small>
  </div>;
}
