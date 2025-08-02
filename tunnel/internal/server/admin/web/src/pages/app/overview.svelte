<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { setupScript } from "$lib/store";
  import { copyCodeToClipboard } from "$lib/utils";
  import {
    Copy,
    Globe,
    Server,
    Shield,
    Terminal,
    Users,
    Cpu,
    HardDrive,
  } from "lucide-svelte";
  import { getContext, onDestroy, onMount } from "svelte";
  import Highlight from "svelte-highlight";
  import bash from "svelte-highlight/languages/bash";
  import "svelte-highlight/styles/atom-one-light.css";

  const helpCommand = "portr -h";

  let team = getContext("team") as string;
  let statsLoading = true;
  let serverStartTime: string | null = null;
  let uptimeDisplay = "...";
  let uptimeInterval: ReturnType<typeof setInterval>;
  let statsPollingInterval: ReturnType<typeof setInterval>;

  let dashboardStats = {
    activeConnections: 0,
    totalUsers: 0,
  };

  let systemStats = {
    memoryUsedMB: 0,
    memoryTotalMB: 0,
    systemMemoryUsedGB: 0,
    systemMemoryTotalGB: 0,
    systemMemoryUsagePercent: 0,
    cpuUsagePercent: 0,
    numCpu: 0,
    goroutines: 0,
    hostname: "",
    os: "",
    architecture: "",
  };

  const getSetupScript = async () => {
    const res = await fetch("/api/v1/config/setup-script", {
      headers: {
        "x-team-slug": team,
      },
    });
    setupScript.set((await res.json())["message"]);
  };

  const formatUptime = (startTimeStr: string) => {
    const startTime = new Date(startTimeStr);
    const now = new Date();
    const uptimeMs = now.getTime() - startTime.getTime();

    const seconds = Math.floor(uptimeMs / 1000) % 60;
    const minutes = Math.floor(uptimeMs / (1000 * 60)) % 60;
    const hours = Math.floor(uptimeMs / (1000 * 60 * 60)) % 24;
    const days = Math.floor(uptimeMs / (1000 * 60 * 60 * 24));

    return `${days}d ${hours}h ${minutes}m ${seconds}s`;
  };

  const updateUptime = () => {
    if (serverStartTime) {
      uptimeDisplay = formatUptime(serverStartTime);
    }
  };

  const getDashboardStats = async (showLoading = true) => {
    if (showLoading) {
      statsLoading = true;
    }
    try {
      const res = await fetch("/api/v1/config/stats", {
        headers: {
          "x-team-slug": team,
        },
      });

      if (res.ok) {
        const data = await res.json();
        const teamStats = data.team_stats;
        const sysStats = data.system_stats;

        dashboardStats = {
          activeConnections: teamStats.active_connections,
          totalUsers: teamStats.team_members,
        };

        // Update system stats
        systemStats = {
          memoryUsedMB: sysStats.memory_used_mb || 0,
          memoryTotalMB: sysStats.memory_total_mb || 0,
          systemMemoryUsedGB: sysStats.system_memory_used_gb || 0,
          systemMemoryTotalGB: sysStats.system_memory_total_gb || 0,
          systemMemoryUsagePercent: sysStats.system_memory_usage_percent || 0,
          cpuUsagePercent: sysStats.cpu_usage_percent || 0,
          numCpu: sysStats.num_cpu || 0,
          goroutines: sysStats.goroutines || 0,
          hostname: sysStats.hostname || "",
          os: sysStats.os || "",
          architecture: sysStats.architecture || "",
        };

        // Get server start time from system_stats (only set it once)
        if (sysStats && sysStats.server_start_time && !serverStartTime) {
          serverStartTime = sysStats.server_start_time;
          // Initialize uptime display
          updateUptime();
        }
      }
    } catch (error) {
      console.error("Failed to fetch stats:", error);
      // Only reset stats on initial load failure, not polling failures
      if (showLoading) {
        // Fall back to empty stats
        dashboardStats = {
          activeConnections: 0,
          totalUsers: 0,
        };
        systemStats = {
          memoryUsedMB: 0,
          memoryTotalMB: 0,
          systemMemoryUsedGB: 0,
          systemMemoryTotalGB: 0,
          systemMemoryUsagePercent: 0,
          cpuUsagePercent: 0,
          numCpu: 0,
          goroutines: 0,
          hostname: "",
          os: "",
          architecture: "",
        };
        uptimeDisplay = "Unknown";
      }
    } finally {
      if (showLoading) {
        statsLoading = false;
      }
    }
  };

  const installCommand = `
  curl -sSf https://install.portr.dev | sh
  `.trim();

  const homebrewCommand = `
  brew install amalshaji/taps/portr
  `.trim();

  const handleCopy = (text: string) => {
    copyCodeToClipboard(text);
  };

  onMount(() => {
    getSetupScript();
    getDashboardStats(true); // Show loading on initial load

    // Set up polling interval to refresh stats every 5 seconds
    statsPollingInterval = setInterval(() => {
      getDashboardStats(false); // Don't show loading on polling updates
    }, 5000);

    // Set up uptime interval to update every second
    uptimeInterval = setInterval(updateUptime, 1000);
  });

  onDestroy(() => {
    if (uptimeInterval) {
      clearInterval(uptimeInterval);
    }
    if (statsPollingInterval) {
      clearInterval(statsPollingInterval);
    }
  });
