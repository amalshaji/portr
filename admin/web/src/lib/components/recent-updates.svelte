<script lang="ts">
  import { getContext } from "svelte";
  import { Link } from "svelte-routing";

  const team = getContext("team") as string;
</script>

<div class="p-6">
  <p class="text-lg py-2 font-semibold leading-none">Recent updates</p>

  <div class="px-6 mt-2">
    <ul class="list-disc space-y-6">
      <li class="space-y-2 space-x-4">
        <div class="space-x-4">
          <span class="font-medium underline">
            <a
              href="https://github.com/amalshaji/portr/releases/tag/0.0.17-beta"
              target="_blank"
            >
              v0.0.17-beta
            </a>
          </span>
          <span class="text-sm"
            >(admin) Added email-password based auth and GitHub auth is now
            optional. Set/change your login password <Link
              class="underline"
              to={`/${team}/my-account`}>here</Link
            >.
          </span>
        </div>
      </li>
    </ul>
  </div>
</div>
