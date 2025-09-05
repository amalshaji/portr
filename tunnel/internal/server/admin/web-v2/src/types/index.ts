export interface User {
  id: string
  email: string
  first_name?: string
  last_name?: string
  is_superuser: boolean
  github_user?: {
    github_avatar_url: string
  }
}

export interface Team {
  id: number
  name: string
  slug: string
}

export interface CurrentTeamUser {
  id: number
  secret_key: string
  role: string
  user: User
}

export interface TeamUser {
  id: number
  created_at: string
  updated_at: string | null
  deleted_at: string | null
  team: Team
  user: User
  role: "admin" | "member"
  secret_key: string
}

export interface InstanceSettings {
  smtp_enabled: boolean
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  from_address: string
  add_user_email_subject: string
  add_user_email_body: string
}

export interface AuthConfig {
  is_first_signup: boolean
  github_auth_enabled: boolean
}

export type ConnectionStatus = "reserved" | "active" | "closed"
export type ConnectionType = "http" | "tcp"

export interface Connection {
  id: number
  type: ConnectionType
  port: number
  subdomain: string
  created_at: string
  started_at: string | null
  closed_at: string | null
  status: ConnectionStatus
  created_by: TeamUser
}

export interface DashboardStats {
  activeConnections: number
  totalUsers: number
}

export interface SystemStats {
  memoryUsedMB: number
  memoryTotalMB: number
  systemMemoryUsedGB: number
  systemMemoryTotalGB: number
  systemMemoryUsagePercent: number
  cpuUsagePercent: number
  numCpu: number
  goroutines: number
  hostname: string
  os: string
  architecture: string
  server_start_time?: string
}
