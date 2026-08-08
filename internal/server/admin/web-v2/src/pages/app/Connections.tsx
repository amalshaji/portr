import { useState, useEffect } from "react";
import { Link, useParams } from "react-router-dom";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import ConnectionType from "@/components/ConnectionType";
import DateField from "@/components/DateField";
import RouteLine, { type RouteState } from "@/components/RouteLine";
import { Pagination } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { cn, updateQueryParam } from "@/lib/utils";
import type { Connection } from "@/types";

const humanizeTimeMs = (ms: number): string => {
  const seconds = Math.floor(ms / 1000) % 60;
  const minutes = Math.floor(ms / (1000 * 60)) % 60;
  const hours = Math.floor(ms / (1000 * 60 * 60)) % 24;
  const days = Math.floor(ms / (1000 * 60 * 60 * 24));

  if (days > 0) return `${days}d ${hours}h ${minutes}m ${seconds}s`;
  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
};

const routeState = (connection: Connection): RouteState => {
  if (connection.status === "active") return "live";
  if (connection.status === "reserved") return "unbound";
  return "closed";
};

const filters = [
  { value: "recent", label: "All" },
  { value: "active", label: "Active" },
] as const;

export default function Connections() {
  const { team } = useParams<{ team: string }>();
  const [connections, setConnections] = useState<Connection[]>([]);
  const [connectionsLoading, setConnectionsLoading] = useState(true);

  const urlParams = new URLSearchParams(window.location.search);
  const [connectionType, setConnectionType] = useState(
    urlParams.get("type") === "active" ? "active" : "recent"
  );
  const [pageNo, setPageNo] = useState(
    parseInt(urlParams.get("page") || "1", 10) || 1
  );
  const [pageSize, setPageSize] = useState(
    parseInt(urlParams.get("page_size") || "10", 10) || 10
  );
  const [totalItems, setTotalItems] = useState(0);

  const getConnections = async (
    type: string = "recent",
    pageNoStr: string = "1",
    pageSizeStr: string = "10"
  ) => {
    if (!team) return;

    setConnectionsLoading(true);
    try {
      const res = await fetch(
        `/api/v1/connections/?type=${type}&page=${pageNoStr}&page_size=${pageSizeStr}`,
        {
          headers: {
            "x-team-slug": team,
          },
        }
      );

      if (res.ok) {
        const data = await res.json();
        setConnections(data.data || []);
        setTotalItems(data.count || 0);
      }
    } catch (error) {
      console.error("Failed to fetch connections:", error);
      setConnections([]);
      setTotalItems(0);
    } finally {
      setConnectionsLoading(false);
    }
  };

  useEffect(() => {
    updateQueryParam(urlParams, "type", connectionType);
    updateQueryParam(urlParams, "page", pageNo.toString());
    updateQueryParam(urlParams, "page_size", pageSize.toString());
    getConnections(connectionType, pageNo.toString(), pageSize.toString());
  }, [connectionType, pageNo, pageSize, team]);

  const columns = ["Type", "Route", "Opened", "Duration", "Opened by"];

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div
          role="group"
          aria-label="Filter connections"
          className="flex w-fit rounded-md border border-border bg-muted/60 p-0.5"
        >
          {filters.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              aria-pressed={connectionType === value}
              onClick={() => {
                setConnectionType(value);
                setPageNo(1);
              }}
              className={cn(
                "rounded-sm px-3 py-1 text-xs font-medium outline-none transition-colors duration-(--portr-duration-micro) ease-portr focus-visible:ring-2 focus-visible:ring-ring",
                connectionType === value
                  ? "bg-background text-foreground shadow-xs"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {label}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2">
          <Label htmlFor="page-size" className="text-xs text-muted-foreground">
            Rows
          </Label>
          <Select
            value={pageSize.toString()}
            onValueChange={(value) => {
              setPageSize(parseInt(value, 10));
              setPageNo(1);
            }}
          >
            <SelectTrigger className="h-8 w-[4.5rem]" id="page-size" size="sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">10</SelectItem>
              <SelectItem value="25">25</SelectItem>
              <SelectItem value="50">50</SelectItem>
              <SelectItem value="100">100</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="overflow-hidden rounded-md border border-border bg-card">
        {connectionsLoading ? (
          <Table>
            <TableHeader>
              <TableRow>
                {columns.map((column) => (
                  <TableHead key={column}>{column}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: pageSize }).map((_, index) => (
                <TableRow key={index}>
                  {columns.map((column) => (
                    <TableCell key={column}>
                      <Skeleton className="h-4 w-full max-w-40" />
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : connections.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <p className="text-sm font-medium">
              {connectionType === "active"
                ? "No tunnels are running"
                : "No connections yet"}
            </p>
            <p className="mx-auto mt-1 max-w-sm text-xs text-muted-foreground">
              {connectionType === "active"
                ? "Connections appear here while a tunnel is open."
                : "Start a tunnel from the Portr client and it shows up here."}
            </p>
            {connectionType !== "active" && (
              <Button asChild variant="outline" size="sm" className="mt-4">
                <Link to={`/${team}/overview`}>Set up the client</Link>
              </Button>
            )}
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                {columns.map((column) => (
                  <TableHead key={column}>{column}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {connections.map((connection) => {
                const duration =
                  connection.status === "active"
                    ? "—"
                    : connection.started_at && connection.closed_at
                    ? humanizeTimeMs(
                        new Date(connection.closed_at).getTime() -
                          new Date(connection.started_at).getTime()
                      )
                    : "—";

                return (
                  <TableRow key={connection.id}>
                    <TableCell>
                      <ConnectionType type={connection.type} />
                    </TableCell>
                    {/* Name, port and state are one binding, not three
                        independent facts — they read as one cell. */}
                    <TableCell className="min-w-64">
                      <RouteLine
                        name={connection.subdomain || `${connection.type} tunnel`}
                        port={connection.port}
                        state={routeState(connection)}
                      />
                    </TableCell>
                    <TableCell>
                      <DateField date={connection.created_at} />
                    </TableCell>
                    <TableCell className="data text-xs">{duration}</TableCell>
                    <TableCell className="text-sm">
                      {connection.created_by.user.first_name
                        ? `${connection.created_by.user.first_name} ${
                            connection.created_by.user.last_name || ""
                          }`
                        : connection.created_by.user.email}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      {totalItems > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">
            <span className="tabular">
              {(pageNo - 1) * pageSize + 1}–
              {Math.min(pageNo * pageSize, totalItems)}
            </span>{" "}
            of <span className="tabular">{totalItems}</span>
          </p>
          <Pagination
            count={totalItems}
            perPage={pageSize}
            currentPage={pageNo}
            onPageChange={setPageNo}
          />
        </div>
      )}
    </div>
  );
}
