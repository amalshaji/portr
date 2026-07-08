"use client";

import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

const starsEndpoint = "https://github-stars.amalshaji.workers.dev/";
const repository = "amalshaji/portr";

type GitHubStarsResponse = Record<string, { stars?: unknown }>;

export function GitHubStarsLink({ className }: { className?: string }) {
  const [stars, setStars] = useState<number | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    async function loadStars() {
      try {
        const response = await fetch(starsEndpoint, {
          cache: "force-cache",
          credentials: "omit",
          signal: controller.signal,
        });
        if (!response.ok) return;

        const data = (await response.json()) as GitHubStarsResponse;
        const value = data[repository]?.stars;
        if (
          typeof value === "number" &&
          Number.isSafeInteger(value) &&
          value >= 0
        ) {
          setStars(value);
        }
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
      }
    }

    void loadStars();
    return () => controller.abort();
  }, []);

  return (
    <a
      href="https://github.com/amalshaji/portr"
      target="_blank"
      rel="noopener noreferrer"
      className={cn("inline-flex items-center justify-center gap-2", className)}
    >
      <svg
        viewBox="0 0 24 24"
        width="18"
        height="18"
        aria-hidden="true"
        className="shrink-0"
      >
        <path
          fill="currentColor"
          d="M12 .7a11.5 11.5 0 0 0-3.64 22.4c.58.1.79-.25.79-.56v-2.24c-3.22.7-3.9-1.37-3.9-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.04 1.77 2.72 1.26 3.38.96.1-.75.4-1.26.74-1.55-2.57-.29-5.27-1.28-5.27-5.68 0-1.26.45-2.28 1.19-3.08-.12-.29-.52-1.46.11-3.04 0 0 .97-.31 3.16 1.18a10.9 10.9 0 0 1 5.76 0c2.2-1.49 3.16-1.18 3.16-1.18.63 1.58.23 2.75.11 3.04.74.8 1.19 1.82 1.19 3.08 0 4.41-2.71 5.38-5.29 5.67.42.36.79 1.06.79 2.14v3.18c0 .31.21.67.8.56A11.5 11.5 0 0 0 12 .7Z"
        />
      </svg>
      <span>GitHub</span>
      {stars !== null && (
        <span
          aria-label={`${stars.toLocaleString()} GitHub stars`}
          aria-live="polite"
          className="tabular-nums text-fd-muted-foreground"
        >
          ★ {stars.toLocaleString()}
        </span>
      )}
    </a>
  );
}
