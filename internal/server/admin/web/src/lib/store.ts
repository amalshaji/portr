import { writable } from "svelte/store";
import type { Settings, User } from "./types";

export const currentUser = writable<User | null>(null);
export const connections = writable([]);
export const connectionsLoading = writable(false);
export const settings = writable<Settings | null>(null);
