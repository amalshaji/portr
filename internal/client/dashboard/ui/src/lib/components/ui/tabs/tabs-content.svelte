<script lang="ts">
  import { getContext } from "svelte";
  import type { Writable } from "svelte/store";

  export let value: string;
  export let class_name = "";
  export let class_list = "";

  type TabsContext = {
    value: Writable<string>;
  };

  const { value: selectedValue } = getContext<TabsContext>("tabs");

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
