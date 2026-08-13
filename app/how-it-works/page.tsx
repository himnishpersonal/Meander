"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AccountNav } from "@/app/account-nav";
import { BrandMark } from "@/app/brand-mark";
import { ProcessVisual } from "@/app/how-it-works/process-visual";

type Stage = "source" | "fingerprint" | "field" | "canonical";

const stages: Array<{ id: Stage; number: string; kicker: string; title: string; body: string; reads: string; creates: string }> = [
  { id: "source", number: "01", kicker: "Bring your walk", title: "Start with somewhere you actually went.", body: "Upload a GPX or OSM activity file, trace the route from a screenshot, or try a public sample. The path is the only required input; location, time, and music can shape the atmosphere.", reads: "Ordered route points · elevation · timestamps", creates: "A repaired, normalized path that preserves direction, turns, loops, and recognizable shape." },
  { id: "fingerprint", number: "02", kicker: "Read its movement", title: "The engine finds the walk inside the line.", body: "Meander measures curvature, direction changes, loop closure, rhythm, pauses, pace changes, and climbing. A first-ever upload uses the same global calibration as every other walk—no personal history required.", reads: "Turns · curves · pauses · pace · climbing · density", creates: "A movement fingerprint made of meaningful events, strengths, directions, and positions." },
  { id: "field", number: "03", kicker: "Build the field", title: "Movement expands beyond the route.", body: "The route becomes the quiet spine of the composition. Flowing bundles grow from its direction while color, texture, hierarchy, and negative space respond to the route and optional atmosphere.", reads: "Route tangent · event strength · time · music energy", creates: "A directional field where the original walk remains discoverable without becoming a literal map." },
  { id: "canonical", number: "04", kicker: "Choose one work", title: "One composition remains.", body: "The Go engine creates private candidates and scores route visibility, flow coherence, collision, hierarchy, negative space, color structure, and composition. You receive one canonical artwork—not a page of finalists.", reads: "WALK-ART-V1 global quality profile", creates: "A repeatable PNG and SVG whose recipe can be traced back to measurable route features." },
];

export default function HowItWorksPage() {
  const [active, setActive] = useState<Stage>("source");

  useEffect(() => {
    const nodes = Array.from(document.querySelectorAll<HTMLElement>("[data-process-stage]"));
    const observer = new IntersectionObserver((entries) => {
      const visible = entries.filter((entry) => entry.isIntersecting).sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
      if (visible) setActive((visible.target as HTMLElement).dataset.processStage as Stage);
    }, { rootMargin: "-28% 0px -42%", threshold: [0.2, 0.55, 0.8] });
    nodes.forEach((node) => observer.observe(node));
    return () => observer.disconnect();
  }, []);

  return <main className="process-page">
    <header className="site-header"><Link className="brand" href="/"><BrandMark />Meander</Link><nav aria-label="Main navigation"><Link className="active" href="/how-it-works">How Meander works</Link><Link href="/gallery">Gallery</Link><Link href="/library">Library</Link></nav><AccountNav /></header>
    <section className="process-hero">
      <p className="section-kicker">From a real route to a one-of-one work</p>
      <h1>Your walk supplies<br />the <span>direction.</span></h1>
      <p>Meander does not invent a picture from a prompt. It interprets the geometry and movement of somewhere you actually went.</p>
      <div className="process-index" aria-label="Walkthrough chapters"><span>01 Route</span><span>02 Fingerprint</span><span>03 Field</span><span>04 Artwork</span><span>05 Keep</span></div>
    </section>

    <section className="process-story" aria-label="How a route becomes art">
      <div className="process-visual-sticky"><div className="process-visual-frame"><ProcessVisual stage={active} /><span>CENTRAL PARK · ONE ROUTE THROUGH EVERY STAGE</span></div><p>{stages.find((stage) => stage.id === active)?.number} / 04</p></div>
      <div className="process-chapters">
        {stages.map((stage) => <article key={stage.id} data-process-stage={stage.id} className={active === stage.id ? "active" : ""}>
          <span>{stage.number} · {stage.kicker}</span><h2>{stage.title}</h2><p>{stage.body}</p>
          <div className="process-facts"><div><b>What Meander reads</b><p>{stage.reads}</p></div><div><b>What it creates</b><p>{stage.creates}</p></div></div>
        </article>)}
      </div>
    </section>

    <section className="art-lifecycle">
      <div className="lifecycle-heading"><p className="section-kicker">05 · Keep it your way</p><h2>The artwork is yours.<br />The raw route stays private.</h2><p>Every new work begins in your private library. You decide if it goes anywhere else.</p></div>
      <div className="lifecycle-steps">
        <article><span>01</span><h3>Saved privately</h3><p>Only you can open the artwork. The uploaded route or screenshot is not stored.</p></article>
        <article><span>02</span><h3>Download it</h3><p>Export a share-ready PNG or a scalable SVG for printing and further design work.</p></article>
        <article><span>03</span><h3>Share by link</h3><p>Create a private link for specific people. It is accessible only to someone who has the link.</p></article>
        <article><span>04</span><h3>Publish optionally</h3><p>Choose to place the finished artwork in Meander’s public gallery—never the raw route.</p></article>
        <article><span>05</span><h3>Take it back</h3><p>Make shared or published work private again from your library at any time.</p></article>
      </div>
      <div className="privacy-boundary"><div><span>Processed temporarily</span><strong>GPX · OSM · route screenshot</strong></div><i>→</i><div><span>Stored in your account</span><strong>Artwork · fingerprint · engine recipe</strong></div></div>
    </section>

    <section className="process-cta"><p className="section-kicker">One walk. One work.</p><h2>Your walk already has a shape.<br /><span>Meander reveals it.</span></h2><Link href="/create">Create from your route <b>↗</b></Link></section>
    <footer><Link className="brand" href="/"><BrandMark />Meander</Link><p>The geometry of going somewhere.</p><div><Link href="/how-it-works">Process</Link><Link href="/gallery">Gallery</Link><Link href="/library">Library</Link></div><small>© 2026 MEANDER · RAW ROUTES STAY PRIVATE</small></footer>
  </main>;
}
