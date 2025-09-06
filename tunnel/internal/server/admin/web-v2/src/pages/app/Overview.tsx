import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import {
  Copy,
  Globe,
  Shield,
  Terminal,
  Users,
  Cpu,
  HardDrive,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { copyCodeToClipboard } from "@/lib/utils";
import type { DashboardStats, SystemStats } from "@/types";

export default function Overview() {
  const { team } = useParams<{ team: string }>();
  const [statsLoading, setStatsLoading] = useState(true);
  const [serverStartTime, setServerStartTime] = useState<string | null>(null);
  const [uptimeDisplay, setUptimeDisplay] = useState("...");
  const [setupScript, setSetupScript] = useState("");

  const [dashboardStats, setDashboardStats] = useState<DashboardStats>({
    activeConnections: 0,
    totalUsers: 0,
  });

  const [systemStats, setSystemStats] = useState<SystemStats>({
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
  });

  const getSetupScript = async () => {
    if (!team) return;
    try {
      const res = await fetch("/api/v1/config/setup-script", {
        headers: {
          "x-team-slug": team,
        },
      });
      const data = await res.json();
      setSetupScript(data.message || "");
    } catch (error) {
      console.error("Failed to fetch setup script:", error);
    }
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
      setUptimeDisplay(formatUptime(serverStartTime));
    }
  };

  const getDashboardStats = async (showLoading = true) => {
    if (!team) return;
    if (showLoading) {
      setStatsLoading(true);
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

        setDashboardStats({
          activeConnections: teamStats.active_connections,
          totalUsers: teamStats.team_members,
        });

        setSystemStats({
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
        });

        if (sysStats && sysStats.server_start_time && !serverStartTime) {
          setServerStartTime(sysStats.server_start_time);
          setUptimeDisplay(formatUptime(sysStats.server_start_time));
        }
      }
    } catch (error) {
      console.error("Failed to fetch stats:", error);
      if (showLoading) {
        setDashboardStats({
          activeConnections: 0,
          totalUsers: 0,
        });
        setSystemStats({
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
        });
        setUptimeDisplay("Unknown");
      }
    } finally {
      if (showLoading) {
        setStatsLoading(false);
      }
    }
  };

  const installCommand = `curl -sSf https://install.portr.dev | sh`;
  const homebrewCommand = `brew install amalshaji/taps/portr`;
  const helpCommand = "portr -h";

  const handleCopy = (text: string) => {
    copyCodeToClipboard(text);
  };

  useEffect(() => {
    getSetupScript();
    getDashboardStats(true);

    // Set up polling interval to refresh stats every 5 seconds
    const statsPollingInterval = setInterval(() => {
      getDashboardStats(false);
    }, 5000);

    // Set up uptime interval to update every second
    const uptimeInterval = setInterval(updateUptime, 1000);

    return () => {
      clearInterval(statsPollingInterval);
      clearInterval(uptimeInterval);
    };
  }, [team]);

  return (
    <div className="space-y-8">
      {/* Dashboard Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-black">
            Dashboard
          </h1>
          <p className="text-gray-600">Welcome to your {team} dashboard.</p>
        </div>
        <Button variant="outline" asChild className="flex items-center gap-2">
          <a href="https://portr.dev" target="_blank" rel="noopener noreferrer">
            <Terminal className="h-4 w-4" />
            Documentation
          </a>
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between space-y-0 pb-2">
            <p className="text-sm font-medium">Active Connections</p>
            <Globe className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="text-2xl font-bold">
            {statsLoading ? "..." : dashboardStats.activeConnections}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between space-y-0 pb-2">
            <p className="text-sm font-medium">Team Members</p>
            <Users className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="text-2xl font-bold">
            {statsLoading ? "..." : dashboardStats.totalUsers}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between space-y-0 pb-2">
            <p className="text-sm font-medium">Server Uptime</p>
            <Shield className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="text-2xl font-bold">{uptimeDisplay}</div>
        </div>

        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between space-y-0 pb-2">
            <p className="text-sm font-medium">Memory Usage</p>
            <HardDrive className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="text-2xl font-bold">
            {statsLoading
              ? "..."
              : `${systemStats.systemMemoryUsagePercent.toFixed(1)}%`}
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            {statsLoading
              ? ""
              : `${systemStats.systemMemoryUsedGB.toFixed(
                  1
                )}GB / ${systemStats.systemMemoryTotalGB.toFixed(1)}GB`}
          </p>
        </div>

        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between space-y-0 pb-2">
            <p className="text-sm font-medium">CPU Usage</p>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="text-2xl font-bold">
            {statsLoading
              ? "..."
              : `${systemStats.cpuUsagePercent.toFixed(1)}%`}
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            {statsLoading ? "" : `${systemStats.numCpu} cores`}
          </p>
        </div>
      </div>

      {/* System Information */}
      <div className="rounded-lg border bg-card p-6">
        <div className="mb-6">
          <h2 className="text-xl font-semibold">System Information</h2>
          <p className="text-muted-foreground mt-1">
            Server hardware and runtime details
          </p>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Hostname
            </p>
            <p className="text-sm">
              {statsLoading ? "..." : systemStats.hostname || "Unknown"}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Operating System
            </p>
            <p className="text-sm">
              {statsLoading ? "..." : systemStats.os || "Unknown"}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Architecture
            </p>
            <p className="text-sm">
              {statsLoading ? "..." : systemStats.architecture || "Unknown"}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              CPU Cores
            </p>
            <p className="text-sm">
              {statsLoading ? "..." : `${systemStats.numCpu} cores`}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              CPU Usage
            </p>
            <p className="text-sm">
              {statsLoading
                ? "..."
                : `${systemStats.cpuUsagePercent.toFixed(2)}%`}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Total System Memory
            </p>
            <p className="text-sm">
              {statsLoading
                ? "..."
                : `${systemStats.systemMemoryTotalGB.toFixed(2)} GB`}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Application Memory Usage
            </p>
            <p className="text-sm">
              {statsLoading
                ? "..."
                : `${systemStats.memoryUsedMB.toFixed(1)} MB`}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Application Memory Pool
            </p>
            <p className="text-sm">
              {statsLoading
                ? "..."
                : `${systemStats.memoryTotalMB.toFixed(1)} MB`}
            </p>
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium text-muted-foreground">
              Active Go Routines
            </p>
            <p className="text-sm">
              {statsLoading ? "..." : systemStats.goroutines.toLocaleString()}
            </p>
          </div>
        </div>
      </div>

      {/* Client Setup Section */}
      <div className="rounded-lg border bg-card p-6">
        <div className="mb-6">
          <h2 className="text-xl font-semibold">Client Setup</h2>
          <p className="text-muted-foreground mt-1">
            Follow these steps to set up and configure the portr client
          </p>
        </div>
        <div className="space-y-6">
          <div className="rounded-lg border bg-muted/50 p-6">
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
              <span className="flex h-6 w-6 rounded-full bg-primary text-primary-foreground items-center justify-center text-xs font-semibold">
                1
              </span>
              Install the portr client
            </h3>

            <div className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground mb-2">
                  Using the install script:
                </p>
                <div className="relative group">
                  <pre className="bg-muted p-3 rounded-lg text-sm font-mono overflow-x-auto">
                    {installCommand}
                  </pre>
                  <button
                    className="absolute right-2 top-2 p-1.5 bg-background border rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                    onClick={() => handleCopy(installCommand)}
                  >
                    <Copy className="h-3 w-3" />
                  </button>
                </div>
              </div>

              <div>
                <p className="text-sm text-muted-foreground mb-2">
                  Or using homebrew:
                </p>
                <div className="relative group">
                  <pre className="bg-muted p-3 rounded-lg text-sm font-mono overflow-x-auto">
                    {homebrewCommand}
                  </pre>
                  <button
                    className="absolute right-2 top-2 p-1.5 bg-background border rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                    onClick={() => handleCopy(homebrewCommand)}
                  >
                    <Copy className="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>

            <p className="mt-4 text-sm text-muted-foreground">
              You can also download the binary from the{" "}
              <a
                href="https://github.com/amalshaji/portr/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline font-medium"
              >
                GitHub releases
              </a>
            </p>
          </div>

          <div className="rounded-lg border bg-muted/50 p-6">
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
              <span className="flex h-6 w-6 rounded-full bg-primary text-primary-foreground items-center justify-center text-xs font-semibold">
                2
              </span>
              Run the following command to set up portr client auth
            </h3>

            <div className="relative group">
              <pre className="bg-muted p-3 rounded-lg text-sm font-mono overflow-x-auto">
                {setupScript}
              </pre>
              <button
                className="absolute right-2 top-2 p-1.5 bg-background border rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                onClick={() => handleCopy(setupScript)}
              >
                <Copy className="h-3 w-3" />
              </button>
            </div>

            <p className="mt-4 text-sm text-muted-foreground">
              Note: use{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                ./portr
              </code>{" "}
              instead of{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                portr
              </code>{" "}
              if the binary is in the same folder and not set in{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                $PATH
              </code>
            </p>
          </div>

          <div className="rounded-lg border bg-muted/50 p-6">
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
              <span className="flex h-6 w-6 rounded-full bg-primary text-primary-foreground items-center justify-center text-xs font-semibold">
                3
              </span>
              You're ready to use the tunnel
            </h3>

            <p className="text-muted-foreground text-sm">
              Run{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                {helpCommand}
              </code>{" "}
              or check out the{" "}
              <a
                href="https://portr.dev/docs/client"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline font-medium"
              >
                client documentation
              </a>{" "}
              for more information.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
