"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { API, MEANDER_REQUEST_HEADERS } from "@/app/api";

type CurrentUser = { DisplayName?: string; Email?: string; display_name?: string; email?: string };

export function AccountNav() {
  const [user, setUser] = useState<CurrentUser | null | undefined>(undefined);
  const [signingOut, setSigningOut] = useState(false);

  useEffect(() => {
    let mounted = true;
    fetch(`${API}/api/v1/me`, { credentials: "include", cache: "no-store" })
      .then(async (response) => response.ok ? response.json() as Promise<CurrentUser> : null)
      .then((currentUser) => { if (mounted) setUser(currentUser); })
      .catch(() => { if (mounted) setUser(null); });
    return () => { mounted = false; };
  }, []);

  async function signOut() {
    setSigningOut(true);
    try {
      await fetch(`${API}/api/v1/auth/logout`, { method: "POST", credentials: "include", headers: MEANDER_REQUEST_HEADERS });
    } finally {
      window.location.assign("/");
    }
  }

  if (user === undefined) return <span className="account-loading" aria-label="Checking account" />;
  if (!user) return <Link className="nav-cta" href="/sign-in?returnTo=/library">Sign in <span>↗</span></Link>;

  const displayName = user.DisplayName || user.display_name || user.Email || user.email || "Your account";
  const firstName = displayName.split(/\s|@/)[0];
  return <div className="account-nav"><Link href="/library" aria-label={`${displayName}'s artwork library`}><span className="account-avatar">{firstName.charAt(0).toUpperCase()}</span><span>{firstName}</span></Link><button type="button" onClick={signOut} disabled={signingOut}>{signingOut ? "Signing out…" : "Sign out"}</button></div>;
}
