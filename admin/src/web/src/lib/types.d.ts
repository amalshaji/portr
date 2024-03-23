export type Team = {
  id: number;
  name: string;
  slug: string;
};

export type CurrentTeamUser = {
  id: number;
  secret_key: String;
  role: String;

  user: CurrentUser;
};

export type CurrentUser = {
  email: string;
  first_name: string?;
  last_name: string?;
  is_superuser: boolean;

  github_user: CurrentGithubUser?;
};

export type CurrentGithubUser = {
  github_avatar_url: string;
};

export type TeamUser = {
  id: number;
  created_at: string;
  updated_at: string | null;
  deleted_at: string | null;
  team: Team;
  user: CurrentUser;
  role: "admin" | "member";
  secret_key: string;
};

export type InstanceSettings = {
  smtp_enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_username: string;
  smtp_password: string;
  from_address: string;
  add_user_email_subject: string;
  add_user_email_body: string;
};

export type ConnectionStatus = "reserved" | "active" | "closed";

export type ConnectionType = "http" | "tcp";

export type Connection = {
  id: number;
  type: ConnectionType;
  port: number;
  subdomain: string;
  created_at: string;
  started_at: string | null;
  closed_at: string | null;
  status: ConnectionStatus;
  created_by: TeamUser;
};

export type Invite = {
  Email: string;
  Role: "admin" | "member";
  Status: "pending" | "accepted" | "expired";
  InvitedByEmail: string;
  InvitedByFirstName: string;
  InvitedByLastName: string;
};

export type ServerAddress = {
  AdminUrl: string;
  SshUrl: string;
};
