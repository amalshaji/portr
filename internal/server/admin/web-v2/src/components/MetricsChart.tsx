import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import Panel from "@/components/Panel";
import {
  ChartContainer,
  type ChartConfig,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import type { ChartData, ChartDataPoint } from "@/types";

interface MetricsChartProps {
  chartData: ChartData;
  isLoading: boolean;
}

type ProcessedMetricPoint = {
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

/** Declared at module scope: a component defined inside MetricChart would get a
 *  new identity on every poll, remounting the chart each time. */
function ChartPanel({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <Panel title={title} description={description} flush>
      <div className="p-3">{children}</div>
    </Panel>
  );
}

function Placeholder({ message }: { message: string }) {
  return (
    <div className="flex h-[220px] items-center justify-center text-xs text-muted-foreground">
      {message}
    </div>
  );
}

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
  data: ProcessedMetricPoint[];
  dataKey: string;
  config: ChartConfig;
  isLoading: boolean;
  isPercentage?: boolean;
}) {
  // An all-zero series draws an invisible line, so say so rather than showing
  // an empty chart.
  const hasNonZeroValues = data?.some(
    (point) =>
      point.value !== 0 && point.value !== null && point.value !== undefined
  );

  const body = isLoading ? (
    <Skeleton className="h-[220px] w-full rounded-sm" />
  ) : !data || data.length === 0 ? (
    <Placeholder message="No data yet" />
  ) : !hasNonZeroValues ? (
    <Placeholder message="Reporting zero across the window" />
  ) : null;

  if (body) {
    return (
      <ChartPanel title={title} description={description}>
        {body}
      </ChartPanel>
    );
  }

  return (
    <ChartPanel title={title} description={description}>
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
              tickMargin={8}
              // One sample every 5s fills the axis with overlapping timestamps;
              // let recharts drop ticks that cannot fit and always keep the
              // ends so the window stays readable.
              interval="preserveStartEnd"
              minTickGap={56}
              tick={{ fontSize: 10 }}
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
              tick={{ fontSize: 10 }}
              width={52}
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
    </ChartPanel>
  );
}

export function MetricsChart({ chartData, isLoading }: MetricsChartProps) {
  // Transform data for individual charts
  const processMetricData = (
    metricData: ChartDataPoint[]
  ): ProcessedMetricPoint[] => {
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
