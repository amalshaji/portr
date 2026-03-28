import { writable } from "svelte/store";
import type { Request } from "./types";

export const currentRequest = writable<Request | null>(null);
