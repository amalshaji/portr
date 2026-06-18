// Generates public/og.png (1200x630) for portr.dev social cards.
// Hand-authored SVG (tunnel motif from the Portr logo) rendered by resvg.
// Run: npm run og
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { Resvg } from "@resvg/resvg-js";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fontsDir = resolve(root, "node_modules/geist/dist/fonts");
const fontFiles = [
  "geist-sans/Geist-Medium.ttf",
  "geist-sans/Geist-SemiBold.ttf",
  "geist-sans/Geist-Bold.ttf",
  "geist-mono/GeistMono-Regular.ttf",
  "geist-mono/GeistMono-Medium.ttf",
  "geist-mono/GeistMono-SemiBold.ttf",
].map((p) => resolve(fontsDir, p));

const W = 1200;
const H = 630;
const C = {
  base: "#06090F",
  indigo: "#2A3A6E",
  teal: "#0E5E6E",
  cyan: "#22D3EE",
  text: "#EAF0F8",
  muted: "#7E8FA8",
  green: "#3FB950",
};

// tunnel rings: concentric circles around a vanishing point, brightest at the
// core (the light at the end of the tunnel), fading out toward the rim.
const cx = 1090;
const cy = 300;
const rings = [];
for (let i = 12; i >= 1; i--) {
  const r = 26 + i * i * 5.2; // quadratic spacing -> perspective depth
  const opacity = Math.max(0.04, 0.52 - i * 0.04).toFixed(3);
  const sw = (1 + (12 - i) * 0.45).toFixed(2);
  rings.push(
    `<circle cx="${cx}" cy="${cy}" r="${r.toFixed(1)}" fill="none" stroke="url(#ring)" stroke-width="${sw}" opacity="${opacity}"/>`
  );
}

const svg = `<svg width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <radialGradient id="auroraIndigo" cx="18%" cy="22%" r="70%">
      <stop offset="0%" stop-color="${C.indigo}" stop-opacity="0.85"/>
      <stop offset="60%" stop-color="${C.indigo}" stop-opacity="0"/>
    </radialGradient>
    <radialGradient id="auroraTeal" cx="90%" cy="40%" r="65%">
      <stop offset="0%" stop-color="${C.teal}" stop-opacity="0.9"/>
      <stop offset="55%" stop-color="${C.teal}" stop-opacity="0"/>
    </radialGradient>
    <radialGradient id="core" cx="50%" cy="50%" r="50%">
      <stop offset="0%" stop-color="#CFF8FF"/>
      <stop offset="35%" stop-color="${C.cyan}"/>
      <stop offset="100%" stop-color="${C.cyan}" stop-opacity="0"/>
    </radialGradient>
    <radialGradient id="ring" cx="50%" cy="50%" r="50%">
      <stop offset="0%" stop-color="#7FF0FF"/>
      <stop offset="100%" stop-color="${C.cyan}"/>
    </radialGradient>
    <radialGradient id="vignette" cx="42%" cy="46%" r="75%">
      <stop offset="55%" stop-color="#000000" stop-opacity="0"/>
      <stop offset="100%" stop-color="#000000" stop-opacity="0.55"/>
    </radialGradient>
    <linearGradient id="wordmark" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="#FFFFFF"/>
      <stop offset="100%" stop-color="#AEC4EC"/>
    </linearGradient>
    <linearGradient id="chip" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="#FFFFFF" stop-opacity="0.06"/>
      <stop offset="100%" stop-color="#FFFFFF" stop-opacity="0.02"/>
    </linearGradient>
    <filter id="glow" x="-60%" y="-60%" width="220%" height="220%">
      <feGaussianBlur stdDeviation="9"/>
    </filter>
    <filter id="coreGlow" x="-150%" y="-150%" width="400%" height="400%">
      <feGaussianBlur stdDeviation="34"/>
    </filter>
  </defs>

  <!-- background -->
  <rect width="${W}" height="${H}" fill="${C.base}"/>
  <rect width="${W}" height="${H}" fill="url(#auroraIndigo)"/>
  <rect width="${W}" height="${H}" fill="url(#auroraTeal)"/>

  <!-- tunnel -->
  <g>${rings.join("")}</g>
  <circle cx="${cx}" cy="${cy}" r="150" fill="url(#core)" opacity="0.5" filter="url(#coreGlow)"/>
  <circle cx="${cx}" cy="${cy}" r="20" fill="url(#core)"/>

  <rect width="${W}" height="${H}" fill="url(#vignette)"/>

  <!-- brand mark: three nested tunnel arches + track -->
  <g transform="translate(80,84)" stroke="${C.cyan}" fill="none" stroke-linecap="round">
    <path d="M2 40 A34 34 0 0 1 70 40" stroke-width="4" opacity="0.95"/>
    <path d="M15 40 A21 21 0 0 1 57 40" stroke-width="4" opacity="0.68"/>
    <path d="M27 40 A9 9 0 0 1 45 40" stroke-width="4" opacity="0.42"/>
    <line x1="29" y1="51" x2="22" y2="66" stroke-width="4" opacity="0.85"/>
    <line x1="43" y1="51" x2="50" y2="66" stroke-width="4" opacity="0.85"/>
  </g>

  <!-- wordmark + tagline -->
  <text x="158" y="142" font-family="Geist" font-weight="700" font-size="74" fill="url(#wordmark)" letter-spacing="-2">Portr</text>
  <text x="82" y="206" font-family="Geist" font-weight="500" font-size="30" fill="${C.muted}" letter-spacing="-0.2">Self-hosted tunnels for your team</text>

  <!-- command chip -->
  <rect x="80" y="270" width="362" height="62" rx="14" fill="url(#chip)" stroke="#FFFFFF" stroke-opacity="0.10"/>
  <text x="106" y="310" font-family="Geist Mono" font-weight="500" font-size="28">
    <tspan fill="${C.green}">$ </tspan><tspan fill="${C.text}">portr </tspan><tspan fill="${C.cyan}">http </tspan><tspan fill="${C.muted}">9000</tspan>
  </text>

  <!-- hero: public URL emerging from the tunnel -->
  <text x="80" y="446" font-family="Geist Mono" font-weight="600" font-size="56" fill="${C.cyan}" filter="url(#glow)" opacity="0.65" letter-spacing="-1.5">blue-fox.portr.dev</text>
  <text x="80" y="446" font-family="Geist Mono" font-weight="600" font-size="56" fill="#BFF6FF" letter-spacing="-1.5">blue-fox.portr.dev</text>
  <text x="82" y="494" font-family="Geist Mono" font-weight="400" font-size="26" fill="${C.muted}">
    <tspan fill="${C.cyan}">&#8627; </tspan>forwards to localhost:9000
  </text>

  <!-- footer -->
  <text x="82" y="580" font-family="Geist Mono" font-weight="400" font-size="22" fill="${C.text}" opacity="0.85">HTTP<tspan fill="${C.cyan}">  ·  </tspan>TCP<tspan fill="${C.cyan}">  ·  </tspan>WebSocket</text>
  <text x="1118" y="580" text-anchor="end" font-family="Geist Mono" font-weight="400" font-size="22" fill="${C.muted}">portr.dev</text>
</svg>`;

const png = new Resvg(svg, {
  font: { fontFiles, loadSystemFonts: false, defaultFontFamily: "Geist" },
  fitTo: { mode: "width", value: W },
}).render().asPng();

const out = resolve(root, "public/og.png");
writeFileSync(out, png);
console.log(`wrote ${out} (${(png.length / 1024).toFixed(0)} KB)`);
