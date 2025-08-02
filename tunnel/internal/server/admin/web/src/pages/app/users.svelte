<script lang="ts">
  import Members from "$lib/components/users/members.svelte";
  import * as Card from "$lib/components/ui/card";
</script>

<div class="space-y-6">
  <h1 class="text-2xl font-bold tracking-tight text-black">Team Members</h1>

  <div class="border border-gray-300 bg-white">
    <div class="p-6 border-b border-gray-300">
      <h2 class="text-xl font-semibold text-black">Team Members</h2>
      <p class="text-gray-600 mt-1">
        Manage your team members and their access levels
      </p>
    </div>
    <div class="p-6">
      <Members />
    </div>
  </div>
</div>
