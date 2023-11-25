export type User = {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string;
  Email: string;
  FirstName: string | null;
  LastName: string | null;
  Role: "superuser" | "admin" | "member";
  avatarUrl: string | null;
};
