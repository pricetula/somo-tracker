/**
 * Curriculum feature — public API barrel.
 */

export { LearningAreasTable } from "./components/learning-areas-table";
export { CurriculumTree } from "./components/curriculum-tree";
export { CreateLearningAreaDialog } from "./components/create-learning-area-dialog";

export {
    useLearningAreas,
    useLearningAreaTree,
    useCreateLearningArea,
    useUpdateLearningArea,
    useDeleteLearningArea,
    useCreateStrand,
    useUpdateStrand,
    useDeleteStrand,
    useCreateSubStrand,
    useUpdateSubStrand,
    useDeleteSubStrand,
    useCreatePerformanceIndicator,
    useUpdatePerformanceIndicator,
    useDeletePerformanceIndicator,
    curriculumKeys,
} from "./hooks/use-curriculum";

export type {
    LearningArea,
    Strand,
    SubStrand,
    PerformanceIndicator,
    StrandTree,
    SubStrandTree,
    LearningAreaTree,
    ListLearningAreasResponse,
} from "@/lib/api/curriculum";
