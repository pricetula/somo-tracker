/**
 * Teacher delivery feature — public API barrel.
 *
 * Import only from this barrel, never from internal paths.
 */

// Components
export { TeacherComplianceChart } from "./components/teacher-compliance-chart";

// Hooks
export {
    useTeacherDeliveryBreakdown,
    useCurrentTermId,
} from "./hooks/use-teacher-delivery-breakdown";
