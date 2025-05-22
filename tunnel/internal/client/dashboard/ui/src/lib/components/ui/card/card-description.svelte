<script lang="ts">
  export let class_name = "";
  export let class_list = "";
</script>

<p class={`text-sm text-muted-foreground ${class_name} ${class_list}`}>
  <slot />
</p>