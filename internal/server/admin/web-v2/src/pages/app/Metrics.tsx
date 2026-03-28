import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Globe, Shield, Terminal, Users } from "lucide-react";
import type { DashboardStats, SystemStats, ChartData } from "@/types";
import { MetricsChart } from "@/components/MetricsChart";

export default function Metrics() {
  const { team } = useParams<{ team: string }>();
  const [statsLoading, setStatsLoading] = useState(true);
  const [serverStartTime, setServerStartTime] = useState<string | null>(null);
  const [uptimeDisplay, setUptimeDisplay] = useState("...");

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

  const [chartData, setChartData] = useState<ChartData>({
    memory_usage: [],
    cpu_usage: [],
  });

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

        // Set chart data if available
        if (data.chart_data) {
          setChartData({
            memory_usage: data.chart_data.memory_usage || [],
            cpu_usage: data.chart_data.cpu_usage || [],
            latest: data.chart_data.latest,
          });
        }

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
        setChartData({
          memory_usage: [],
          cpu_usage: [],
        });
        setUptimeDisplay("Unknown");
      }
    } finally {
      if (showLoading) {
        setStatsLoading(false);
      }
    }
  };

  useEffect(() => {
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
      {/* Metrics Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-black">
            System Metrics
          </h1>
          <p className="text-gray-600">
            Monitor system performance and connections in real-time.
          </p>
        </div>
        <Terminal className="h-8 w-8 text-muted-foreground" />
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-3">
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
      </div>

      {/* Metrics Charts */}
      <MetricsChart chartData={chartData} isLoading={statsLoading} />

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
    </div>
  );
}
