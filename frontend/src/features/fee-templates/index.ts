/**
 * Fee Templates feature — public API barrel.
 */

export { FeeTemplatesList } from "./components/fee-templates-list";

export {
    useFeeTemplates,
    useCreateFeeTemplate,
    useUpdateFeeTemplate,
    useDeleteFeeTemplate,
    feeTemplateKeys,
} from "./hooks/use-fee-templates";

export type { FeeTemplate, CreateFeeTemplatePayload, UpdateFeeTemplatePayload } from "./types";
