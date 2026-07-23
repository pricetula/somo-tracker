/**
 * Fee Categories feature — public API barrel.
 */

export { FeeCategoriesList } from "./components/fee-categories-list";

export {
    useFeeCategories,
    useCreateFeeCategory,
    useUpdateFeeCategory,
    useDeleteFeeCategory,
    feeCategoryKeys,
} from "./hooks/use-fee-categories";

export type { FeeCategory, CreateFeeCategoryPayload, UpdateFeeCategoryPayload } from "./types";
