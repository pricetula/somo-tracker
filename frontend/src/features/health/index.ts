/**
 * Health feature — public API barrel.
 */

export { IncidentList } from "./components/incident-list";
export { CreateIncidentDialog } from "./components/create-incident-dialog";
export { StudentHealthView } from "./components/student-health-view";

export {
    useMedicalIncidents,
    useMedicalIncident,
    useCreateMedicalIncident,
    useDeleteMedicalIncident,
    useStudentHealth,
    useUpsertHealthProfile,
    healthKeys,
} from "./hooks/use-health";

export type {
    MedicalIncident,
    StudentHealthProfile,
    StudentHealthResponse,
    CreateMedicalIncidentPayload,
    UpsertHealthProfilePayload,
} from "./types";
