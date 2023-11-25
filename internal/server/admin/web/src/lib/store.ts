import { writable } from "svelte/store";
import type { User } from "./types";

export const currentUser = writable<User | null>(null);
export const connections = writable([]);
export const connectionsLoading = writable(false);
