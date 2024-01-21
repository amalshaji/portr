<script lang="ts">
  import { Link } from "svelte-routing";

  export let url: string;
</script>

<div>
  <Link let:active to={url}>
    <div
      class="flex text-[14px] items-center px-2 py-2 text-gray-600 rounded-lg dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 {active
        ? 'bg-gray-100 dark:bg-gray-800 dark:text-gray-200 text-gray-900'
        : ''}"
    >
      <slot />
    </div>
  </Link>
</div>
