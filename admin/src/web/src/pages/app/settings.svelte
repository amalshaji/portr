<script lang="ts">
  import { Input } from "$lib/components/ui/input";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { currentUser } from "$lib/store";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";
  import { getContext } from "svelte";
  import { copyCodeToClipboard } from "$lib/utils";

  let team = getContext("team") as string;

  let isRotatingSecretKey = false;

  const rotateSecretKey = async () => {
    isRotatingSecretKey = true;
    try {
      const res = await fetch(`/api/v1/user/me/rotate-secret-key`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
      });
      if (res.ok) {
        const secret_key = (await res.json()).secret_key;
        // @ts-ignore
        $currentUser = {
          ...$currentUser,
          secret_key: secret_key,
        };
        toast.success("New secret key generated");
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
    } finally {
      isRotatingSecretKey = false;
    }
  };
</script>

<div class="w-full">
  <Card.Root class="rounded-sm border-none shadow-none w-full">
    <Card.Header class="space-y-3">
      <Card.Title class="text-lg">Secret key</Card.Title>
      <Card.Description
        >The secret key to authenticate client connection</Card.Description
      >
    </Card.Header>
    <Card.Content class="space-y-2 flex items-center w-1/2">
      <Input
        type="text"
        readonly
        value={$currentUser?.secret_key}
        on:click={() => copyCodeToClipboard($currentUser?.secret_key)}
      />
    </Card.Content>
    <Card.Footer>
      <Button
        variant="outline"
        on:click={rotateSecretKey}
        disabled={isRotatingSecretKey}
      >
        {#if isRotatingSecretKey}
          <Reload class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Rotate key
      </Button>
    </Card.Footer>
  </Card.Root>
</div>
