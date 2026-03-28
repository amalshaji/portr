<script lang="ts">
  import { getContext } from "svelte";

  export let value: string;
  export let class_name = "";
  export let class_list = "";

  const { onValueChange, value: selectedValue } = getContext("tabs") || {};

  $: selected = $selectedValue === value;

  function handleClick() {
    if (onValueChange) {
      onValueChange(value);
    }
  }
</script>

<button
  role="tab"
  aria-selected={selected}
  data-state={selected ? "active" : "inactive"}
  class={`px-4 relative font-medium text-sm transition-all focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 data-[state=active]:text-primary data-[state=active]:border-b-2 data-[state=active]:border-primary ${class_name} ${class_list}`}
  on:click={handleClick}
>
  <slot />
</button>