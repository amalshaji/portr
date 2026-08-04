import type { Metadata } from "next";
import { LandingPage } from "@/components/landing/landing-page";

const title = "Portr - Open-source tunnels for development teams";
const description =
  "Expose local HTTP, TCP, and WebSocket services on public URLs with a self-hosted tunnel, local request inspector, replay, and team controls.";

export const metadata: Metadata = {
  title,
  description,
  keywords: [
    "tunnel",
    "ngrok alternative",
    "self-hosted",
    "HTTP tunnel",
    "TCP tunnel",
    "WebSocket",
    "team collaboration",
    "request inspector",
    "localhost tunnel",
    "portr",
  ],
  openGraph: {
    title,
    description,
    type: "website",
    url: "https://portr.dev",
    images: "/og.png",
  },
  twitter: {
    card: "summary_large_image",
    title,
    description,
    images: "/og.png",
  },
};

export default function HomePage() {
  return <LandingPage />;
}
