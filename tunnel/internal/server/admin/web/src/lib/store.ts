import { writable } from "svelte/store";
import type {
  Connection,
  CurrentTeamUser,
  InstanceSettings,
  Team,
  TeamUser,
} from "./types";

export const currentUser = writable<CurrentTeamUser | null>(null);
export const currentUserTeams = writable<Team[]>([]);
export const instanceSettings = writable<InstanceSettings | null>(null);

export const connections = writable<Connection[]>([]);
export const connectionsLoading = writable(false);

export const users = writable<TeamUser[]>([]);
export const usersLoading = writable(false);

export const setupScript = writable<string>(
  "portr auth set --token ************************ --remote **************"
);
