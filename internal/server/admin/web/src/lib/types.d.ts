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
  UserInviteEmailTemplate: string;
};

export type Connection = {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string | null;
  DeletedAt: string | null;
  Subdomain: string;
  ClosedAt: string | null;
  UserID: number;
  User: User;
};

export type Invite = {
  Email: string;
  Role: "admin" | "member";
  Status: "pending" | "accepted" | "expired";
  InvitedByTeamUserID: number;
  InvitedByTeamUser: TeamUser;
};

export type ServerAddress = {
  AdminUrl: string;
  SshUrl: string;
};
