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
} from "@/components/ui/chart";
import type { ChartData } from "@/types";

interface MetricsChartProps {
  chartData: ChartData;
  isLoading: boolean;
}

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
  data: any[];
  dataKey: string;
  config: any;
  isLoading: boolean;
  isPercentage?: boolean;
}) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription className="text-xs">{description}</CardDescription>
        </CardHeader>
        <CardContent className="p-4">
          <div className="h-[220px] flex items-center justify-center text-muted-foreground">
            Loading...
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription className="text-xs">{description}</CardDescription>
        </CardHeader>
        <CardContent className="p-4">
          <div className="h-[220px] flex items-center justify-center text-muted-foreground">
            No data available
          </div>
        </CardContent>
      </Card>
    );
  }

  // Check if all values are 0 (which might make the line invisible)
  const hasNonZeroValues = data.some(
    (point) =>
      point.value !== 0 && point.value !== null && point.value !== undefined
  );
  if (!hasNonZeroValues && data.length > 0) {
    return (
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription className="text-xs">{description}</CardDescription>
        </CardHeader>
        <CardContent className="p-4">
          <div className="h-[220px] flex items-center justify-center text-muted-foreground">
            Data available but all values are 0
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription className="text-xs">{description}</CardDescription>
      </CardHeader>
      <CardContent className="p-4">
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
              stroke="#3b82f6"
              strokeWidth={2}
              dot={false}
              activeDot={{
                r: 6,
                stroke: "#3b82f6",
                strokeWidth: 2,
                fill: "#fff",
              }}
            />
          </LineChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

export function MetricsChart({ chartData, isLoading }: MetricsChartProps) {
  // Transform data for individual charts
  const processMetricData = (metricData: any[]) => {
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
  const memoryUsageConfig = {
    value: {
      label: "Memory Usage (MB)",
    },
  };

  const cpuUsageConfig = {
    value: {
      label: "CPU Usage (%)",
    },
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight mb-2">
          System Metrics
        </h2>
        <p className="text-muted-foreground">
          Real-time monitoring of system performance and connections
        </p>
      </div>

      {/* Charts Grid */}
      <div className="grid gap-4 md:grid-cols-2">
        <MetricChart
          title="CPU Usage"
          description="CPU utilization percentage"
          data={processMetricData(chartData.cpu_usage)}
          dataKey="value"
          config={cpuUsageConfig}
          isLoading={isLoading}
          isPercentage={true}
        />

        <MetricChart
          title="Memory Usage"
          description="System memory usage"
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
    </div>
  );
}
