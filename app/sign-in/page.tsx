"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { API } from "@/app/api";

type GoogleCredentialResponse = { credential: string };
type GoogleIdentity = {
  initialize: (configuration: { client_id: string; callback: (response: GoogleCredentialResponse) => void; auto_select?: boolean }) => void;
  renderButton: (element: HTMLElement, options: Record<string, string | number>) => void;
};
type AuthState = "checking" | "ready" | "signing-in" | "success" | "error";

declare global {
  interface Window {
    google?: { accounts: { id: GoogleIdentity } };
  }
}

function safeReturnPath() {
  const requested = new URLSearchParams(window.location.search).get("returnTo");
  return requested?.startsWith("/") && !requested.startsWith("//") ? requested : "/create";
}

export default function SignInPage() {
  const button = useRef<HTMLDivElement>(null);
  const [error, setError] = useState("");
  const [clientID, setClientID] = useState(process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || "");
  const [configurationLoaded, setConfigurationLoaded] = useState(Boolean(process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID));
  const [authState, setAuthState] = useState<AuthState>("checking");

  useEffect(() => {
    let mounted = true;
    fetch(`${API}/api/v1/me`, { credentials: "include", cache: "no-store" })
      .then((response) => {
        if (!mounted) return;
        if (response.ok) window.location.replace(safeReturnPath());
        else setAuthState("ready");
      })
      .catch(() => { if (mounted) setAuthState("ready"); });
    return () => { mounted = false; };
  }, []);

  useEffect(() => {
    if (clientID) return;
    let mounted = true;
    fetch(`${API}/api/v1/config`, { cache: "no-store" })
      .then(async (response) => {
        const body = await response.json();
        if (!response.ok) throw new Error("Configuration unavailable");
        return body as { google_client_id?: string };
      })
      .then((body) => { if (mounted) setClientID(body.google_client_id || ""); })
      .catch(() => { if (mounted) setError("Google Sign-In configuration could not be loaded. Please try again shortly."); })
      .finally(() => { if (mounted) setConfigurationLoaded(true); });
    return () => { mounted = false; };
  }, [clientID]);

  useEffect(() => {
    if (!clientID) return;
    let mounted = true;

    const signIn = async (response: GoogleCredentialResponse) => {
      setError("");
      setAuthState("signing-in");
      if (button.current) button.current.style.display = "none";
      try {
        const result = await fetch(`${API}/api/v1/auth/google`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ credential: response.credential }),
        });
        const body = await result.json().catch(() => ({}));
        if (!result.ok) throw new Error(body.error || "Google sign-in could not be completed.");

        const session = await fetch(`${API}/api/v1/me`, { credentials: "include", cache: "no-store" });
        if (!session.ok) throw new Error("Google approved the sign-in, but the browser did not keep your session. Please allow cookies for Meander and try again.");

        setAuthState("success");
        window.setTimeout(() => window.location.replace(safeReturnPath()), 450);
      } catch (reason) {
        if (!mounted) return;
        setAuthState("error");
        setError(reason instanceof Error ? reason.message : "Google sign-in could not be completed.");
        render();
      }
    };

    const render = () => {
      if (!mounted || !button.current || !window.google) return;
      button.current.replaceChildren();
      button.current.style.display = "block";
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
      script.onerror = () => { if (mounted) { setAuthState("error"); setError("Google Sign-In could not load. Check your connection and try again."); } };
      document.head.appendChild(script);
    }
    return () => { mounted = false; };
  }, [clientID]);

  const busy = authState === "checking" || authState === "signing-in" || authState === "success";
  return <main className="sign-in-page"><header className="site-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><Link className="nav-cta" href="/create">Create first <span>↗</span></Link></header><section className="sign-in-card"><p className="section-kicker">Save your work</p><h1>Sign in to<br />keep moving.</h1><p>Use your Google account to save your walks, keep artwork private, and connect Strava when it arrives.</p><div className="google-sign-in" ref={button} aria-label="Continue with Google" />{busy && <div className={`auth-progress ${authState === "success" ? "complete" : ""}`} role="status"><span>{authState === "success" ? "✓" : ""}</span><div><strong>{authState === "checking" ? "Checking your session…" : authState === "signing-in" ? "Creating your Meander account…" : "You’re signed in."}</strong><p>{authState === "success" ? "Taking you to the creation studio." : "This should only take a moment."}</p></div></div>}{error && <p className="engine-error" role="alert">{error}</p>}{configurationLoaded && !clientID && !error && <p className="engine-error" role="alert">Google Sign-In is still being connected. Please try again shortly.</p>}<small>Meander only uses your Google identity to create your account. Your activity data stays separate and is only connected when you choose Strava.</small></section></main>;
}
