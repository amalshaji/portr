export type Team = {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string | null;
  DeletedAt: string | null;
  Name: string;
  Slug: string;
};

export type User = {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string | null;
  DeletedAt: string | null;
  Email: string;
  FirstName: string | null;
  LastName: string | null;
  GithubAvatarUrl: string | null;
  IsSuperUser: boolean;
  Teams: Team[];
};

export type TeamUser = {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string | null;
  DeletedAt: string | null;
  TeamID: number;
  Team: Team;
  UserID: number;
  User: User;
  Role: "superuser" | "admin" | "member";
  SecretKey: string;
};

type BaseSettings = {
  AllowRandomUserSignup: boolean;
  RandomUserSignupAllowedDomains: string;
  SignupRequiresInvite: boolean;
};

export type SettingsForSignup = BaseSettings;

export type Settings = BaseSettings & {
  SmtpEnabled: boolean;
  SmtpHost: string;
  SmtpPort: number;
  SmtpUsername: string;
  SmtpPassword: string;
  FromAddress: string;
  AddMemberEmailSubject: string;
  AddMemberEmailTemplate: string;
};

export type ConnectionStatus = "reserved" | "active" | "closed";

export type Connection = {
  ID: number;
  Subdomain: string;
  CreatedAt: string;
  StartedAt: string | null;
  ClosedAt: string | null;
  Status: ConnectionStatus;
  UserID: number;
  User: User;
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
