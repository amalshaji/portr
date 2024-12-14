<script lang="ts">
  import Input from "./ui/input/input.svelte";
  import Label from "./ui/label/label.svelte";

  export let data: string;

  let dataMap = {};

  data.split("&").forEach((pair) => {
    const [key, value] = pair.split("=");
    // @ts-ignore
    dataMap[key] = value;
  });
</script>

<div class="space-y-2">
  {#each Object.entries(dataMap) as [key, value]}
    <div>
      <Label for={key} class="font-normal">{decodeURIComponent(key)}</Label>
      <Input
        value={decodeURIComponent(value)}
        class="outline-none ring-0 w-1/2 overflow-auto bg-white border"
        readonly
      ></Input>
    </div>
  {/each}
</div>
