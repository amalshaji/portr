<script lang="ts">
  import * as Alert from "$lib/components/ui/alert";
  import { ExclamationTriangle } from "radix-icons-svelte";

  export let message: string = "";
  export let title: string = "Error";
</script>

<Alert.Root 
  variant="destructive" 
  class="border border-red-600 bg-red-50 text-red-800" 
  style="border-radius: 0;"
>
  <ExclamationTriangle class="h-4 w-4 text-red-600" />
  <Alert.Title class="text-red-800">{title}</Alert.Title>
  <Alert.Description class="text-red-700">
    {message}
  </Alert.Description>
</Alert.Root>