<script lang="ts">
  import { Link } from "svelte-routing";

  export let url: string;
</script>

<div>
  <Link let:active to={url}>
    <div
      class="flex text-[14px] items-center px-2 py-2 rounded-lg {active
        ? 'ring-0 bg-[#F4F4F5]'
        : ''}"
    >
      <slot />
    </div>
  </Link>
</div>
