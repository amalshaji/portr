import {
  defineConfig,
  defineDocs,
  frontmatterSchema,
  metaSchema,
} from 'fumadocs-mdx/config';
import { z } from 'zod';

// Enhanced frontmatter schema with SEO fields following Fumadocs patterns
const enhancedFrontmatterSchema = frontmatterSchema.extend({
  keywords: z.array(z.string()).optional(),
  author: z.string().optional(),
  image: z.string().optional(),
  canonical: z.string().optional(),
  lastModified: z.string().optional(),
  preview: z.string().optional(),
  full: z.boolean().optional(),
});

// You can customise Zod schemas for frontmatter and `meta.json` here
// see https://fumadocs.vercel.app/docs/mdx/collections#define-docs
export const docs = defineDocs({
  docs: {
    schema: enhancedFrontmatterSchema,
  },
  meta: {
    schema: metaSchema,
  },
});

export default defineConfig({
  mdxOptions: {
    remarkPlugins: [],
    rehypePlugins: [],
  },
});
