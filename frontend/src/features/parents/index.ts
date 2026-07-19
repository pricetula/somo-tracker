/**
 * Parents feature — public API barrel.
 */

export { CreateParentForm } from "./components/create-parent-form";
export { ParentDetailView } from "./components/parent-detail";
export { LinkStudentDialog } from "./components/link-student-dialog";

export {
    useParents,
    useParentMap,
    useParentDetail,
    useMyParentProfile,
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
    ListParentsResponse,
    CreateParentPayload,
    UpdateParentPayload,
    LinkStudentPayload,
} from "./types";
