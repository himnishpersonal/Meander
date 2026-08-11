"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";

const API = (process.env.NEXT_PUBLIC_MEANDER_API_URL || "http://localhost:8080").replace(/\/$/, "");

export default function SignInPage() {
  const [email, setEmail] = useState(""); const [message, setMessage] = useState(""); const [link, setLink] = useState(""); const [error, setError] = useState("");
  async function submit(event: FormEvent) { event.preventDefault(); setError(""); setLink(""); setMessage("Sending your sign-in link…"); try { const response = await fetch(`${API}/api/v1/auth/magic-link`, { method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({email}) }); const body = await response.json(); if (!response.ok) throw new Error(body.error || "Could not send a sign-in link."); setMessage(body.status === "development_magic_link" ? "Development sign-in link created." : "Check your email for a secure sign-in link."); if (body.magic_link) setLink(body.magic_link); } catch (reason) { setMessage(""); setError(reason instanceof Error ? reason.message : "Could not send a sign-in link."); } }
  return <main className="sign-in-page"><header className="site-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><Link className="nav-cta" href="/create">Create first <span>↗</span></Link></header><section className="sign-in-card"><p className="section-kicker">Save your work</p><h1>Sign in to<br />keep moving.</h1><p>We use a secure email link—no password to remember. Your library stays private unless you choose to share a work.</p><form onSubmit={submit}><label htmlFor="email">Email address</label><input id="email" type="email" required value={email} onChange={(event)=>setEmail(event.target.value)} placeholder="you@example.com" /><button className="primary-action" type="submit">Send secure link <span>→</span></button></form>{message && <p className="auth-message">{message}</p>}{link && <a className="dev-magic-link" href={link}>Open development sign-in link ↗</a>}{error && <p className="engine-error">{error}</p>}<small>By continuing, you agree to receive a sign-in email. You can delete your account and artwork at any time.</small></section></main>;
}
