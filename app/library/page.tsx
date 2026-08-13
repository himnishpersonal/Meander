"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AccountNav } from "@/app/account-nav";
import { API } from "@/app/api";
import { BrandMark } from "@/app/brand-mark";

type Visibility = "private" | "unlisted" | "public";
type Artwork = { id: string; share_id: string; title: string; subtitle: string; palette: string; visibility: Visibility; created_at: string; preview_url: string; share_url: string };

const accessCopy: Record<Visibility, { label: string; detail: string }> = {
  private: { label: "Only you", detail: "This work is private in your library." },
  unlisted: { label: "Shared by link", detail: "Only people with its private link can open it." },
  public: { label: "Published", detail: "This work appears in the public Meander gallery." },
};

export default function LibraryPage() {
  const [artworks, setArtworks] = useState<Artwork[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [busy, setBusy] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<Record<string, string>>({});

  useEffect(() => {
    fetch(`${API}/api/v1/me/artworks`, { credentials: "include", cache: "no-store" })
      .then(async (response) => { if (!response.ok) throw new Error(); return response.json(); })
      .then((body) => { setArtworks(body.artworks || []); setStatus("ready"); })
      .catch(() => setStatus("error"));
  }, []);

  async function changeVisibility(artwork: Artwork, visibility: Visibility, message: string) {
    setBusy(artwork.id);
    const response = await fetch(`${API}/api/v1/artworks/${artwork.id}`, { method: "PATCH", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ visibility }) });
    setBusy(null);
    if (!response.ok) { setFeedback((current) => ({ ...current, [artwork.id]: "That change could not be saved." })); return false; }
    setArtworks((items) => items.map((item) => item.id === artwork.id ? { ...item, visibility } : item));
    setFeedback((current) => ({ ...current, [artwork.id]: message }));
    return true;
  }

  async function copyShareLink(artwork: Artwork) {
    if (artwork.visibility === "private") {
      const updated = await changeVisibility(artwork, "unlisted", "Private share link created.");
      if (!updated) return;
    }
    const url = `${window.location.origin}${artwork.share_url}`;
    try { await navigator.clipboard.writeText(url); setFeedback((current) => ({ ...current, [artwork.id]: "Private share link copied." })); }
    catch { setFeedback((current) => ({ ...current, [artwork.id]: url })); }
  }

  return <main className="library-page">
    <header className="site-header"><Link className="brand" href="/"><BrandMark />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link href="/gallery">Gallery</Link><Link href="/create">Create</Link></nav><AccountNav /></header>
    <section className="library-intro"><p className="section-kicker">Your library</p><h1>Walks you<br />made <span>yours.</span></h1><p>Every new work begins private. Download it, make a link for specific people, or intentionally publish it to the gallery.</p></section>
    <section className="library-content">
      {status === "loading" && <p className="library-state">Loading your Meanders…</p>}
      {status === "error" && <div className="library-state"><strong>Sign in to see your library.</strong><p>Your saved artworks remain private. Continue with Google to return to your work.</p><Link className="primary-action" href="/sign-in?returnTo=/library">Continue with Google <span>↗</span></Link></div>}
      {status === "ready" && artworks.length === 0 && <div className="library-state"><strong>Your first walk is waiting.</strong><p>Generate an artwork and it will arrive here private by default.</p><Link className="primary-action" href="/create">Create from a route <span>↗</span></Link></div>}
      {status === "ready" && artworks.length > 0 && <div className="library-grid">{artworks.map((artwork) => {
        const access = accessCopy[artwork.visibility];
        return <article key={artwork.id}>
          <div className="library-art"><img src={`${API}${artwork.preview_url}`} alt={`Generated artwork for ${artwork.title}`} /><span className={`access-chip ${artwork.visibility}`}>{access.label}</span></div>
          <div className="library-card-copy"><p>{new Date(artwork.created_at).toLocaleDateString()}</p><h2>{artwork.title}</h2><small>{artwork.subtitle}</small>
            <div className="access-state"><span>{access.label}</span><p>{access.detail}</p></div>
            <div className="library-primary-actions"><a href={`${API}${artwork.preview_url}`} download>Download PNG ↓</a><button type="button" onClick={() => copyShareLink(artwork)} disabled={busy === artwork.id}>{artwork.visibility === "private" ? "Create share link" : "Copy share link"} ↗</button></div>
            <div className="library-access-actions">
              {artwork.visibility !== "public" && <button type="button" onClick={() => changeVisibility(artwork, "public", "Published to the Meander gallery.")} disabled={busy === artwork.id}>Publish to gallery</button>}
              {artwork.visibility !== "private" && <button type="button" onClick={() => changeVisibility(artwork, "private", "Made private. Existing links no longer open.")} disabled={busy === artwork.id}>Make private again</button>}
              {artwork.visibility !== "private" && <Link href={artwork.share_url}>Open shared work</Link>}
            </div>
            {feedback[artwork.id] && <p className="library-feedback" role="status">{feedback[artwork.id]}</p>}
          </div>
        </article>;
      })}</div>}
    </section>
  </main>;
}
