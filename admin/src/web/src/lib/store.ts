import { writable } from "svelte/store";
import type {
  Connection,
  CurrentTeamUser,
  Invite,
  ServerAddress,
  Settings,
  SettingsForSignup,
  Team,
  TeamUser,
} from "./types";

export const currentUser = writable<CurrentTeamUser | null>(null);
export const currentUserTeams = writable<Team[]>([]);
export const settings = writable<Settings | null>(null);

export const connections = writable<Connection[]>([]);
export const connectionsLoading = writable(false);

export const users = writable<TeamUser[]>([]);
export const usersLoading = writable(false);

export const invites = writable<Invite[]>([]);
export const invitesLoading = writable(false);

export const settingsForSignup = writable<SettingsForSignup | null>(null);

export const setupScript = writable<string>(
  "./portr auth set --token ************************ --remote **************"
);
