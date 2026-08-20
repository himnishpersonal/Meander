import Link from "next/link";
import { AccountNav } from "@/app/account-nav";

export const metadata = { title: "Terms | Meander" };

export default function TermsPage() {
  return <main className="legal-page"><header className="site-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link href="/privacy">Privacy</Link></nav><AccountNav /></header>
    <section className="legal-hero"><p className="section-kicker">Terms of use</p><h1>Create with care.<br /><span>Share by choice.</span></h1><p>These draft terms explain the rules for using Meander’s route-to-art creation service.</p><small>Draft only: complete company identity, contact details, governing law, and attorney review before public launch.</small></section>
    <section className="legal-content"><article><h2>Using Meander</h2><p>Meander is a creative tool, not a navigation, fitness, emergency, or safety service. Do not rely on it for route guidance or decisions that could affect safety. You must be at least 16 years old, or the age of digital consent where you live, to create an account.</p></article>
      <article><h2>Your uploads and artwork</h2><p>You keep rights you have in your uploaded material. You grant Meander a limited, non-exclusive license to process your inputs, create the requested artwork, store it for your account, and display it only according to visibility choices you make. You confirm that you have the rights or permission needed to upload route files, screenshots, music/context text, and any other supplied material.</p></article>
      <article><h2>Public sharing</h2><p>You choose whether work remains private, is available by link, or appears in the public gallery. You are responsible for content you publish. Meander may remove public work that violates these terms, infringes rights, threatens safety, or creates legal risk.</p></article>
      <article><h2>Acceptable use</h2><p>Do not upload unlawful material, attempt to bypass limits or access controls, probe other users’ data, automate abusive requests, impersonate others, or use Meander to infringe intellectual-property or privacy rights.</p></article>
      <article><h2>Availability and changes</h2><p>Meander may change, suspend, or discontinue features, especially during beta. To the extent allowed by law, the service is provided as available. These terms must be finalized with applicable warranties, liability limits, dispute terms, business identity, and a legal contact before public launch.</p></article>
    </section><footer className="legal-footer"><Link href="/privacy">Privacy</Link><Link href="/copyright">Copyright</Link><Link href="/account">Account controls</Link></footer></main>;
}
