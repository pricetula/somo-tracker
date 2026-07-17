/**
 * Curriculum feature — public API barrel.
 */

export { CurriculumTree } from "./components/curriculum-tree";
export { LearningAreaCombobox } from "./components/learning-area-combobox";
export { CreateLearningAreaDialog } from "./components/create-learning-area-dialog";
export { CreateStrandDialog } from "./components/create-strand-dialog";
export { CreateSubStrandDialog } from "./components/create-sub-strand-dialog";
export { CreateIndicatorDialog } from "./components/create-indicator-dialog";

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
