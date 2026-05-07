<script lang="ts">
  import { setContext } from "svelte";
  import { createEventDispatcher } from "svelte";
  import { writable } from "svelte/store";

  export let value: string;
  export let class_name = "";
  export let class_list = "";

  const dispatch = createEventDispatcher();
  const selectedValue = writable(value);

  setContext("tabs", {
    value: selectedValue,
    onValueChange,
  });

  $: selectedValue.set(value);

  export function onValueChange(newValue: string) {
    value = newValue;
    selectedValue.set(newValue);
    dispatch("valueChange", newValue);
  }
</script>

<div class={`${class_name} ${class_list}`}>
  <slot />
</div>
