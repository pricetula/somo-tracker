/**
 * Curriculum feature — public API barrel.
 */

export { SeedDefaultButton } from "./components/seed-default-button";
export { CurriculumPage } from "./components/curriculum-page";
export { CurriculumDetailPage } from "./components/curriculum-detail-page";
export { StrandDetailPage } from "./components/strand-detail-page";
export { CurriculumTree } from "./components/curriculum-tree";
export { LearningAreaCombobox } from "./components/learning-area-combobox";
export { CreateLearningAreaDialog } from "./components/create-learning-area-dialog";
export { CreateStrandDialog } from "./components/create-strand-dialog";
export { CreateSubStrandDialog } from "./components/create-sub-strand-dialog";
export { CreateIndicatorDialog } from "./components/create-indicator-dialog";

export {
    useLearningAreas,
    useLearningAreaTree,
    useSeedDefaultCBC,
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
