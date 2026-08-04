import { MessageCircleQuestion } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function IssueLink() {
  return (
    <Button
      variant="ghost"
      size="sm"
      asChild
      className="rounded-full px-3 text-muted-foreground hover:text-foreground"
    >
      <a
        href="https://github.com/amalshaji/portr/issues/new?assignees=&labels=&projects=&template=bug_report.md&title="
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Report an issue"
      >
        <MessageCircleQuestion className="size-4" />
        <span className="hidden sm:inline">Report an issue</span>
      </a>
    </Button>
  );
}
