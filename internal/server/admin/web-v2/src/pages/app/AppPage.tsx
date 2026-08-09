import { useEffect, useState } from "react"
import {
  Link,
  Route,
  Routes,
  useNavigate,
  useParams,
} from "react-router-dom"
import {
  Activity,
  ArrowUpDown,
  EllipsisVertical,
  Globe,
  HelpCircle,
  Home,
  LogOut,
  PlusCircle,
  Settings,
  User,
  Users,
} from "lucide-react"
import AppLayout from "@/components/AppLayout"
import NewTeamDialog from "@/components/NewTeamDialog"
import SidebarLink from "@/components/SidebarLink"
import TeamSelector from "@/components/TeamSelector"
import ThemeToggle from "@/components/ThemeToggle"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from "@/components/ui/sidebar"
import { useUserStore } from "@/lib/store"
import AutoSignupSettings from "../auto-signup/AutoSignupSettings"
import NotFound from "../NotFound"
import Connections from "./Connections"
import Metrics from "./Metrics"
import MyAccount from "./MyAccount"
import Overview from "./Overview"
import ReservedDomains from "./ReservedDomains"
import UsersPage from "./UsersPage"

const operateNav = [
  { label: "Overview", path: "overview", icon: Home },
  { label: "Metrics", path: "metrics", icon: Activity },
  { label: "Connections", path: "connections", icon: ArrowUpDown },
  { label: "Reserved domains", path: "reserved-domains", icon: Globe },
]

const manageNav = [
  { label: "Users", path: "users", icon: Users },
  { label: "Account & settings", path: "my-account", icon: User },
]

