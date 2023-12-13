import { writable } from "svelte/store";
import type {
  Connection,
  Invite,
  ServerAddress,
  Settings,
  SettingsForSignup,
  TeamUser,
  User,
} from "./types";

export const currentUser = writable<User | null>(null);
export const currentTeamUser = writable<TeamUser | null>(null);
export const settings = writable<Settings | null>(null);

export const connections = writable<Connection[]>([]);
export const connectionsLoading = writable(false);

export const users = writable<User[]>([]);
export const usersLoading = writable(false);

export const invites = writable<Invite[]>([]);
export const invitesLoading = writable(false);

export const settingsForSignup = writable<SettingsForSignup | null>(null);

export const serverAddress = writable<ServerAddress | null>(null);
