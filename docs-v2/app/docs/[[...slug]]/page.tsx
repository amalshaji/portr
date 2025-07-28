import { source } from "@/lib/source";
import {
  DocsPage,
  DocsBody,
  DocsDescription,
  DocsTitle,
} from "fumadocs-ui/page";
import { notFound } from "next/navigation";
import { createRelativeLink } from "fumadocs-ui/mdx";
import { getMDXComponents } from "@/mdx-components";

export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const MDXContent = page.data.body;

  return (
    <DocsPage toc={page.data.toc} full={page.data.full}>
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription>{page.data.description}</DocsDescription>
      <DocsBody>
        <MDXContent
          components={getMDXComponents({
            // this allows you to link to other pages with relative file paths
            a: createRelativeLink(source, page),
          })}
        />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const { title, description } = page.data;
  const url = `https://portr.dev${page.url}`;

  // Generate Open Graph image URL using Fumadocs convention
  const image = ["/docs-og", ...(params.slug || []), "image.png"].join("/");

  return {
    title,
    description,
    keywords: page.data.keywords || [
      "portr",
      "tunnel",
      "self-hosted",
      "documentation",
    ],
    authors: [{ name: page.data.author || "Portr Team" }],
    openGraph: {
      title,
      description: description || "Portr documentation",
      type: "article",
      url,
      siteName: "Portr Documentation",
      images: image,
    },
    twitter: {
      card: "summary_large_image",
      title,
      description: description || "Portr documentation",
      images: image,
    },
    alternates: {
      canonical: page.data.canonical || url,
    },
  };
}
