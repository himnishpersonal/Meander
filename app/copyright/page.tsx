import Link from "next/link";
import { AccountNav } from "@/app/account-nav";

export const metadata = { title: "Copyright | Meander" };

export default function CopyrightPage() {
  return <main className="legal-page"><header className="site-header"><Link className="brand" href="/"><span className="brand-line" />Meander</Link><nav aria-label="Main navigation"><Link href="/how-it-works">How Meander works</Link><Link href="/gallery">Gallery</Link></nav><AccountNav /></header>
    <section className="legal-hero"><p className="section-kicker">Copyright and attribution</p><h1>Respect the<br /><span>journey behind it.</span></h1><p>Meander is designed for original creation from walks you have the right to use.</p></section>
    <section className="legal-content"><article><h2>Upload only what you may use</h2><p>Only upload route files, screenshots, and other inputs you own or have permission to use. Do not use Meander to reproduce or distribute copyrighted maps, imagery, logos, artwork, or private routes without permission.</p></article>
      <article><h2>Map-source note</h2><p>Meander’s public validation fixtures include OpenStreetMap geometry. Attribution: <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">© OpenStreetMap contributors</a>, available under the Open Database License. Fixtures are used for engine testing, not navigation. Do not treat screenshots from other mapping services as freely reusable.</p></article>
      <article><h2>Report a concern</h2><p>If a public Meander artwork infringes your copyright, trademark, privacy, or other rights, report the artwork URL, your contact details, a description of the work at issue, and a good-faith statement of your claim. Before public gallery launch, Meander must publish its designated copyright-agent contact here and, if launching in the United States, register that agent with the U.S. Copyright Office.</p></article>
      <article><h2>Removal process</h2><p>Meander will review credible reports, remove or disable access when appropriate, notify the uploader where required, and keep a record of the decision. This page is not a completed DMCA designation until the contact and registration steps above are complete.</p></article>
    </section><footer className="legal-footer"><Link href="/privacy">Privacy</Link><Link href="/terms">Terms</Link><Link href="/account">Account controls</Link></footer></main>;
}
