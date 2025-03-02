<script lang="ts">
  export let class_name = "";
  export let class_list = "";
</script>

<div class={`flex items-center p-6 pt-0 ${class_name} ${class_list}`}>
  <slot />
</div>