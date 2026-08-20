"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { API } from "@/app/api";
import { BrandMark } from "@/app/brand-mark";

type Artwork = { id: string; title: string; subtitle: string; palette: string; preview_url: string; artwork_url: string; visibility: string; created_at?: string; metrics?: { distance_km?: number; hard_turns?: number; composition_score?: number } };

export default function SharedArtwork({ params }: { params: Promise<{ shareId: string }> }) {
  const [artwork, setArtwork] = useState<Artwork | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => { params.then(({ shareId }) => fetch(`${API}/api/v1/share/${shareId}`, { cache: "no-store" }).then(async (response) => { if (!response.ok) throw new Error(); return response.json(); }).then(setArtwork).catch(() => setError(true))); }, [params]);
  if (error) return <main className="share-page"><header className="site-header"><Link className="brand" href="/"><BrandMark />Meander</Link></header><div className="share-state"><p className="section-kicker">Not available</p><h1>This work is private.</h1><p>Its creator has not enabled sharing.</p><Link className="primary-action" href="/create">Make your own <span>↗</span></Link></div></main>;
  if (!artwork) return <main className="share-page"><div className="share-loading"><BrandMark /><span>Opening a walk’s pattern…</span></div></main>;
  const distance = artwork.metrics?.distance_km && artwork.metrics.distance_km > 0 ? `${artwork.metrics.distance_km.toFixed(2)} km` : "Geometry only";
  const rhythm = (artwork.metrics?.hard_turns || 0) > 12 ? "Angular" : (artwork.metrics?.hard_turns || 0) > 4 ? "Varied" : "Flowing";
  const date = artwork.created_at ? new Date(artwork.created_at).toLocaleDateString(undefined, { month: "long", day: "numeric", year: "numeric" }) : "A recorded walk";
  return <main className="share-page">
    <header className="site-header"><Link className="brand" href="/"><BrandMark />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link href="/gallery">Gallery</Link></nav><Link className="nav-cta" href="/create">Make your own <span>↗</span></Link></header>
    <section className="share-work">
      <div className="share-art"><div className="contour-rings" aria-hidden="true" /><img src={`${API}${artwork.preview_url}`} alt={`Meander artwork for ${artwork.title}`} /></div>
      <div className="share-copy"><p className="section-kicker">{artwork.visibility === "public" ? "Published in the Meander gallery" : "Shared privately by its creator"}</p><h1>{artwork.title}</h1><p className="share-subtitle">{artwork.subtitle}</p><p className="share-date">{date}</p><div className="share-metrics"><div><span>Movement</span><strong>{distance}</strong></div><div><span>Route rhythm</span><strong>{rhythm}</strong></div><div><span>Edition</span><strong>One of one</strong></div></div><p className="share-interpretation">A directional composition drawn from the path’s bends, pauses, and overall journey—preserving the feeling of going somewhere without reproducing the map.</p><div className="share-actions"><a className="primary-action" href={`${API}${artwork.preview_url}`} download>Download PNG <span>↓</span></a><a href={`${API}${artwork.artwork_url}`} download>Vector SVG ↓</a></div><small>Created with Meander · raw location data is not included in this shared view</small></div>
    </section>
  </main>;
}
