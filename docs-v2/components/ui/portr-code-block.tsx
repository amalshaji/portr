"use client";

import { CodeBlock, Pre, type CodeBlockProps } from "fumadocs-ui/components/codeblock";

function classes(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export function PortrCodeBlock({ className, viewportProps, children, ...props }: CodeBlockProps) {
  return (
    <CodeBlock
      {...props}
      className={classes("portr-code-block", className)}
      viewportProps={{
        ...viewportProps,
        className: classes("portr-code-viewport", viewportProps?.className),
      }}
      Actions={({ className: actionsClassName, ...actionsProps }) => (
        <div {...actionsProps} className={classes("portr-code-actions", actionsClassName)} />
      )}
    >
      <Pre className="portr-code-pre">{children}</Pre>
    </CodeBlock>
  );
}
