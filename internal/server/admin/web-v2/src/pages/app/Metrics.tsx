import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Globe, Shield, Users } from "lucide-react";
import type { DashboardStats, SystemStats, ChartData } from "@/types";
import { MetricsChart } from "@/components/MetricsChart";
import { Skeleton } from "@/components/ui/skeleton";

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

  const readouts = [
    {
      label: "Active tunnels",
      value: dashboardStats.activeConnections.toString(),
      Icon: Globe,
    },
    {
      label: "Team members",
      value: dashboardStats.totalUsers.toString(),
      Icon: Users,
    },
    { label: "Server uptime", value: uptimeDisplay, Icon: Shield },
  ];

  const systemInfo = [
    { label: "Hostname", value: systemStats.hostname || "unknown" },
    { label: "Operating system", value: systemStats.os || "unknown" },
    { label: "Architecture", value: systemStats.architecture || "unknown" },
    { label: "CPU cores", value: `${systemStats.numCpu}` },
    { label: "CPU usage", value: `${systemStats.cpuUsagePercent.toFixed(2)}%` },
    {
      label: "System memory",
      value: `${systemStats.systemMemoryTotalGB.toFixed(2)} GB`,
    },
    {
      label: "App memory in use",
      value: `${systemStats.memoryUsedMB.toFixed(1)} MB`,
    },
    {
      label: "App memory pool",
      value: `${systemStats.memoryTotalMB.toFixed(1)} MB`,
    },
    { label: "Goroutines", value: systemStats.goroutines.toLocaleString() },
  ];

  return (
    <div className="space-y-6">
      <div className="grid gap-3 sm:grid-cols-3">
        {readouts.map(({ label, value, Icon }) => (
          <div
            key={label}
            className="rounded-md border border-border bg-card p-4"
          >
            <div className="flex items-center justify-between gap-2">
              <p className="eyebrow">{label}</p>
              <Icon className="size-3.5 text-muted-foreground" />
            </div>
            {statsLoading ? (
              <Skeleton className="mt-2 h-8 w-24" />
            ) : (
              <p className="tabular mt-1 font-display text-2xl font-semibold">
                {value}
              </p>
            )}
          </div>
        ))}
      </div>

      <MetricsChart chartData={chartData} isLoading={statsLoading} />

      <section className="overflow-hidden rounded-md border border-border bg-card">
        <header className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold">System information</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Hardware and runtime details for the machine running Portr.
          </p>
        </header>
        <dl className="grid gap-x-8 gap-y-0 p-2 sm:grid-cols-2 lg:grid-cols-3">
          {systemInfo.map(({ label, value }) => (
            <div
              key={label}
              className="flex items-baseline justify-between gap-4 border-b border-border/60 px-2 py-2.5 last:border-b-0"
            >
              <dt className="text-xs text-muted-foreground">{label}</dt>
              <dd className="data min-w-0 truncate text-xs font-medium">
                {statsLoading ? (
                  <Skeleton className="h-3.5 w-16" />
                ) : (
                  value
                )}
              </dd>
            </div>
          ))}
        </dl>
      </section>
    </div>
  );
}
