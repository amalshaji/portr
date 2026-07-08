// Generates public/og.png (1200x630) for portr.dev social cards.
// Satori builds the terminal, the protocol orbs and text; the glowing
// connector curves come from one embedded SVG layer. Rendered with resvg.
// Run: npm run og
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import satori from "satori";
import { Resvg } from "@resvg/resvg-js";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fontsDir = resolve(root, "node_modules/geist/dist/fonts");
const font = (p) => readFileSync(resolve(fontsDir, p));

const W = 1200;
const H = 630;
const C = {
  base: "#070A10",
  cyan: "#22D3EE",
  cyanBright: "#7FEAF8",
  dot: "#39465A",
  termBg: "rgba(16,23,35,0.55)",
  termBorder: "rgba(125,160,200,0.16)",
};
const SANS = "Geist";
const MONO = "Geist Mono";
const CYAN_RGB = "34,211,238"; // C.cyan as rgb, for translucent glows
const glow = (a) => `0 0 18px rgba(${CYAN_RGB},${a})`;

// hyperscript — satori takes React-element-shaped plain objects.
const h = (type, props = {}, ...children) => {
  const style = props.style ?? {};
  if (type === "div" && style.display === undefined) style.display = "flex";
  return { type, props: { ...props, style, children: children.flat() } };
};
const t = (text, style) => h("div", { style: { display: "flex", ...style } }, text);
const dot = () => h("div", { style: { width: 13, height: 13, borderRadius: 999, background: C.dot } });

// --- orbs ---------------------------------------------------------------
const ORBS = [
  { label: "HTTP", cx: 1015, cy: 150, d: 132, fs: 30 },
  { label: "TCP", cx: 1092, cy: 330, d: 120, fs: 30 },
  { label: "WEBSOCKET", cx: 1010, cy: 500, d: 132, fs: 17 },
];
const orb = ({ label, cx, cy, d, fs }) =>
  h(
    "div",
    {
      style: {
        position: "absolute",
        left: cx - d / 2,
        top: cy - d / 2,
        width: d,
        height: d,
        borderRadius: 999,
        border: `3px solid ${C.cyan}`,
        boxShadow: `0 0 26px rgba(${CYAN_RGB},0.45), inset 0 0 20px rgba(${CYAN_RGB},0.12)`,
        alignItems: "center",
        justifyContent: "center",
      },
    },
    t(label, { fontFamily: SANS, fontWeight: 700, fontSize: fs, color: C.cyanBright, letterSpacing: 1, textShadow: glow(0.55) })
  );

// terminal geometry — shared so the connector start can't drift from the box
const TERM = { left: 150, top: 176, w: 470, h: 286 };
const EX = TERM.left + TERM.w; // connectors exit the terminal's right edge

// --- connector curves (glowing) -----------------------------------------
// start x is derived (EX); control points + orb-side endpoints are tuned to ORBS
const PATHS = [
  `M${EX} 304 C 800 304 820 198 960 180`,
  `M${EX} 322 C 815 356 905 330 1033 322`,
  `M${EX} 332 C 800 360 768 470 957 460`,
];
const stroke = (d, w, o, color) =>
  `<path d="${d}" fill="none" stroke="${color}" stroke-width="${w}" stroke-linecap="round" opacity="${o}"/>`;
const connectorSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">${PATHS.map(
  (d) =>
    stroke(d, 16, 0.06, C.cyan) +
    stroke(d, 9, 0.13, C.cyan) +
    stroke(d, 5, 0.32, C.cyan) +
    stroke(d, 2.5, 1, C.cyanBright)
).join("")}</svg>`;
const connectorUri = `data:image/svg+xml;utf8,${encodeURIComponent(connectorSvg)}`;

// --- terminal -----------------------------------------------------------
const terminal = h(
  "div",
  {
    style: {
      position: "absolute",
      left: TERM.left,
      top: TERM.top,
      width: TERM.w,
      height: TERM.h,
      borderRadius: 16,
      background: C.termBg,
      border: `1px solid ${C.termBorder}`,
      boxShadow: "0 40px 90px rgba(0,0,0,0.5)",
      flexDirection: "column",
      overflow: "hidden",
    },
  },
  // titlebar
  h(
    "div",
    {
      style: {
        height: 54,
        alignItems: "center",
        paddingLeft: 22,
        gap: 11,
        borderBottom: `1px solid rgba(125,160,200,0.12)`,
      },
    },
    dot(),
    dot(),
    dot()
  ),
  // body
  h(
    "div",
    {
      style: {
        flex: 1,
        flexDirection: "column",
        justifyContent: "center",
        paddingLeft: 40,
        gap: 16,
        fontFamily: MONO,
        fontSize: 36,
        fontWeight: 500,
        color: C.cyan,
      },
    },
    t("$ portr http 9000", { textShadow: glow(0.45), whiteSpace: "pre" }),
    t("$ portr tcp  9000", { textShadow: glow(0.45), whiteSpace: "pre" })
  )
);

// --- root ---------------------------------------------------------------
const tree = h(
  "div",
  {
    style: {
      position: "relative",
      width: W,
      height: H,
      backgroundColor: C.base,
      backgroundImage: "radial-gradient(120% 130% at 34% 40%, #141C2A 0%, #0A0E16 46%, #06090E 100%)",
    },
  },
  h("img", { width: W, height: H, src: connectorUri, style: { position: "absolute", left: 0, top: 0 } }),
  terminal,
  ...ORBS.map(orb)
);

const svg = await satori(tree, {
  width: W,
  height: H,
  fonts: [
    { name: SANS, data: font("geist-sans/Geist-Bold.ttf"), weight: 700, style: "normal" },
    { name: MONO, data: font("geist-mono/GeistMono-Medium.ttf"), weight: 500, style: "normal" },
  ],
});

const png = new Resvg(svg, { fitTo: { mode: "width", value: W } }).render().asPng();
const out = resolve(root, "public/og.png");
writeFileSync(out, png);
console.log(`wrote ${out} (${(png.length / 1024).toFixed(0)} KB)`);
