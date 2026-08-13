"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AccountNav } from "@/app/account-nav";
import { API } from "@/app/api";
import { BrandMark } from "@/app/brand-mark";

type Artwork = { id: string; title: string; subtitle: string; created_at: string; preview_url: string; share_url: string; features?: { DistanceKM?: number } };

export default function GalleryPage() {
  const [artworks, setArtworks] = useState<Artwork[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    fetch(`${API}/api/v1/gallery`, { cache: "no-store" }).then(async (response) => { if (!response.ok) throw new Error(); return response.json(); }).then((body) => { setArtworks(body.artworks || []); setStatus("ready"); }).catch(() => setStatus("error"));
  }, []);

  return <main className="public-gallery-page">
    <header className="site-header"><Link className="brand" href="/"><BrandMark />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link className="active" href="/gallery">Gallery</Link><Link href="/library">Library</Link></nav><AccountNav /></header>
    <section className="public-gallery-hero"><p className="section-kicker">The public field</p><h1>Walks people chose<br />to <span>let wander.</span></h1><p>Finished artworks intentionally published by their creators. Raw routes, GPS files, and screenshots never appear here.</p></section>
    <section className="public-gallery-content">
      {status === "loading" && <p className="gallery-state">Opening the gallery…</p>}
      {status === "error" && <div className="gallery-state"><strong>The gallery is taking a short walk.</strong><p>Please return in a moment.</p></div>}
      {status === "ready" && artworks.length === 0 && <div className="gallery-empty"><p className="section-kicker">The first wall is open</p><h2>No one has published a work yet.</h2><p>Create the first piece, then choose “Publish to gallery” from your private library.</p><Link className="primary-action" href="/create">Create the first work <span>↗</span></Link></div>}
      {status === "ready" && artworks.length > 0 && <div className="public-gallery-grid">{artworks.map((artwork, index) => <Link href={artwork.share_url} key={artwork.id} className={index % 5 === 0 ? "gallery-feature" : ""}><div><img src={`${API}${artwork.preview_url}`} alt={`Published Meander artwork for ${artwork.title}`} /></div><span>{new Date(artwork.created_at).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })}</span><h2>{artwork.title}</h2><p>{artwork.subtitle}</p></Link>)}</div>}
    </section>
    <section className="gallery-cta"><div><p className="section-kicker">Keep yours private—or add it here</p><h2>Every work begins with a real walk.</h2></div><Link href="/create">Create from your route <span>↗</span></Link></section>
  </main>;
}
