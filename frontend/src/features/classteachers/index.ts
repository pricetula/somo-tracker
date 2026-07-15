/**
 * Class Teachers feature — public API barrel.
 */

export { ClassTeacherList } from "./components/class-teacher-list";
export { AssignTeacherDialog } from "./components/assign-teacher-dialog";

export {
    useClassTeachersByClass,
    useClassTeachersByTeacher,
    useCreateClassTeacher,
    useDeleteClassTeacher,
    classTeacherKeys,
} from "./hooks/use-classteachers";

export type { ClassTeacher, ClassTeacherListResponse, CreateClassTeacherPayload } from "./types";
