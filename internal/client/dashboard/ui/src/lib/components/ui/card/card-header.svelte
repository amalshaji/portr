<script lang="ts">
  export let class_name = "";
  export let class_list = "";
</script>

<div class={`flex flex-col space-y-1.5 p-6 ${class_name} ${class_list}`}>
  <slot />
</div>