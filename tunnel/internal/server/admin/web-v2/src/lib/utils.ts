import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import { toast } from "sonner"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const updateQueryParam = (
  urlParams: URLSearchParams,
  key: string,
  value: string
) => {
  urlParams.set(key, value)
  const newUrl = `${window.location.pathname}?${urlParams.toString()}`
  window.history.pushState({}, "", newUrl)
}

export const copyCodeToClipboard = (code: string) => {
  navigator.clipboard.writeText(code)
  toast.success("Code copied to clipboard")
}
