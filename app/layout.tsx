import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Meander — Every Walk Leaves a Pattern",
  description: "Turn the geometry of a real walk into a one-of-one generative print.",
  icons: {
    icon: [{ url: "/icon.svg", type: "image/svg+xml" }],
    shortcut: "/favicon.svg",
  },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>;
}
