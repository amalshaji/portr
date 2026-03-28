<script lang="ts">
  import { currentRequest } from "$lib/store";
  import { onMount } from "svelte";
  import Button from "./ui/button/button.svelte";
  import Input from "./ui/input/input.svelte";
  import Label from "./ui/label/label.svelte";

  let formData,
    data = {};

  const getResponse = async () => {
    const response = await fetch(
      `/api/tunnels/render/${$currentRequest?.ID}?type=request`
    );
    return await response.formData();
  };

  const getFileUrl = async (file: File) => {
    console.log(file.type);
    const blob = new Blob([await file.arrayBuffer()]);
    return URL.createObjectURL(blob);
  };

  onMount(async () => {
    formData = await getResponse();
    for (const [key, value] of formData.entries()) {
      // @ts-ignore
      data[key] = value;
    }
  });
</script>

{#if data}
  <div class="space-y-2 p-4">
    {#each Object.entries(data) as [key, value]}
      <div>
        <Label for="email" class="font-normal">{key}</Label> <br />
        {#if value instanceof File}
          {#if value.type.startsWith("image/svg+xml")}
            <img src={URL.createObjectURL(value)} class="h-16" alt={key} />
          {:else}
            {#await getFileUrl(value) then url}
              {#if value.type.startsWith("image/")}
                <img src={url} alt={key} class="h-36" />
              {:else}
                <Button href={url} download={value.name}>
                  Download {value.name}
                </Button>
              {/if}
            {/await}
          {/if}
        {:else}
          <Input
            {value}
            class="outline-none ring-0 w-1/2 overflow-auto bg-white border"
            readonly
          ></Input>
        {/if}
      </div>
    {/each}
  </div>
{/if}
