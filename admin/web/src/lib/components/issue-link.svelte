<script>
  import { Button } from "$lib/components/ui/button";
  import { Bug } from "lucide-svelte";
</script>

<Button
  href="https://github.com/amalshaji/portr/issues/new?assignees=&labels=&projects=&template=bug_report.md&title="
  variant="link"
  target="_blank"
>
  Report a bug
  <Bug class="h-4 w-4 ml-1" />
</Button>
