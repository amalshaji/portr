import { useEffect, useState } from "react";
import { Routes, Route, useParams, useNavigate } from "react-router-dom";
import {
  Home,
  Users,
  ArrowUpDown,
  User,
  PlusCircle,
  HelpCircle,
  LogOut,
  EllipsisVertical,
} from "lucide-react";
import AppLayout from "@/components/AppLayout";
import SidebarLink from "@/components/SidebarLink";
import TeamSelector from "@/components/TeamSelector";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarHeader,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarFooter,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { useUserStore } from "@/lib/store";
import NewTeamDialog from "@/components/NewTeamDialog";
import Overview from "./Overview";
import Connections from "./Connections";
import UsersPage from "./UsersPage";
import MyAccount from "./MyAccount";
import NotFound from "../NotFound";

export default function AppPage() {
  const { team } = useParams<{ team: string }>();
  const navigate = useNavigate();
  const { currentUser, setCurrentUser } = useUserStore();
  const [newTeamDialogOpen, setNewTeamDialogOpen] = useState(false);

  useEffect(() => {
    if (!team) return;

    const getLoggedInUser = async () => {
      try {
        const response = await fetch("/api/v1/user/me", {
          headers: {
            "Content-Type": "application/json",
            "x-team-slug": team,
          },
        });
        if (response.ok) {
          const userData = await response.json();
          setCurrentUser(userData);
        }
      } catch (err) {
        console.error("Failed to get user:", err);
      }
    };

    getLoggedInUser();
  }, [team, setCurrentUser]);

  const handleLogout = async () => {
    try {
      const res = await fetch("/api/v1/auth/logout", {
        method: "POST",
      });
      if (res.ok) {
        navigate("/");
      }
    } catch (err) {
      console.error("Logout failed:", err);
    }
  };

  const handleImageError = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const target = e.target as HTMLImageElement;
    if (target) {
      target.style.display = "none";
      const sibling = target.nextElementSibling as HTMLElement;
      if (sibling) {
        sibling.style.display = "flex";
      }
    }
  };

  const sidebar = (
    <>
      <SidebarHeader>
        <TeamSelector />
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Main</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild>
                  <SidebarLink to={`/${team}/overview`}>
                    <Home className="h-4 w-4" />
                    Overview
                  </SidebarLink>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild>
                  <SidebarLink to={`/${team}/connections`}>
                    <ArrowUpDown className="h-4 w-4" />
                    Connections
                  </SidebarLink>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Management</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild>
                  <SidebarLink to={`/${team}/users`}>
                    <Users className="h-4 w-4" />
                    Users
                  </SidebarLink>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild>
                  <SidebarLink to={`/${team}/my-account`}>
                    <User className="h-4 w-4" />
                    Account & Settings
                  </SidebarLink>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {currentUser?.user?.is_superuser && (
          <SidebarGroup>
            <SidebarGroupLabel>Admin</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton onClick={() => setNewTeamDialogOpen(true)}>
                    <PlusCircle className="h-4 w-4" />
                    New Team
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        <SidebarSeparator />

        <SidebarGroup>
          <SidebarGroupContent>
            <div className="border bg-muted/50 p-3 m-2 rounded-lg">
              <div className="flex items-center gap-3">
                <HelpCircle className="h-5 w-5" />
                <div>
                  <h3 className="text-sm font-medium">Need help?</h3>
                  <p className="text-xs text-muted-foreground">
                    Check our documentation
                  </p>
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                asChild
                className="mt-2 w-full text-xs"
              >
                <a
                  href="https://portr.dev"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  View Documentation
                </a>
              </Button>
            </div>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton className="w-full">
                  <div className="flex items-center gap-3 flex-1">
                    <div className="relative">
                      {currentUser?.user?.github_user?.github_avatar_url ? (
                        <img
                          className="h-8 w-8 border object-cover bg-muted"
                          src={currentUser.user.github_user.github_avatar_url}
                          alt={`${
                            currentUser.user.first_name ||
                            currentUser.user.email
                          } avatar`}
                          style={{ borderRadius: 0 }}
                          onError={handleImageError}
                        />
                      ) : (
                        <div className="h-8 w-8 rounded-full bg-muted flex items-center justify-center">
                          <span className="text-sm font-medium">
                            {currentUser?.user?.first_name
                              ? currentUser.user.first_name
                                  .charAt(0)
                                  .toUpperCase()
                              : currentUser?.user?.email
                                  ?.charAt(0)
                                  .toUpperCase() || "?"}
                          </span>
                        </div>
                      )}
                      <span className="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 bg-green-600 border-2 border-background rounded-full" />
                    </div>
                    <div className="flex flex-col justify-start flex-1 min-w-0">
                      <span className="text-sm font-medium truncate">
                        {currentUser?.user?.first_name
                          ? `${currentUser.user.first_name} ${
                              currentUser.user.last_name || ""
                            }`
                          : currentUser?.user?.email}
                      </span>
                    </div>
                  </div>
                  <EllipsisVertical className="h-4 w-4" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-[200px]">
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="text-red-600"
                >
                  <LogOut className="h-4 w-4 mr-2" />
                  <span>Logout</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </>
  );

  return (
    <>
      <NewTeamDialog
        isOpen={newTeamDialogOpen}
        setIsOpen={setNewTeamDialogOpen}
      />
      <AppLayout sidebar={sidebar}>
        <Routes>
          <Route path="/overview" element={<Overview />} />
          <Route path="/connections" element={<Connections />} />
          <Route path="/my-account" element={<MyAccount />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </AppLayout>
    </>
  );
}
