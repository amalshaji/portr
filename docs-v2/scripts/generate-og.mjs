// Generates public/og.png (1200x630) for portr.dev social cards.
// Run: npm run og
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import satori from "satori";
import { Resvg } from "@resvg/resvg-js";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fontsDir = resolve(root, "node_modules/geist/dist/fonts");
const font = (p) => readFileSync(resolve(fontsDir, p));

// --- design tokens ------------------------------------------------------
const C = {
  bgTop: "#0B1018",
  bgBottom: "#080C14",
  border: "#1C2636",
  text: "#E7ECF4",
  muted: "#74859B",
  cyan: "#22D3EE",
  green: "#3FB950",
  indigo: "#8AA4E0",
};
const SANS = "Geist";
const MONO = "Geist Mono";

// hyperscript: satori takes React-element-shaped plain objects.
// satori requires every <div> to declare display explicitly — default to flex.
const h = (type, props = {}, ...children) => {
  const style = props.style ?? {};
  if (type === "div" && style.display === undefined) style.display = "flex";
  return { type, props: { ...props, style, children: children.flat() } };
};
const row = (style, ...c) => h("div", { style: { display: "flex", alignItems: "center", ...style } }, ...c);
const col = (style, ...c) => h("div", { style: { display: "flex", flexDirection: "column", ...style } }, ...c);
const t = (text, style) => h("div", { style: { display: "flex", ...style } }, text);

const dot = (color) => h("div", { style: { width: 13, height: 13, borderRadius: 999, background: color } });

const tree = col(
  {
    width: 1200,
    height: 630,
    backgroundImage: `linear-gradient(160deg, ${C.bgTop}, ${C.bgBottom})`,
    fontFamily: MONO,
  },
  // titlebar -------------------------------------------------------------
  row(
    {
      height: 70,
      paddingLeft: 36,
      paddingRight: 36,
      justifyContent: "space-between",
      borderBottom: `1px solid ${C.border}`,
    },
    row(
      { gap: 18 },
      row({ gap: 9 }, dot("#27313F"), dot("#27313F"), dot("#27313F")),
      t("portr", { fontFamily: SANS, fontSize: 27, fontWeight: 600, color: C.indigo, letterSpacing: -0.5 }),
      t("self-hosted tunnels for teams", { fontFamily: SANS, fontSize: 18, color: C.muted })
    ),
    t("portr.dev", { fontSize: 18, color: C.muted })
  ),
  // body -----------------------------------------------------------------
  col(
    { flex: 1, paddingLeft: 64, paddingRight: 64, justifyContent: "center", gap: 14 },
    // command
    row(
      { fontSize: 34, fontWeight: 500, gap: 14 },
      t("$", { color: C.green }),
      t("portr", { color: C.text }),
      t("http", { color: C.cyan }),
      t("9000", { color: C.muted })
    ),
    // status — green dot avoids ✓ tofu in mono
    row(
      { fontSize: 22, gap: 12, marginBottom: 22 },
      dot(C.green),
      t("tunnel established", { color: C.muted })
    ),
    // signature: public URL -> local port, on one line
    row(
      { gap: 24 },
      h("div", { style: { width: 4, height: 52, borderRadius: 4, background: C.cyan } }),
      t("https://blue-fox.portr.dev", {
        fontSize: 42,
        fontWeight: 600,
        color: C.cyan,
        letterSpacing: -1,
        whiteSpace: "nowrap",
        flexShrink: 0,
      }),
      // arrow rendered in Geist Sans, which carries U+2192
      t("→", { fontFamily: SANS, fontSize: 30, color: C.muted, flexShrink: 0 }),
      t("localhost:9000", { fontSize: 30, fontWeight: 500, color: C.text, whiteSpace: "nowrap", flexShrink: 0 })
    )
  ),
  // footer ---------------------------------------------------------------
  row(
    {
      height: 74,
      paddingLeft: 72,
      paddingRight: 72,
      justifyContent: "space-between",
      borderTop: `1px solid ${C.border}`,
      fontSize: 20,
    },
    row(
      { gap: 16, color: C.text },
      t("HTTP"),
      t("·", { color: C.cyan }),
      t("TCP"),
      t("·", { color: C.cyan }),
      t("WebSocket")
    ),
    t("inspect · replay · team access", { color: C.muted })
  )
);

const svg = await satori(tree, {
  width: 1200,
  height: 630,
  fonts: [
    { name: SANS, data: font("geist-sans/Geist-Medium.ttf"), weight: 500, style: "normal" },
    { name: SANS, data: font("geist-sans/Geist-SemiBold.ttf"), weight: 600, style: "normal" },
    { name: MONO, data: font("geist-mono/GeistMono-Regular.ttf"), weight: 400, style: "normal" },
    { name: MONO, data: font("geist-mono/GeistMono-Medium.ttf"), weight: 500, style: "normal" },
    { name: MONO, data: font("geist-mono/GeistMono-SemiBold.ttf"), weight: 600, style: "normal" },
  ],
});

const png = new Resvg(svg, { fitTo: { mode: "width", value: 1200 } }).render().asPng();
const out = resolve(root, "public/og.png");
writeFileSync(out, png);
console.log(`wrote ${out} (${(png.length / 1024).toFixed(0)} KB)`);
