export type User = {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string | null;
  DeletedAt: string | null;
  Email: string;
  FirstName: string | null;
  LastName: string | null;
  Role: "superuser" | "admin" | "member";
  avatarUrl: string | null;
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
  InvitedByUserID: number;
  InvitedByUser: User;
};
