export function StructuredData() {
  const organizationData = {
    "@type": "Organization",
    "@context": "https://schema.org",
    name: "Portr",
    url: "https://portr.dev",
    description: "Self-hosted tunnel solution designed for teams",
    foundingDate: "2024",
    sameAs: [
      "https://github.com/amalshaji/portr",
      "https://news.ycombinator.com/item?id=39913197",
    ],
  };

  const softwareApplicationData = {
    "@type": "SoftwareApplication",
    "@context": "https://schema.org",
    name: "Portr",
    description:
      "Self-hosted tunnel solution designed for teams. Expose local HTTP, TCP, or WebSocket connections to the public internet.",
    url: "https://portr.dev",
    downloadUrl: "https://github.com/amalshaji/portr/releases",
    operatingSystem: ["Windows", "macOS", "Linux"],
    applicationCategory: "DeveloperApplication",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "USD",
    },
    aggregateRating: {
      "@type": "AggregateRating",
      ratingValue: "4.8",
      reviewCount: "172",
    },
  };

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify(organizationData),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify(softwareApplicationData),
        }}
      />
    </>
  );
}
