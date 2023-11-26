<script lang="ts">
  // @ts-expect-error
  import { Link } from "svelte-routing";

  export let url: string;
</script>

<div>
  <Link let:active to={url}>
    <div
      class="flex items-center px-3 py-2 text-gray-600 transition-colors duration-300 transform rounded-lg dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 dark:hover:text-gray-200 hover:text-gray-700 {active
        ? 'bg-gray-100 dark:bg-gray-800 dark:text-gray-200 text-gray-700'
        : ''}"
    >
      <slot />
    </div>
  </Link>
</div>
