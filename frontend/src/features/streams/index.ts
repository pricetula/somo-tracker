/**
 * Streams feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { AddStreamForm } from "./components/add-stream-form";
export { StreamCombobox } from "./components/stream-combobox";

export {
    useStreamList,
    useCreateStream,
    useUpdateStream,
    useDeleteStream,
    streamKeys,
} from "./hooks/use-streams";

export type { StreamComboboxProps } from "./components/stream-combobox";
export type { Stream, StreamListResult } from "./types";
