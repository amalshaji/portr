import { generateOGImage } from "fumadocs-ui/og";

export async function GET() {
  return generateOGImage({
    title: "Portr",
    description: "Self-hosted tunnel solution designed for teams",
    site: "Portr",
    primaryTextColor: "rgb(0, 0, 0)",
    primaryColor: "rgb(71, 85, 105)", // slate-600
  });
}
