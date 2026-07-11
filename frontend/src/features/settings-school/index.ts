/**
 * Settings-school feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { StreamsSection } from "./components/streams-section";
export { StreamPill } from "./components/stream-pill";

export {
    useStreamList,
    useCreateStream,
    useUpdateStream,
    useDeleteStream,
    streamKeys,
} from "./hooks/use-streams";

export type { Stream, StreamListResult } from "./types";
