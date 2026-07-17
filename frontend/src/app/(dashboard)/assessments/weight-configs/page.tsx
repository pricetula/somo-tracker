/**
 * Weight Configurations page — manage KNEC weight profiles.
 *
 * Lists KNEC national weighting formulas with filters. SCHOOL_ADMIN can
 * create new weight configs defining how assessment types contribute to
 * target exam placement scores (KPSEA, KJSEA, KSSEA).
 */

import { WeightConfigsList } from "@/features/assessments";

export default function WeightConfigsPage() {
    return <WeightConfigsList />;
}
