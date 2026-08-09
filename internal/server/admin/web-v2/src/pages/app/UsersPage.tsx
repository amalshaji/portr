import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Plus, Mail, MoreHorizontal, Trash2, LoaderCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import Panel from "@/components/Panel";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import DateField from "@/components/DateField";
import InviteUserDialog from "@/components/InviteUserDialog";
import { useUserStore } from "@/lib/store";
import { toast } from "sonner";
import { Pagination } from "@/components/ui/pagination";
import type { TeamUser } from "@/types";

export default function UsersPage() {
  const { team } = useParams<{ team: string }>();
  const { currentUser } = useUserStore();
  const [users, setUsers] = useState<TeamUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalUsers, setTotalUsers] = useState(0);
  const usersPerPage = 10;

  const getUsers = async (page: number = currentPage) => {
    if (!team) return;

    setUsersLoading(true);
    try {
      const res = await fetch(
        `/api/v1/team/users?page=${page}&page_size=${usersPerPage}`,
        {
          headers: {
            "x-team-slug": team,
          },
        }
      );

      if (res.ok) {
        const data = await res.json();
        setUsers(data.data || []);
        setTotalUsers(data.count || 0);
      }
    } catch (error) {
      console.error("Failed to fetch users:", error);
      setUsers([]);
      setTotalUsers(0);
    } finally {
      setUsersLoading(false);
    }
  };

  useEffect(() => {
    getUsers();
  }, [team]);

  const handleInviteUser = () => {
    setInviteDialogOpen(true);
  };

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    getUsers(page);
  };

  const handleDeleteUser = async (userId: number) => {
    if (!team) return;

    setDeleteLoading(userId.toString());
    try {
      const response = await fetch(`/api/v1/team/users/${userId}`, {
        method: "DELETE",
        headers: {
          "x-team-slug": team,
        },
      });

      if (response.ok) {
        setUsers((prev) => prev.filter((user) => user.id !== userId));
        toast.success("User removed from team");
      } else {
        toast.error("Failed to remove user");
      }
    } catch (error) {
      console.error("Error deleting user:", error);
      toast.error("Failed to remove user");
    } finally {
      setDeleteLoading(null);
    }
  };

  const canDeleteUser = (user: TeamUser) => {
    if (!currentUser) return false;
    if (currentUser.role === "member") return false;
    if (user.id === currentUser.id) return false;
    if (user.user.is_superuser && !currentUser.user.is_superuser) return false;
    return true;
  };

  return (
    <>
      <InviteUserDialog
        isOpen={inviteDialogOpen}
        setIsOpen={setInviteDialogOpen}
        onSuccess={() => getUsers(currentPage)}
      />
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <p className="text-sm text-muted-foreground">
            People with access to <span className="data">{team}</span> and what
            they can do.
          </p>
          <Button
            onClick={handleInviteUser}
            disabled={currentUser?.role === "member"}
            size="sm"
          >
            <Plus className="size-4" />
            Invite user
          </Button>
        </div>

        <Panel flush>
            {usersLoading ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Joined</TableHead>
                    <TableHead className="w-[50px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {Array.from({ length: 5 }).map((_, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Skeleton className="size-8 rounded-sm" />
                          <div className="space-y-1.5">
                            <Skeleton className="h-3.5 w-32" />
                            <Skeleton className="h-3 w-44" />
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-5 w-16" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-3.5 w-28" />
                      </TableCell>
                      <TableCell />
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : users.length === 0 ? (
              <div className="px-6 py-12 text-center">
                <p className="text-sm font-medium">No users yet</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Invite a teammate to give them access to this team.
                </p>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Joined</TableHead>
                    <TableHead className="w-[50px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((teamUser) => (
                    <TableRow key={teamUser.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className="flex size-8 items-center justify-center overflow-hidden rounded-sm bg-muted">
                            {teamUser.user.github_user?.github_avatar_url ? (
                              <img
                                src={
                                  teamUser.user.github_user.github_avatar_url
                                }
                                alt=""
                                className="size-8 object-cover"
                              />
                            ) : (
                              <span className="data text-xs font-semibold">
                                {teamUser.user.first_name
                                  ? teamUser.user.first_name
                                      .charAt(0)
                                      .toUpperCase()
                                  : teamUser.user.email.charAt(0).toUpperCase()}
                              </span>
                            )}
                          </div>
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium">
                              {teamUser.user.first_name
                                ? `${teamUser.user.first_name} ${
                                    teamUser.user.last_name || ""
                                  }`
                                : teamUser.user.email}
                            </p>
                            <p className="data flex items-center gap-1 truncate text-xs text-muted-foreground">
                              <Mail className="size-3 shrink-0" />
                              {teamUser.user.email}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            teamUser.user.is_superuser
                              ? "destructive"
                              : teamUser.role === "admin"
                              ? "default"
                              : "secondary"
                          }
                          className="capitalize"
                        >
                          {teamUser.user.is_superuser
                            ? "Superuser"
                            : teamUser.role}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <DateField date={teamUser.created_at} />
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            {canDeleteUser(teamUser) && (
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <DropdownMenuItem
                                    className="text-destructive cursor-pointer"
                                    onSelect={(e) => e.preventDefault()}
                                  >
                                    <Trash2 className="mr-2 size-4" />
                                    Remove from team
                                  </DropdownMenuItem>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>
                                      Remove {teamUser.user.email} from {team}?
                                    </AlertDialogTitle>
                                    <AlertDialogDescription>
                                      They lose access to this team immediately.
                                      Their tunnels stop working and this cannot
                                      be undone.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>
                                      Cancel
                                    </AlertDialogCancel>
                                    <AlertDialogAction
                                      onClick={() =>
                                        handleDeleteUser(teamUser.id)
                                      }
                                      className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                      disabled={
                                        deleteLoading === teamUser.id.toString()
                                      }
                                    >
                                      {deleteLoading ===
                                        teamUser.id.toString() && (
                                        <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
                                      )}
                                      Remove
                                    </AlertDialogAction>
                                  </AlertDialogFooter>
                                </AlertDialogContent>
                              </AlertDialog>
                            )}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
        </Panel>

        {totalUsers > usersPerPage && (
          <div className="flex justify-center">
            <Pagination
              count={totalUsers}
              perPage={usersPerPage}
              currentPage={currentPage}
              onPageChange={handlePageChange}
            />
          </div>
        )}
      </div>
    </>
  );
}
