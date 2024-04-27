import starlightPlugin from "@astrojs/starlight-tailwind";
import colors from "tailwindcss/colors";

const accent = {
  200: "#dfc0bb",
  600: "#a15046",
  900: "#4a2722",
  950: "#341d1a",
};
const gray = {
  100: "#f6f6f6",
  200: "#eeeeee",
  300: "#c2c2c2",
  400: "#8b8b8b",
  500: "#585858",
  700: "#383838",
  800: "#272727",
  900: "#181818",
};

/** @type {import('tailwindcss').Config} */
export default {
  content: ["./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}"],
  theme: {
    extend: {
      colors: {
        accent,
        gray,
      },
      fontFamily: {
        sans: ["Geist Sans"],
        mono: ["Geist Mono"],
      },
    },
  },
  plugins: [starlightPlugin()],
};
