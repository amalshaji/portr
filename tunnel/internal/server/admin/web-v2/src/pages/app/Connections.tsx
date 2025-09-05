import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import ConnectionStatus from "@/components/ConnectionStatus";
import ConnectionType from "@/components/ConnectionType";
import DateField from "@/components/DateField";
import { updateQueryParam } from "@/lib/utils";
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

export default function Connections() {
  const { team } = useParams<{ team: string }>();
  const [connections, setConnections] = useState<Connection[]>([]);
  const [connectionsLoading, setConnectionsLoading] = useState(true);
  const [checked, setChecked] = useState(false);

  const urlParams = new URLSearchParams(window.location.search);
  const [connectionType, setConnectionType] = useState(
    urlParams.get("type") || "recent"
  );
  const [pageNo, setPageNo] = useState(
    parseInt(urlParams.get("page") || "1", 10) || 1
  );
  const [totalItems, setTotalItems] = useState(0);

  const getConnections = async (
    type: string = "recent",
    pageNoStr: string = "1"
  ) => {
    if (!team) return;

    setConnectionsLoading(true);
    try {
      const res = await fetch(
        `/api/v1/connections/?type=${type}&page=${pageNoStr}`,
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
    if (checked) {
      if (connectionType === "recent") {
        setConnectionType("active");
        setPageNo(1);
      }
    } else {
      if (connectionType === "active") {
        setConnectionType("recent");
        setPageNo(1);
      }
    }
  }, [checked, connectionType]);

  useEffect(() => {
    updateQueryParam(urlParams, "type", connectionType);
    updateQueryParam(urlParams, "page", pageNo.toString());
    getConnections(connectionType, pageNo.toString());
  }, [connectionType, pageNo, team]);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Connections</h1>
          <p className="text-muted-foreground">
            Manage your tunnel connections
          </p>
        </div>

        <div className="flex items-center space-x-2">
          <Checkbox
            id="show-active"
            checked={checked}
            onCheckedChange={(checked) => setChecked(checked as boolean)}
          />
          <Label
            htmlFor="show-active"
            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
          >
            Show only active connections
          </Label>
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-col sm:flex-row sm:justify-between gap-4">
          <div>
            <CardTitle className="text-xl">Connection History</CardTitle>
            <CardDescription>
              View and manage your tunnel connections
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-sm border overflow-hidden">
            {connectionsLoading ? (
              <div className="p-6 text-center">
                <p className="text-muted-foreground">Loading connections...</p>
              </div>
            ) : connections.length === 0 ? (
              <div className="p-6 text-center">
                <p className="text-muted-foreground">No connections found</p>
                <p className="text-sm text-muted-foreground mt-1">
                  {connectionType === "active"
                    ? "No active connections at the moment"
                    : "Start a tunnel to see connections here"}
                </p>
              </div>
            ) : (
              <>
                <div className="w-full flex justify-end p-2 border-b bg-muted/50">
                  {/* Pagination would go here */}
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Type</TableHead>
                      <TableHead>Port</TableHead>
                      <TableHead>Subdomain</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created at</TableHead>
                      <TableHead>Duration</TableHead>
                      <TableHead>Created by</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {connections.map((connection) => {
                      const duration =
                        connection.status === "active"
                          ? "-"
                          : connection.started_at && connection.closed_at
                          ? humanizeTimeMs(
                              new Date(connection.closed_at).getTime() -
                                new Date(connection.started_at).getTime()
                            )
                          : "-";

                      return (
                        <TableRow key={connection.id}>
                          <TableCell>
                            <ConnectionType type={connection.type} />
                          </TableCell>
                          <TableCell className="font-mono text-sm">
                            {connection.port || "-"}
                          </TableCell>
                          <TableCell className="font-mono text-sm">
                            {connection.subdomain || "-"}
                          </TableCell>
                          <TableCell>
                            <ConnectionStatus status={connection.status} />
                          </TableCell>
                          <TableCell>
                            <DateField date={connection.created_at} />
                          </TableCell>
                          <TableCell className="text-sm">{duration}</TableCell>
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
              </>
            )}
          </div>
        </CardContent>
      </Card>

      {totalItems > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {connections.length} of {totalItems} connections
          </p>
          {/* Simple pagination could be added here */}
        </div>
      )}
    </div>
  );
}
