import "@/app/global.css";
import { Provider } from "./provider";
import { GeistSans } from "geist/font/sans";
import type { ReactNode } from "react";
import type { Metadata, Viewport } from "next";

export const metadata: Metadata = {
  title: {
    template: "%s | Portr",
    default: "Portr - Self-Hosted Tunnel Solution for Teams",
  },
  description:
    "Self-hosted tunnel solution designed for teams. Expose local HTTP, TCP, or WebSocket connections to the public internet with admin dashboard, request inspector, and team collaboration.",
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
  ],
  authors: [{ name: "Amal Shaji", url: "https://github.com/amalshaji" }],
  creator: "Amal Shaji",
  publisher: "Portr",
  metadataBase: new URL("https://portr.dev"),
  alternates: {
    canonical: "/",
  },
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "https://portr.dev",
    title: "Portr - Self-Hosted Tunnel Solution for Teams",
    description:
      "Self-hosted tunnel solution designed for teams. Expose local HTTP, TCP, or WebSocket connections to the public internet.",
    siteName: "Portr",
  },
  twitter: {
    card: "summary_large_image",
    title: "Portr - Self-Hosted Tunnel Solution for Teams",
    description:
      "Self-hosted tunnel solution designed for teams. Expose local HTTP, TCP, or WebSocket connections to the public internet.",
    creator: "@amalshaji",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "white" },
    { media: "(prefers-color-scheme: dark)", color: "black" },
  ],
  width: "device-width",
  initialScale: 1,
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={GeistSans.className} suppressHydrationWarning>
      <head>
        <link rel="icon" href="/logo.svg" type="image/svg+xml" />
        <link rel="apple-touch-icon" href="/apple-touch-icon.svg" />
        <link rel="manifest" href="/manifest.json" />
      </head>
      <body className="flex flex-col min-h-screen">
        <Provider>{children}</Provider>
        {/* 100% privacy-first analytics */}
        <script
          data-collect-dnt="true"
          async
          src="https://sa.portr.dev/latest.js"
        ></script>
      </body>
    </html>
  );
}
