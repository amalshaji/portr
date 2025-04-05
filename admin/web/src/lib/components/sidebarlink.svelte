<script lang="ts">
  import { Link } from "svelte-routing";

  export let url: string;
  export let extraClass: string = "";
</script>

<div>
  <Link let:active to={url}>
    <div
      class="flex items-center gap-3 rounded-lg px-3 py-2 transition-all hover:text-primary text-sm font-medium {active
        ? 'bg-[#f2f0f0] text-black'
        : 'text-muted-foreground'} {extraClass}"
    >
      <slot />
    </div>
  </Link>
</div>
