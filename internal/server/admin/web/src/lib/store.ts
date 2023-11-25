import { writable } from "svelte/store";
import type { Connection, Invite, Settings, User } from "./types";

export const currentUser = writable<User | null>(null);
export const settings = writable<Settings | null>(null);

export const connections = writable<Connection[]>([]);
export const connectionsLoading = writable(false);

export const users = writable<User[]>([]);
export const usersLoading = writable(false);

export const invites = writable<Invite[]>([]);
export const invitesLoading = writable(false);
