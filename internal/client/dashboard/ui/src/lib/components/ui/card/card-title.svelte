<script lang="ts">
  export let class_name = "";
  export let class_list = "";
</script>

<h3 class={`text-lg font-semibold leading-none tracking-tight ${class_name} ${class_list}`}>
  <slot />
</h3>