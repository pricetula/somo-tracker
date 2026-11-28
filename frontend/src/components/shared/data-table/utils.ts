export { cn } from "@/lib/utils";

/**
 * Returns a user-safe error message. Raw Error messages may contain internal
 * stack traces, database details, or other implementation details that should
 * not be surfaced to end users.
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    // Common safe messages; anything else is genericised.
    const safe = [
      "Failed to load records.",
      "Failed to delete records.",
      "Network error.",
      "Unauthorized.",
      "Forbidden.",
      "Not found.",
    ];
    if (safe.includes(error.message)) {
      return error.message;
    }
    return "An unexpected error occurred.";
  }
  if (typeof error === "string") {
    return error;
  }
  return "An unexpected error occurred.";
}

export function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}
