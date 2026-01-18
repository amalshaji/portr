import { Bug } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function IssueLink() {
  return (
    <Button variant="link" asChild className="text-black hover:text-gray-700">
      <a
        href="https://github.com/amalshaji/portr/issues/new?assignees=&labels=&projects=&template=bug_report.md&title="
        target="_blank"
        rel="noopener noreferrer"
      >
        Report an issue
        <Bug className="h-4 w-4 ml-1" />
      </a>
    </Button>
  );
}
