import { writable } from "svelte/store";
import type {
  Connection,
  CurrentTeamUser,
  Invite,
  InstanceSettings,
  Team,
  TeamUser,
  TeamSettings,
} from "./types";

export const currentUser = writable<CurrentTeamUser | null>(null);
export const currentUserTeams = writable<Team[]>([]);
export const instanceSettings = writable<InstanceSettings | null>(null);
export const teamSettings = writable<TeamSettings | null>(null);

export const connections = writable<Connection[]>([]);
export const connectionsLoading = writable(false);

export const users = writable<TeamUser[]>([]);
export const usersLoading = writable(false);

export const invites = writable<Invite[]>([]);
export const invitesLoading = writable(false);

export const setupScript = writable<string>(
  "./portr auth set --token ************************ --remote **************"
);
