/**
 * Parents feature — public API barrel.
 */

export { ParentsTable } from "./components/parents-table";
export { CreateParentForm } from "./components/create-parent-form";
export { ParentDetailView } from "./components/parent-detail";
export { LinkStudentDialog } from "./components/link-student-dialog";

export {
    useParents,
    useParentDetail,
    useCreateParent,
    useUpdateParent,
    useDeleteParent,
    useLinkStudent,
    useUnlinkStudent,
    parentKeys,
} from "./hooks/use-parents";

export type {
    Parent,
    ParentDetail,
    StudentLink,
    CreateParentPayload,
    UpdateParentPayload,
    LinkStudentPayload,
} from "./types";
