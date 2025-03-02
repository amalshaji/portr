<script lang="ts">
  export let class_name = "";
  export let class_list = "";
</script>

<div class={`flex border-b ${class_name} ${class_list}`} role="tablist">
  <slot />
</div>