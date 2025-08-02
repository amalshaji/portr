import { generateOGImage } from "fumadocs-ui/og";
import { source } from "@/lib/source";
import { notFound } from "next/navigation";

export const dynamic = 'force-static';

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ slug: string[] }> }
) {
  const { slug } = await params;
  const page = source.getPage(slug.slice(0, -1));

  if (!page) notFound();

  return generateOGImage({
    title: page.data.title,
    description: page.data.description,
    site: "Portr",
    primaryTextColor: "rgb(0, 0, 0)",
    primaryColor: "rgb(71, 85, 105)", // slate-600
  });
}

export function generateStaticParams() {
  return source.generateParams().map((page) => ({
    ...page,
    slug: [...page.slug, "image.png"],
  }));
}
