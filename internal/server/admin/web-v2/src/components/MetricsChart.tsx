import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import type { ChartData } from "@/types";

interface MetricsChartProps {
  chartData: ChartData;
  isLoading: boolean;
}

type MetricDataPoint = { timestamp: string; value: number };
type ProcessedMetricDataPoint = {
  timestamp: number;
  timestampLabel: string;
  value: number;
};

// Humanize number formatting for Y-axis labels
const humanizeNumber = (
  value: number,
  isPercentage: boolean = false,
  useDecimals: boolean = false
): string => {
  if (isPercentage) {
    if (useDecimals) {
      return `${value.toFixed(1)}%`;
    }
    return `${Math.round(value)}%`;
  }

  // For memory values (MB)
  if (value >= 1024) {
    return `${(value / 1024).toFixed(1)} GB`;
  } else if (value >= 1) {
    return `${value.toFixed(0)} MB`;
  } else {
    return `${(value * 1024).toFixed(0)} KB`;
  }
};

// Individual chart component for each metric
function MetricChart({
  title,
  description,
  data,
  dataKey,
  config,
  isLoading,
  isPercentage = false,
}: {
  title: string;
  description: string;
  data: ProcessedMetricDataPoint[];
  dataKey: string;
  config: ChartConfig;
  isLoading: boolean;
  isPercentage?: boolean;
}) {
  // Every state renders the same card, only the body changes.
  const Shell = ({ children }: { children: React.ReactNode }) => (
    <Card className="gap-0 rounded-md border-border py-0 shadow-none">
      <CardHeader className="border-b border-border px-4 py-3">
        <CardTitle className="text-sm font-semibold">{title}</CardTitle>
        <CardDescription className="text-xs">{description}</CardDescription>
      </CardHeader>
      <CardContent className="p-3">{children}</CardContent>
    </Card>
  );

  const Placeholder = ({ message }: { message: string }) => (
    <div className="flex h-[220px] items-center justify-center text-xs text-muted-foreground">
      {message}
    </div>
  );

  if (isLoading) {
    return (
      <Shell>
        <Skeleton className="h-[220px] w-full rounded-sm" />
      </Shell>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Shell>
        <Placeholder message="No data yet" />
      </Shell>
    );
  }

  // An all-zero series draws an invisible line, so say so rather than showing
  // an empty chart.
  const hasNonZeroValues = data.some(
    (point) =>
      point.value !== 0 && point.value !== null && point.value !== undefined
  );
  if (!hasNonZeroValues) {
    return (
      <Shell>
        <Placeholder message="Reporting zero across the window" />
      </Shell>
    );
  }

  return (
    <Shell>
      <ChartContainer config={config} className="h-[220px] w-full">
          <LineChart
            accessibilityLayer
            data={data}
            margin={{
              left: 8,
              right: 8,
              top: 8,
              bottom: 8,
            }}
          >
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="timestamp"
              tickLine={false}
              axisLine={false}
              tickMargin={4}
              tickFormatter={(value) => {
                // Find the corresponding data point and return its formatted time
                if (data && typeof value === "number") {
                  const dataPoint = data.find(
                    (point) => point.timestamp === value
                  );
                  if (dataPoint && dataPoint.timestampLabel) {
                    return dataPoint.timestampLabel;
                  }
                }
                return value;
              }}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tickMargin={4}
              width={50}
              domain={["dataMin - 1", "dataMax + 1"]}
              tickFormatter={(value) => humanizeNumber(value, isPercentage)}
            />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  formatter={(value) => {
                    if (typeof value === "number") {
                      return [humanizeNumber(value, isPercentage, true), ""];
                    }
                    return [value, ""];
                  }}
                />
              }
              labelFormatter={(value, payload) => {
                if (
                  payload &&
                  payload[0] &&
                  payload[0].payload &&
                  payload[0].payload.timestampLabel
                ) {
                  return payload[0].payload.timestampLabel;
                }
                return value;
              }}
            />
            <Line
              dataKey={dataKey}
              type="linear"
              stroke="var(--color-chart-1)"
              strokeWidth={2}
              dot={false}
              activeDot={{
                r: 5,
                stroke: "var(--color-chart-1)",
                strokeWidth: 2,
                fill: "var(--color-card)",
              }}
              animationDuration={0}
            />
          </LineChart>
      </ChartContainer>
    </Shell>
  );
}

export function MetricsChart({ chartData, isLoading }: MetricsChartProps) {
  // Transform data for individual charts
  const processMetricData = (metricData: MetricDataPoint[] | undefined) => {
    if (!metricData) return [];

    // Ensure we're showing the most recent data by sorting chronologically
    const sortedData = [...metricData].sort(
      (a, b) =>
        new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );

    const processed = sortedData.map((point) => ({
      timestamp: new Date(point.timestamp).getTime(), // Use actual timestamp for proper time progression
      timestampLabel: new Date(point.timestamp).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      }),
      value: point.value,
    }));
    return processed;
  };

  // Individual chart configurations
  const memoryUsageConfig: ChartConfig = {
    value: {
      label: "Memory Usage (MB)",
    },
  };

  const cpuUsageConfig: ChartConfig = {
    value: {
      label: "CPU Usage (%)",
    },
  };

  return (
    <div className="grid gap-3 md:grid-cols-2">
        <MetricChart
          title="CPU usage"
          description="Percentage of available CPU in use"
          data={processMetricData(chartData.cpu_usage)}
          dataKey="value"
          config={cpuUsageConfig}
          isLoading={isLoading}
          isPercentage={true}
        />

        <MetricChart
          title="Memory usage"
          description="System memory held by the server"
          data={processMetricData(chartData.memory_usage)?.map((item) => ({
            ...item,
            value: item.value / (1024 * 1024), // Convert bytes to MB
          }))}
          dataKey="value"
          config={memoryUsageConfig}
          isLoading={isLoading}
          isPercentage={false}
        />
    </div>
  );
}
