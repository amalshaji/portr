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
import Panel from "@/components/Panel";
import RouteLine, {
  connectionRouteName,
  connectionRouteState,
} from "@/components/RouteLine";
import SegmentedControl from "@/components/SegmentedControl";
import { Pagination } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDuration } from "@/lib/humanize";
import { updateQueryParam } from "@/lib/utils";
import type { Connection } from "@/types";

const filters = [
  { value: "recent" as const, label: "All" },
  { value: "active" as const, label: "Active" },
];

type ConnectionFilter = (typeof filters)[number]["value"];

export default function Connections() {
  const { team } = useParams<{ team: string }>();
  const [connections, setConnections] = useState<Connection[]>([]);
  const [connectionsLoading, setConnectionsLoading] = useState(true);

  const urlParams = new URLSearchParams(window.location.search);
  const [connectionType, setConnectionType] = useState<ConnectionFilter>(
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
        <SegmentedControl<ConnectionFilter>
          ariaLabel="Filter connections"
          options={filters}
          value={connectionType}
          onChange={(value) => {
            setConnectionType(value);
            setPageNo(1);
          }}
        />

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

      <Panel flush>
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
                    ? formatDuration(
                        new Date(connection.closed_at).getTime() -
                          new Date(connection.started_at).getTime()
                      )
                    : "—";

                return (
                  <TableRow key={connection.id}>
                    <TableCell>
                      <ConnectionType type={connection.type} />
                    </TableCell>
                    {/* Endpoint and state read as one cell rather than two
                        columns the eye has to join up. */}
                    <TableCell className="min-w-64">
                      <RouteLine
                        name={connectionRouteName(connection)}
                        state={connectionRouteState(connection)}
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
      </Panel>

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
