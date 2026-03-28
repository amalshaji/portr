<script lang="ts">
  export let class_name = "";
  export let class_list = "";
</script>

<div class={`rounded-lg border bg-card text-card-foreground shadow-sm ${class_name} ${class_list}`}>
  <slot />
</div>