"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

const API = (process.env.NEXT_PUBLIC_MEANDER_API_URL || "http://localhost:8080").replace(/\/$/, "");

type GoogleCredentialResponse = { credential: string };
type GoogleIdentity = {
  initialize: (configuration: { client_id: string; callback: (response: GoogleCredentialResponse) => void; auto_select?: boolean }) => void;
  renderButton: (element: HTMLElement, options: Record<string, string | number>) => void;
};

declare global {
  interface Window {
    google?: { accounts: { id: GoogleIdentity } };
  }
}

export default function SignInPage() {
  const button = useRef<HTMLDivElement>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const clientID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;

  useEffect(() => {
    if (!clientID) {
      return;
    }

    let mounted = true;
    const signIn = async (response: GoogleCredentialResponse) => {
      setError("");
      setMessage("Securing your Meander…");
      try {
        const result = await fetch(`${API}/api/v1/auth/google`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ credential: response.credential }),
        });
        const body = await result.json();
        if (!result.ok) throw new Error(body.error || "Google sign-in could not be completed.");
        window.location.assign("/library");
      } catch (reason) {
        if (!mounted) return;
        setMessage("");
        setError(reason instanceof Error ? reason.message : "Google sign-in could not be completed.");
      }
    };
    const render = () => {
      if (!mounted || !button.current || !window.google) return;
      button.current.replaceChildren();
      window.google.accounts.id.initialize({ client_id: clientID, callback: signIn, auto_select: false });
      window.google.accounts.id.renderButton(button.current, { theme: "outline", size: "large", text: "continue_with", shape: "rectangular", width: 360 });
    };
    const existing = document.getElementById("google-identity-services");
    if (existing) {
      if (window.google) render(); else existing.addEventListener("load", render, { once: true });
    } else {
      const script = document.createElement("script");
      script.id = "google-identity-services";
      script.src = "https://accounts.google.com/gsi/client";
      script.async = true;
      script.onload = render;
      document.head.appendChild(script);
    }
    return () => { mounted = false; };
  }, [clientID]);

  return <main className="sign-in-page"><header className="site-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><Link className="nav-cta" href="/create">Create first <span>↗</span></Link></header><section className="sign-in-card"><p className="section-kicker">Save your work</p><h1>Sign in to<br />keep moving.</h1><p>Use your Google account to save your walks, keep artwork private, and connect Strava when it arrives.</p><div className="google-sign-in" ref={button} aria-label="Continue with Google" />{message && <p className="auth-message">{message}</p>}{(error || !clientID) && <p className="engine-error">{error || "Google Sign-In is not configured on this device yet."}</p>}<small>Meander only uses your Google identity to create your account. Your activity data stays separate and is only connected when you choose Strava.</small></section></main>;
}
