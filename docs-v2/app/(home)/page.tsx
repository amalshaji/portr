import Link from "next/link";
import type { Metadata } from "next";
import {
  Terminal,
  TypingAnimation,
  AnimatedSpan,
} from "@/components/magicui/terminal";
import { GithubInfo } from "fumadocs-ui/components/github-info";
import { SparklesText } from "@/components/magicui/sparkles-text";
import Logo from "@/components/ui/logo";

export const metadata: Metadata = {
  title: "Portr - Self-Hosted Tunnel Solution for Teams",
  description:
    "Expose local HTTP, TCP, or WebSocket connections to the public internet with a self-hosted tunnel solution designed for teams. Features admin dashboard, request inspector, and team collaboration.",
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
    title: "Portr - Self-Hosted Tunnel Solution for Teams",
    description:
      "Expose local HTTP, TCP, or WebSocket connections to the public internet with a self-hosted tunnel solution designed for teams.",
    type: "website",
    url: "https://portr.dev",
    images: "/og-image",
  },
  twitter: {
    card: "summary_large_image",
    title: "Portr - Self-Hosted Tunnel Solution for Teams",
    description:
      "Expose local HTTP, TCP, or WebSocket connections to the public internet with a self-hosted tunnel solution designed for teams.",
    images: "/og-image",
  },
};

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col">
      {/* Hero Section */}
      <section className="flex flex-col justify-center text-center px-4 py-16 bg-gradient-to-b from-fd-accent/20 to-transparent">
        <div className="max-w-4xl mx-auto">
          <div className="flex items-center justify-center gap-2 mb-6">
            <Logo className="h-12 w-12 sm:h-16 sm:w-16" />
            <h1 className="text-4xl sm:text-5xl font-bold bg-gradient-to-r from-fd-foreground to-fd-muted-foreground bg-clip-text text-transparent">
              Portr
            </h1>
          </div>
          <p className="text-fd-muted-foreground text-lg sm:text-xl mb-8 max-w-2xl mx-auto leading-relaxed px-4 sm:px-0">
            Expose local HTTP, TCP, or WebSocket connections to the public
            internet with a self-hosted tunnel solution designed for teams.
          </p>

          {/* Interactive Terminal Demo */}
          <div className="w-full flex justify-center mb-8 px-4 sm:px-0">
            <div className="w-full max-w-2xl overflow-x-auto">
              <Terminal className="w-full min-w-fit mx-auto">
                <AnimatedSpan delay={0} className="flex gap-1">
                  <span className="text-green-400">$ </span>
                  <TypingAnimation delay={500}>portr http 9000</TypingAnimation>
                </AnimatedSpan>
                <AnimatedSpan delay={2000} className="flex gap-1">
                  <span className="text-blue-400">✓ </span>
                  <span className="text-gray-300">
                    Tunnel created successfully
                  </span>
                </AnimatedSpan>
                <AnimatedSpan delay={2500} className="flex gap-1">
                  <span className="text-gray-400">→ </span>
                  <span className="text-cyan-300 break-all">
                    https://abc123.portr.dev
                  </span>
                </AnimatedSpan>
                <AnimatedSpan delay={3000} className="flex gap-1">
                  <span className="text-gray-400">→ </span>
                  <span className="text-gray-300">Inspector: </span>
                  <span className="text-yellow-300 break-all">
                    http://localhost:7777
                  </span>
                </AnimatedSpan>
              </Terminal>
            </div>
          </div>

          <div className="flex flex-col md:flex-row gap-4 justify-center items-center mb-8 px-4 sm:px-0">
            <Link
              href="/docs"
              className="w-full md:w-auto px-6 sm:px-8 py-3 sm:py-4 bg-fd-primary text-fd-primary-foreground font-semibold rounded-lg hover:opacity-90 transition-opacity text-base sm:text-lg text-center"
            >
              <SparklesText className="text-base sm:text-lg" sparklesCount={3}>
                Read Documentation
              </SparklesText>
            </Link>
            <GithubInfo
              owner="amalshaji"
              repo="portr"
              className="w-full md:w-auto px-6 sm:px-8 py-3 sm:py-4 border border-fd-border text-fd-foreground font-semibold rounded-lg hover:bg-fd-accent transition-colors text-base sm:text-lg text-center"
            />
            <Link
              href="https://news.ycombinator.com/item?id=39913197"
              className="w-full md:w-auto px-6 sm:px-8 py-3 sm:py-4 border border-fd-border text-fd-foreground font-semibold rounded-lg hover:bg-fd-accent transition-colors text-base sm:text-lg flex items-center justify-center gap-2"
              target="_blank"
              rel="noopener noreferrer"
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 122.88 122.88"
                className="text-orange-500 flex-shrink-0"
              >
                <path
                  fill="#FF6600"
                  d="M18.43,0h86.02c10.18,0,18.43,8.25,18.43,18.43v86.02c0,10.18-8.25,18.43-18.43,18.43H18.43 C8.25,122.88,0,114.63,0,104.45l0-86.02C0,8.25,8.25,0,18.43,0L18.43,0z"
                />
                <polygon
                  fill="#FFFFFF"
                  points="29.76,21.84 42,21.84 61.44,60.72 80.88,21.36 93.12,21.36 66.24,70.32 66.24,102.96 56.64,102.96 56.64,70.32 29.76,21.84"
                />
              </svg>
              <span className="text-center">Hacker News • 172 ↑</span>
            </Link>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="px-4 py-16">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Key Features</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <div className="w-12 h-12 bg-fd-primary/10 rounded-lg flex items-center justify-center mb-4">
                <svg
                  className="w-6 h-6 text-fd-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold mb-2">Admin Dashboard</h3>
              <p className="text-fd-muted-foreground">
                Complete web interface to monitor active connections, manage
                teams, and control access permissions.
              </p>
            </div>

            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <div className="w-12 h-12 bg-fd-primary/10 rounded-lg flex items-center justify-center mb-4">
                <svg
                  className="w-6 h-6 text-fd-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold mb-2">Request Inspector</h3>
              <p className="text-fd-muted-foreground">
                Built-in HTTP request inspector to debug, analyze, and replay
                requests in real-time.
              </p>
            </div>

            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <div className="w-12 h-12 bg-fd-primary/10 rounded-lg flex items-center justify-center mb-4">
                <svg
                  className="w-6 h-6 text-fd-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold mb-2">Multiple Protocols</h3>
              <p className="text-fd-muted-foreground">
                Support for HTTP, TCP, and WebSocket tunnels with custom
                subdomain support.
              </p>
            </div>

            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <div className="w-12 h-12 bg-fd-primary/10 rounded-lg flex items-center justify-center mb-4">
                <svg
                  className="w-6 h-6 text-fd-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold mb-2">Team Collaboration</h3>
              <p className="text-fd-muted-foreground">
                Create teams, invite members, and share tunnel access across
                your organization.
              </p>
            </div>

            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <div className="w-12 h-12 bg-fd-primary/10 rounded-lg flex items-center justify-center mb-4">
                <svg
                  className="w-6 h-6 text-fd-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold mb-2">Self-Hosted</h3>
              <p className="text-fd-muted-foreground">
                Deploy on your own infrastructure with full control over your
                tunneling solution.
              </p>
            </div>

            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <div className="w-12 h-12 bg-fd-primary/10 rounded-lg flex items-center justify-center mb-4">
                <svg
                  className="w-6 h-6 text-fd-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13 10V3L4 14h7v7l9-11h-7z"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-semibold mb-2">Easy Setup</h3>
              <p className="text-fd-muted-foreground">
                Simple installation with homebrew, install script, or direct
                binary download.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Demo Videos Section */}
      <section className="px-4 py-16 bg-fd-accent/5">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">
            See Portr in Action
          </h2>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 sm:gap-8">
            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <h3 className="text-xl font-semibold mb-4">
                Request Inspector Demo
              </h3>
              <div className="aspect-video bg-fd-muted rounded-lg mb-4 overflow-hidden">
                <iframe
                  width="100%"
                  height="100%"
                  src="https://www.youtube.com/embed/_4tipDzuoSs"
                  title="Portr Inspector Demo"
                  frameBorder="0"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowFullScreen
                  className="rounded-lg"
                />
              </div>
              <p className="text-fd-muted-foreground text-sm">
                Learn how to create tunnels and inspect HTTP requests in
                real-time.
              </p>
            </div>

            <div className="bg-fd-card border border-fd-border rounded-lg p-6">
              <h3 className="text-xl font-semibold mb-4">
                Admin Dashboard Overview
              </h3>
              <div className="aspect-video bg-fd-muted rounded-lg mb-4 overflow-hidden">
                <iframe
                  width="100%"
                  height="100%"
                  src="https://www.youtube.com/embed/Wv5j3YQk3Ew"
                  title="Portr Admin Dashboard Demo"
                  frameBorder="0"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowFullScreen
                  className="rounded-lg"
                />
              </div>
              <p className="text-fd-muted-foreground text-sm">
                Explore the admin dashboard for team and connection management.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Getting Started Section */}
      <section className="px-4 py-16">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl font-bold mb-6">Ready to Get Started?</h2>
          <p className="text-fd-muted-foreground text-lg mb-8">
            Follow our comprehensive guides to set up Portr for your team in
            minutes.
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <Link
              href="/docs/getting-started"
              className="bg-fd-card border border-fd-border rounded-lg p-6 hover:bg-fd-accent transition-colors group"
            >
              <div className="text-2xl mb-2">📚</div>
              <h3 className="font-semibold mb-2 group-hover:text-fd-primary">
                Getting Started
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Learn the basics
              </p>
            </Link>
            <Link
              href="/docs/client/installation"
              className="bg-fd-card border border-fd-border rounded-lg p-6 hover:bg-fd-accent transition-colors group"
            >
              <div className="text-2xl mb-2">💻</div>
              <h3 className="font-semibold mb-2 group-hover:text-fd-primary">
                Install Client
              </h3>
              <p className="text-sm text-fd-muted-foreground">Set up the CLI</p>
            </Link>
            <Link
              href="/docs/server"
              className="bg-fd-card border border-fd-border rounded-lg p-6 hover:bg-fd-accent transition-colors group"
            >
              <div className="text-2xl mb-2">🚀</div>
              <h3 className="font-semibold mb-2 group-hover:text-fd-primary">
                Deploy Server
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Self-host Portr
              </p>
            </Link>
            <Link
              href="/docs/client/http-tunnel"
              className="bg-fd-card border border-fd-border rounded-lg p-6 hover:bg-fd-accent transition-colors group"
            >
              <div className="text-2xl mb-2">🔗</div>
              <h3 className="font-semibold mb-2 group-hover:text-fd-primary">
                Create Tunnels
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Start tunneling
              </p>
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}
