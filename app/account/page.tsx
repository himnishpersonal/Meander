"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { API, MEANDER_REQUEST_HEADERS } from "@/app/api";
import { AccountNav } from "@/app/account-nav";

export default function AccountPage() {
  const [confirmed, setConfirmed] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [signedIn, setSignedIn] = useState<boolean | null>(null);

  useEffect(() => {
    fetch(`${API}/api/v1/me`, { credentials: "include", cache: "no-store" })
      .then((r) => setSignedIn(r.ok)).catch(() => setSignedIn(false));
  }, []);

  async function deleteAccount(event: FormEvent) {
    event.preventDefault();
    if (confirmed !== "DELETE") return;
    setBusy(true); setMessage("");
    try {
      const response = await fetch(`${API}/api/v1/me`, { method: "DELETE", credentials: "include", headers: { "Content-Type": "application/json", ...MEANDER_REQUEST_HEADERS }, body: JSON.stringify({ confirmation: confirmed }) });
      if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || "We could not delete your account.");
      window.location.assign("/?account=deleted");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "We could not delete your account.");
      setBusy(false);
    }
  }

  return <main className="legal-page">
    <header className="site-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link href="/privacy">Privacy</Link></nav><AccountNav /></header>
    <section className="legal-hero"><p className="section-kicker">Account controls</p><h1>Your account,<br /><span>your decision.</span></h1><p>Meander keeps finished artwork in your library. Raw route uploads and screenshots are processed for a creation and are not retained as account files.</p></section>
    <section className="legal-content account-content">
      {signedIn === false && <div className="legal-callout"><strong>Sign in to manage your account.</strong><Link href="/sign-in?returnTo=/account">Continue with Google →</Link></div>}
      {signedIn && <><article><h2>Artwork and sharing</h2><p>Use your <Link href="/library">library</Link> to download, delete, make private, share by link, or publish individual works. Making an artwork private revokes its Meander share link; it cannot retract copies someone already downloaded.</p></article>
      <article className="danger-zone"><p className="section-kicker">Permanent action</p><h2>Delete your account</h2><p>This permanently removes your account, active sessions, finished artworks, and associated stored artwork files. It cannot be undone.</p><form onSubmit={deleteAccount}><label htmlFor="delete-confirmation">Type <b>DELETE</b> to confirm</label><div><input id="delete-confirmation" value={confirmed} onChange={(e) => setConfirmed(e.target.value)} autoComplete="off" /><button type="submit" disabled={busy || confirmed !== "DELETE"}>{busy ? "Deleting…" : "Delete account permanently"}</button></div>{message && <p className="engine-error" role="alert">{message}</p>}</form></article></>}
    </section>
    <footer className="legal-footer"><Link href="/privacy">Privacy</Link><Link href="/terms">Terms</Link><Link href="/copyright">Copyright</Link></footer>
  </main>;
}
