import defaultMdxComponents from 'fumadocs-ui/mdx';
import { Card, Cards } from 'fumadocs-ui/components/card';
import { Callout } from 'fumadocs-ui/components/callout';
import type { MDXComponents } from 'mdx/types';
import type { ComponentProps } from 'react';
import { PortrCodeBlock } from '@/components/ui/portr-code-block';

function classes(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(' ');
}

function PortrCallout({ className, ...props }: ComponentProps<typeof Callout>) {
  return <Callout {...props} className={classes('portr-callout', className)} />;
}

function PortrCards({ className, ...props }: ComponentProps<typeof Cards>) {
  return <Cards {...props} className={classes('portr-cards', className)} />;
}

function PortrCard({ className, ...props }: ComponentProps<typeof Card>) {
  return <Card {...props} className={classes('portr-card', className)} />;
}

// use this function to get MDX components, you will need it for rendering MDX
export function getMDXComponents(components?: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    pre: PortrCodeBlock,
    Callout: PortrCallout,
    Cards: PortrCards,
    Card: PortrCard,
    ...components,
  };
}
