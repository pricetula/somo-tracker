/**
 * Settings-school feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 *
 * NOTE: Stream types, hooks, and combobox have moved to @/features/streams.
 * Import stream-related types, hooks, and components from there instead.
 * This barrel still exports the legacy StreamPill and StreamsSection components.
 */

export { StreamsSection } from "./components/streams-section";
export { StreamPill } from "./components/stream-pill";
