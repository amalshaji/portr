import "@/app/global.css";
import "@fontsource-variable/sora";
import { Provider } from "./provider";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import type { ReactNode } from "react";
import type { Metadata, Viewport } from "next";
import Script from "next/script";

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
  manifest: "/site.webmanifest",
  icons: {
    icon: [{ url: "/favicon.svg", type: "image/svg+xml" }],
    shortcut: "/favicon.svg",
    apple: [{ url: "/apple-touch-icon.png", sizes: "180x180", type: "image/png" }],
  },
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
  width: "device-width",
  initialScale: 1,
  themeColor: "#02142a",
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      suppressHydrationWarning
    >
      <body className="flex min-h-screen flex-col">
        <Provider>{children}</Provider>
        {/* 100% privacy-first analytics */}
        <Script
          data-collect-dnt="true"
          async
          src="https://sa.portr.dev/latest.js"
        />
      </body>
    </html>
  );
}
