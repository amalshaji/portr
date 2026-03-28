<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let value: string;
  export let class_name = "";
  export let class_list = "";

  const dispatch = createEventDispatcher();

  export function onValueChange(newValue: string) {
    value = newValue;
    dispatch("valueChange", newValue);
  }
</script>

<div class={`${class_name} ${class_list}`}>
  <slot />
</div>