/**
 * Settings-school feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 *
 * NOTE: Stream hooks (useStreamList, useCreateStream, etc.) have moved to
 * @/features/streams. Import stream-related hooks and components from there.
 */

export { StreamsSection } from "./components/streams-section";
export { StreamPill } from "./components/stream-pill";

export type { Stream, StreamListResult } from "./types";