</script>

<svelte:head>
  <style>
    @font-face {
      font-family: "Geist Mono";
      src: url("/static/geist-mono-latin-400-normal.woff2") format("woff2");
    }
  </style>
</svelte:head>

<div class="space-y-8">
  <!-- Dashboard Header -->
  <div class="flex justify-between items-center">
    <div>
      <h1 class="text-2xl font-bold tracking-tight text-black">Dashboard</h1>
      <p class="text-gray-600">Welcome to your {team} dashboard.</p>
    </div>
    <Button
      variant="outline"
      class="flex items-center gap-2 border border-gray-400 bg-white text-black hover:bg-gray-50 focus:outline-none focus:ring-0"
      style="border-radius: 0;"
      href="https://portr.dev"
      target="_blank"
    >
      <Terminal class="h-4 w-4" />
      Documentation
    </Button>
  </div>

  <!-- Stats Cards -->
  <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
    <div class="border border-gray-300 bg-white">
      <div class="p-6">
        <div class="flex items-center justify-between space-y-0 pb-2">
          <p class="text-sm font-medium text-black">Active Connections</p>
          <div
            class="h-8 w-8 border border-gray-300 bg-gray-100 flex items-center justify-center"
          >
            <Globe class="h-4 w-4 text-black" />
          </div>
        </div>
        <div class="flex items-center pt-3">
          <div class="text-2xl font-bold text-black">
            {statsLoading ? "..." : dashboardStats.activeConnections}
          </div>
        </div>
      </div>
    </div>

    <div class="border border-gray-300 bg-white">
      <div class="p-6">
        <div class="flex items-center justify-between space-y-0 pb-2">
          <p class="text-sm font-medium text-black">Team Members</p>
          <div
            class="h-8 w-8 border border-gray-300 bg-gray-100 flex items-center justify-center"
          >
            <Users class="h-4 w-4 text-black" />
          </div>
        </div>
        <div class="pt-3">
          <div class="text-2xl font-bold text-black">
            {statsLoading ? "..." : dashboardStats.totalUsers}
          </div>
        </div>
      </div>
    </div>

    <div class="border border-gray-300 bg-white">
      <div class="p-6">
        <div class="flex items-center justify-between space-y-0 pb-2">
          <p class="text-sm font-medium text-black">Server Uptime</p>
          <div
            class="h-8 w-8 border border-gray-300 bg-gray-100 flex items-center justify-center"
          >
            <Shield class="h-4 w-4 text-black" />
          </div>
        </div>
        <div class="pt-3">
          <div class="text-2xl font-bold text-black">
            {uptimeDisplay}
          </div>
        </div>
      </div>
    </div>

    <div class="border border-gray-300 bg-white">
      <div class="p-6">
        <div class="flex items-center justify-between space-y-0 pb-2">
          <p class="text-sm font-medium text-black">Memory Usage</p>
          <div
            class="h-8 w-8 border border-gray-300 bg-gray-100 flex items-center justify-center"
          >
            <HardDrive class="h-4 w-4 text-black" />
          </div>
        </div>
        <div class="pt-3">
          <div class="text-2xl font-bold text-black">
            {statsLoading
              ? "..."
              : `${systemStats.systemMemoryUsagePercent.toFixed(1)}%`}
          </div>
          <p class="text-xs text-gray-600 mt-1">
            {statsLoading
              ? ""
              : `${systemStats.systemMemoryUsedGB.toFixed(1)}GB / ${systemStats.systemMemoryTotalGB.toFixed(1)}GB`}
          </p>
        </div>
      </div>
    </div>

    <div class="border border-gray-300 bg-white">
      <div class="p-6">
        <div class="flex items-center justify-between space-y-0 pb-2">
          <p class="text-sm font-medium text-black">CPU Usage</p>
          <div
            class="h-8 w-8 border border-gray-300 bg-gray-100 flex items-center justify-center"
          >
            <Cpu class="h-4 w-4 text-black" />
          </div>
        </div>
        <div class="pt-3">
          <div class="text-2xl font-bold text-black">
            {statsLoading
              ? "..."
              : `${systemStats.cpuUsagePercent.toFixed(1)}%`}
          </div>
          <p class="text-xs text-gray-600 mt-1">
            {statsLoading ? "" : `${systemStats.numCpu} cores`}
          </p>
        </div>
      </div>
    </div>
  </div>

  <!-- System Information -->
  <div class="border border-gray-300 bg-white">
    <div class="p-6">
      <div class="mb-6">
        <h2 class="text-xl font-semibold text-black">System Information</h2>
        <p class="text-gray-600 mt-1">Server hardware and runtime details</p>
      </div>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">Hostname</p>
          <p class="text-sm">
            {statsLoading ? "..." : systemStats.hostname || "Unknown"}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">
            Operating System
          </p>
          <p class="text-sm">
            {statsLoading ? "..." : systemStats.os || "Unknown"}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">Architecture</p>
          <p class="text-sm">
            {statsLoading ? "..." : systemStats.architecture || "Unknown"}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">CPU Cores</p>
          <p class="text-sm">
            {statsLoading ? "..." : `${systemStats.numCpu} cores`}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">CPU Usage</p>
          <p class="text-sm">
            {statsLoading
              ? "..."
              : `${systemStats.cpuUsagePercent.toFixed(2)}%`}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">
            Total System Memory
          </p>
          <p class="text-sm">
            {statsLoading
              ? "..."
              : `${systemStats.systemMemoryTotalGB.toFixed(2)} GB`}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">
            Application Memory Usage
          </p>
          <p class="text-sm">
            {statsLoading ? "..." : `${systemStats.memoryUsedMB.toFixed(1)} MB`}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">
            Application Memory Pool
          </p>
          <p class="text-sm">
            {statsLoading
              ? "..."
              : `${systemStats.memoryTotalMB.toFixed(1)} MB`}
          </p>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium text-muted-foreground">
            Active Go Routines
          </p>
          <p class="text-sm">
            {statsLoading ? "..." : systemStats.goroutines.toLocaleString()}
          </p>
        </div>
      </div>
    </div>
  </div>

  <!-- Client Setup Section -->
  <div class="border border-gray-300 bg-white">
    <div class="p-6">
      <div class="mb-6">
        <h2 class="text-xl font-semibold text-black">Client Setup</h2>
        <p class="text-gray-600 mt-1">
          Follow these steps to set up and configure the portr client
        </p>
      </div>
      <div class="space-y-6">
      <div class="bg-gray-50 border border-gray-300 p-6">
        <h3 class="text-sm font-medium mb-3 flex items-center gap-2 text-black">
          <span
            class="flex h-6 w-6 border border-gray-400 bg-white items-center justify-center text-xs font-semibold text-black"
            >1</span
          >
          Install the portr client
        </h3>

        <div class="space-y-4">
          <div>
            <p class="text-sm text-gray-600 mb-2">Using the install script:</p>
            <div class="relative group">
              <Highlight
                language={bash}
                code={installCommand}
                class="border border-gray-400 text-sm my-2 overflow-hidden"
              />
              <button
                class="absolute right-2 top-2 p-1 bg-white border border-gray-400 opacity-0 group-hover:opacity-100 transition-opacity"
                on:click={() => handleCopy(installCommand)}
              >
                <Copy class="h-4 w-4 text-black" />
              </button>
            </div>
          </div>

          <div>
            <p class="text-sm text-gray-600 mb-2">Or using homebrew:</p>
            <div class="relative group">
              <Highlight
                language={bash}
                code={homebrewCommand}
                class="border border-gray-400 text-sm my-2 overflow-hidden"
              />
              <button
                class="absolute right-2 top-2 p-1 bg-white border border-gray-400 opacity-0 group-hover:opacity-100 transition-opacity"
                on:click={() => handleCopy(homebrewCommand)}
              >
                <Copy class="h-4 w-4 text-black" />
              </button>
            </div>
          </div>
        </div>

        <p class="mt-4 text-sm text-gray-600">
          You can also download the binary from the <a
            href="https://github.com/amalshaji/portr/releases"
            target="_blank"
            class="text-black hover:underline font-medium">GitHub releases</a
          >
        </p>
      </div>

      <div class="bg-gray-50 border border-gray-300 p-6">
        <h3 class="text-sm font-medium mb-3 flex items-center gap-2 text-black">
          <span
            class="flex h-6 w-6 border border-gray-400 bg-white items-center justify-center text-xs font-semibold text-black"
            >2</span
          >
          Run the following command to set up portr client auth
        </h3>

        <div class="relative group">
          <Highlight
            language={bash}
            code={$setupScript}
            class="border border-gray-400 text-sm my-2 overflow-hidden"
          />
          <button
            class="absolute right-2 top-2 p-1 bg-white border border-gray-400 opacity-0 group-hover:opacity-100 transition-opacity"
            on:click={() => handleCopy($setupScript)}
          >
            <Copy class="h-4 w-4 text-black" />
          </button>
        </div>

        <p class="mt-4 text-sm text-gray-600">
          Note: use <code class="bg-gray-100 px-1 py-0.5 rounded-sm"
            >./portr</code
          >
          instead of
          <code class="bg-gray-100 px-1 py-0.5 rounded-sm">portr</code>
          if the binary is in the same folder and not set in
          <code class="bg-gray-100 px-1 py-0.5 rounded-sm">$PATH</code>
        </p>
      </div>

      <div class="bg-gray-50 border border-gray-300 p-6">
        <h3 class="text-sm font-medium mb-3 flex items-center gap-2 text-black">
          <span
            class="flex h-6 w-6 border border-gray-400 bg-white items-center justify-center text-xs font-semibold text-black"
            >3</span
          >
          You're ready to use the tunnel
        </h3>

        <p class="text-gray-600">
          Run <code class="bg-gray-100 px-1 py-0.5 rounded-sm"
            >{helpCommand}</code
          >
          or check out the
          <a
            href="https://portr.dev"
            target="_blank"
            class="text-black hover:underline font-medium"
          >
            client documentation
          </a> for more information.
        </p>
      </div>
      </div>
    </div>
  </div>
</div>