export default function AppPage() {
  const { team } = useParams<{ team: string }>()
  const navigate = useNavigate()
  const { currentUser, setCurrentUser } = useUserStore()
  const [newTeamDialogOpen, setNewTeamDialogOpen] = useState(false)

  useEffect(() => {
    if (!team) return

    const getLoggedInUser = async () => {
      try {
        const response = await fetch("/api/v1/user/me", {
          headers: {
            "Content-Type": "application/json",
            "x-team-slug": team,
          },
        })
        if (response.ok) {
          const userData = await response.json()
          setCurrentUser(userData)
        }
      } catch (err) {
        console.error("Failed to get user:", err)
      }
    }

    getLoggedInUser()
  }, [team, setCurrentUser])

  const handleLogout = async () => {
    try {
      const res = await fetch("/api/v1/auth/logout", { method: "POST" })
      if (res.ok) navigate("/")
    } catch (err) {
      console.error("Logout failed:", err)
    }
  }

  const displayName = currentUser?.user?.first_name
    ? `${currentUser.user.first_name} ${currentUser.user.last_name || ""}`.trim()
    : currentUser?.user?.email || "Signed in"
  const avatarInitial = (
    currentUser?.user?.first_name ||
    currentUser?.user?.email ||
    "?"
  )
    .charAt(0)
    .toUpperCase()

  const renderNavGroup = (
    label: string,
    items: typeof operateNav,
  ) => (
    <SidebarGroup className="pt-1 first:pt-2">
      <SidebarGroupLabel className="eyebrow h-6 px-2.5">
        {label}
      </SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map(({ label: itemLabel, path, icon: Icon }) => (
            <SidebarMenuItem key={path}>
              <SidebarMenuButton
                asChild
                tooltip={itemLabel}
                className="h-9 rounded-md p-0 hover:bg-transparent active:bg-transparent"
              >
                <SidebarLink to={`/${team}/${path}`}>
                  <Icon className="size-4" />
                  <span>{itemLabel}</span>
                </SidebarLink>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )

  const sidebar = (
    <>
      <SidebarHeader className="gap-2.5 p-2.5 pb-2">
        <Link
          to={`/${team}/overview`}
          aria-label="Portr overview"
          className="flex h-9 items-center gap-2.5 rounded-md px-2 outline-none transition-colors hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-sidebar-ring group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0"
        >
          <span className="flex size-7 shrink-0 items-center justify-center rounded-sm bg-[var(--portr-night-deep)]">
            <img
              src={`${import.meta.env.BASE_URL}portr-mark.svg`}
              alt=""
              className="size-5"
            />
          </span>
          <span className="min-w-0 group-data-[collapsible=icon]:hidden">
            <span className="block text-sm font-semibold tracking-tight text-foreground">
              Portr
            </span>
          </span>
        </Link>
        <TeamSelector />
      </SidebarHeader>

      <SidebarContent className="px-1.5 pb-2">
        {renderNavGroup("Operate", operateNav)}
        {renderNavGroup("Manage", manageNav)}

        {currentUser?.user?.is_superuser && (
          <SidebarGroup className="pt-1">
            <SidebarGroupLabel className="eyebrow h-6 px-2.5">
              Admin
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    onClick={() => setNewTeamDialogOpen(true)}
                    tooltip="New team"
                    className="h-9 gap-2.5 rounded-md px-2.5 text-sidebar-foreground/75"
                  >
                    <PlusCircle className="size-4" />
                    <span>New team</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    tooltip="GitHub auto signup"
                    className="h-9 rounded-md p-0 hover:bg-transparent active:bg-transparent"
                  >
                    <SidebarLink to={`/${team}/auto-signup`}>
                      <Settings className="size-4" />
                      <span>GitHub auto signup</span>
                    </SidebarLink>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        <SidebarSeparator className="my-1" />

        <SidebarGroup className="group-data-[collapsible=icon]:hidden">
          <SidebarGroupContent>
            <Button
              variant="ghost"
              asChild
              className="h-9 w-full justify-start gap-2.5 rounded-md px-2.5 text-sm font-normal text-sidebar-foreground/75"
            >
              <a
                href="https://portr.dev/docs"
                target="_blank"
                rel="noopener noreferrer"
              >
                <HelpCircle className="size-4" />
                <span>Documentation</span>
              </a>
            </Button>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="gap-2 border-t border-sidebar-border p-2.5">
        <div className="flex items-center justify-between gap-2 group-data-[collapsible=icon]:hidden">
          <span className="eyebrow">Theme</span>
          <ThemeToggle />
        </div>

        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton
                  size="lg"
                  tooltip={currentUser?.user?.email || "Account"}
                  className="h-11 w-full rounded-md px-2 hover:bg-sidebar-accent data-[state=open]:bg-sidebar-accent"
                >
                  <div className="flex min-w-0 flex-1 items-center gap-2.5">
                    <div className="relative size-7 shrink-0">
                      <div className="flex size-7 items-center justify-center rounded-sm bg-sidebar-primary text-sidebar-primary-foreground">
                        <span className="data text-[0.65rem] font-semibold">
                          {avatarInitial}
                        </span>
                      </div>
                      {currentUser?.user?.github_user?.github_avatar_url && (
                        <img
                          className="absolute inset-0 size-7 rounded-sm border object-cover bg-muted"
                          src={
                            currentUser.user.github_user.github_avatar_url
                          }
                          alt={`${displayName} avatar`}
                          onError={(event) => {
                            event.currentTarget.style.display = "none"
                          }}
                        />
                      )}
                    </div>
                    <div className="min-w-0 flex-1 text-left group-data-[collapsible=icon]:hidden">
                      <span className="block truncate text-xs font-medium">
                        {displayName}
                      </span>
                      <span className="data block truncate text-[0.65rem] font-normal text-muted-foreground">
                        {currentUser?.user?.email || "Account"}
                      </span>
                    </div>
                  </div>
                  <EllipsisVertical className="size-4 group-data-[collapsible=icon]:hidden" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52 p-1">
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="text-destructive focus:text-destructive"
                >
                  <LogOut className="mr-2 size-4" />
                  <span>Log out</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </>
  )

  return (
    <>
      <NewTeamDialog
        isOpen={newTeamDialogOpen}
        setIsOpen={setNewTeamDialogOpen}
      />
      <AppLayout sidebar={sidebar}>
        <Routes>
          <Route path="/overview" element={<Overview />} />
          <Route path="/metrics" element={<Metrics />} />
          <Route path="/connections" element={<Connections />} />
          <Route path="/reserved-domains" element={<ReservedDomains />} />
          <Route path="/my-account" element={<MyAccount />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/auto-signup" element={<AutoSignupSettings />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </AppLayout>
    </>
  )
}
