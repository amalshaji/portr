import { writable } from "svelte/store";

export const connections = writable([]);
export const connectionsLoading = writable(false);
