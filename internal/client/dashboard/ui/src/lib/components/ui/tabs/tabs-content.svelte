<script lang="ts">
  import { getContext } from "svelte";

  export let value: string;
  export let class_name = "";
  export let class_list = "";

  const { value: selectedValue } = getContext("tabs") || {};

  $: selected = $selectedValue === value;
</script>

{#if selected}
  <div
    role="tabpanel"
    data-state={selected ? "active" : "inactive"}
    class={`${class_name} ${class_list}`}
  >
    <slot />
  </div>
{/if}