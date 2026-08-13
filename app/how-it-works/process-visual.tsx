"use client";

import { useEffect, useRef } from "react";

type ProcessStage = "source" | "fingerprint" | "field" | "canonical";

function point(index: number, total: number, width: number, height: number) {
  const p = index / Math.max(total - 1, 1);
  return {
    x: width * (0.17 + p * 0.66) + Math.sin(p * 18) * width * 0.055 + Math.sin(p * 7) * width * 0.025,
    y: height * (0.68 - p * 0.34) + Math.sin(p * 12.5) * height * 0.14 + Math.cos(p * 29) * height * 0.025,
  };
}

function route(context: CanvasRenderingContext2D, width: number, height: number, color: string, lineWidth: number, offset = 0) {
  context.beginPath();
  for (let index = 0; index < 180; index++) {
    const current = point(index, 180, width, height);
    const previous = point(Math.max(0, index - 1), 180, width, height);
    const next = point(Math.min(179, index + 1), 180, width, height);
    const dx = next.x - previous.x;
    const dy = next.y - previous.y;
    const length = Math.hypot(dx, dy) || 1;
    const x = current.x + (-dy / length) * offset;
    const y = current.y + (dx / length) * offset;
    if (index === 0) context.moveTo(x, y); else context.lineTo(x, y);
  }
  context.strokeStyle = color;
  context.lineWidth = lineWidth;
  context.lineCap = "round";
  context.lineJoin = "round";
  context.stroke();
}

export function ProcessVisual({ stage }: { stage: ProcessStage }) {
  const canvas = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const draw = () => {
      if (!canvas.current) return;
      const rect = canvas.current.getBoundingClientRect();
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(300, Math.floor(rect.width));
      const height = Math.max(360, Math.floor(rect.height));
      canvas.current.width = width * ratio;
      canvas.current.height = height * ratio;
      const context = canvas.current.getContext("2d");
      if (!context) return;
      context.scale(ratio, ratio);
      context.fillStyle = "#ece8dc";
      context.fillRect(0, 0, width, height);

      if (stage === "source") {
        context.strokeStyle = "rgba(23,55,43,.09)";
        context.lineWidth = 1;
        for (let x = 0; x < width; x += 38) { context.beginPath(); context.moveTo(x, 0); context.lineTo(x, height); context.stroke(); }
        for (let y = 0; y < height; y += 38) { context.beginPath(); context.moveTo(0, y); context.lineTo(width, y); context.stroke(); }
        route(context, width, height, "rgba(228,108,67,.18)", 12);
        route(context, width, height, "#e46c43", 3.5);
      }

      if (stage === "fingerprint") {
        route(context, width, height, "rgba(23,55,43,.12)", 16);
        route(context, width, height, "#17372b", 2.4);
        for (let index = 8; index < 180; index += 15) {
          const current = point(index, 180, width, height);
          context.beginPath();
          context.arc(current.x, current.y, index % 30 === 8 ? 6 : 3.5, 0, Math.PI * 2);
          context.fillStyle = index % 30 === 8 ? "#e46c43" : "#d5b66f";
          context.fill();
        }
      }

      if (stage === "field" || stage === "canonical") {
        const colors = ["rgba(42,125,185,.62)", "rgba(121,91,185,.58)", "rgba(228,108,67,.6)", "rgba(96,122,88,.62)"];
        for (let band = -13; band <= 13; band++) {
          const distance = band * 7.2;
          const fade = Math.max(0.17, 0.68 - Math.abs(band) * 0.032);
          route(context, width, height, colors[(band + 28) % colors.length].replace(/\.[0-9]+\)$/, `${fade})`), Math.abs(band) < 3 ? 2.5 : 1.25, distance);
        }
        route(context, width, height, "rgba(24,84,123,.16)", 7);
        route(context, width, height, "rgba(24,84,123,.46)", 1.7);
      }
    };
    draw();
    window.addEventListener("resize", draw);
    return () => window.removeEventListener("resize", draw);
  }, [stage]);

  if (stage === "canonical") {
    return <img className="process-final-art" src="/generated/central-park.svg" alt="Canonical artwork generated from the same walking route" />;
  }
  return <canvas ref={canvas} className="process-canvas" aria-label={`${stage} view of the route-to-art process`} />;
}
