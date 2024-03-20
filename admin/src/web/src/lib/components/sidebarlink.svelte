<script lang="ts">
  import { Link } from "svelte-routing";

  export let url: string;
</script>

<div>
  <Link let:active to={url}>
    <div
      class="flex text-[14px] items-center px-2 py-2 rounded-lg {active
        ? 'ring-1 ring-gray-300 dark:ring-gray-600'
        : ''}"
    >
      <slot />
    </div>
  </Link>
</div>
