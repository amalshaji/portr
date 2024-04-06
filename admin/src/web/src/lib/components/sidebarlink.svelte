<script lang="ts">
  import { Link } from "svelte-routing";

  export let url: string;
</script>

<div>
  <Link let:active to={url}>
    <div
      class="flex items-center gap-3 rounded-lg px-3 py-2 transition-all hover:text-primary {active
        ? 'bg-muted text-primary'
        : 'text-muted-foreground'}"
    >
      <slot />
    </div>
  </Link>
</div>
