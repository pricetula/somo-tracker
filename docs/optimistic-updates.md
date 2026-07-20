# Optimistic Updates — Audit & Implementation Tasks

> **Status:** Audit complete — 4/78 mutations have optimistic updates.
> **Owner:** Platform team
> **Priority:** High (UI responsiveness)

---

## Golden Pattern

When adding an optimistic update, follow the existing pattern used by `useToggleTeacherActive`:

```ts
// ✅ Canonical optimistic update + rollback
onMutate: async (variables) => {
    await queryClient.cancelQueries({ queryKey: keys.all });
    const previousQueries = queryClient.getQueriesData<ResponseType>({
        queryKey: keys.all,
    });

    queryClient.setQueriesData<ResponseType>({ queryKey: keys.all }, (old) => {
        if (!old) return old;
        return {
            ...old,
            items: old.items.map((item) =>
                item.id === variables.id ? { ...item, /* updated fields */ } : item
            ),
        };
    });

    return { previousQueries };
},
onError: (err, _vars, context) => {
    if (context?.previousQueries) {
        for (const [key, data] of context.previousQueries) {
            queryClient.setQueryData(key, data);
        }
    }
    toast.error(getErrorMessage(err));
},
onSettled: () => {
    queryClient.invalidateQueries({ queryKey: keys.all });
},
```

---

## Task List

### Phase 1 — Deletes (highest perceived latency)

| # | Hook | Feature | File | Complexity |
|---|------|---------|------|------------|
| 1 | `useDeleteStudent` | Students | `features/students/hooks/use-students.ts` | Low — single ID, filter from list cache |
| 2 | `useDeleteTeacher` | Teachers | `features/teachers/hooks/use-teachers.ts` | Low |
| 3 | `useDeleteAdmin` | Admins | `features/admin/hooks/use-admins.ts` | Low |
| 4 | `useDeleteNurse` | Nurses | `features/nurses/hooks/use-nurses.ts` | Low |
| 5 | `useDeleteFinanceStaff` | Finance | `features/finance/hooks/use-finance.ts` | Low |
| 6 | `useDeleteParent` | Parents | `features/parents/hooks/use-parents.ts` | Low |
| 7 | `useDeleteClasses` | Classes | `features/classes/hooks/use-classes.ts` | Low — bulk IDs |
| 8 | `useDeleteBehaviorNote` | Behavior | `features/behavior/hooks/use-behavior.ts` | Low |
| 9 | `useDeleteMedicalIncident` | Health | `features/health/hooks/use-health.ts` | Low |
| 10 | `useDeleteStream` | Streams | `features/streams/hooks/use-streams.ts` | Low |
| 11 | `useDeleteSchool` | School | `features/school/hooks/use-schools.ts` | Low |
| 12 | `useDeleteScaleProfile` | Assessments | `features/assessments/hooks/use-assessments.ts` | Low |
| 13 | `useDeleteSession` | Assessments | `features/assessments/hooks/use-assessments.ts` | Low |
| 14 | `useDeleteWeightConfig` | Assessments | `features/assessments/hooks/use-assessments.ts` | Low |
| 15 | `useDeleteLearningArea` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium — tree cache |
| 16 | `useDeleteStrand` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium — invalidate parent tree |
| 17 | `useDeleteSubStrand` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 18 | `useDeletePerformanceIndicator` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 19 | `useDeleteFeeCategory` | Fee Categories | `features/fee-categories/hooks/use-fee-categories.ts` | Low |
| 20 | `useDeleteFeeTemplate` | Fee Templates | `features/fee-templates/hooks/use-fee-templates.ts` | Low |
| 21 | `useDeleteClassTeacher` | Class Teachers | `features/classteachers/hooks/use-classteachers.ts` | Low |
| 22 | `useDeleteTimeBlock` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Low |
| 23 | `useDeleteTimeBlocksByName` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Medium — batch delete by name |
| 24 | `useDeleteSlot` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Low |
| 25 | `useDeleteAcademicYear` | Academic Years | `features/academic-years/hooks/use-academic-years.ts` | Low |

### Phase 2 — Toggles (missing `onMutate`)

| # | Hook | Feature | File | Notes |
|---|------|---------|------|-------|
| 26 | `useToggleScaleProfile` | Assessments | `features/assessments/hooks/use-assessments.ts` | Mirror pattern from teacher/nurse/admins |

### Phase 3 — Creates (pre-insert into list cache)

| # | Hook | Feature | File | Complexity |
|---|------|---------|------|------------|
| 27 | `useCreateStudent` | Students | `features/students/hooks/use-student-detail.ts` | Medium — needs response shape |
| 28 | `useCreateStudents` | Students (batch) | `features/students/hooks/use-student-detail.ts` | High — batch insert |
| 29 | `useCreateEnrollment` | Students | `features/students/hooks/use-student-detail.ts` | Low — prepend to detail |
| 30 | `useCreateParent` | Parents | `features/parents/hooks/use-parents.ts` | Medium |
| 31 | `useCreateBehaviorCategory` | Behavior | `features/behavior/hooks/use-behavior.ts` | Low |
| 32 | `useCreateBehaviorNote` | Behavior | `features/behavior/hooks/use-behavior.ts` | Low |
| 33 | `useCreateMedicalIncident` | Health | `features/health/hooks/use-health.ts` | Medium |
| 34 | `useCreateStream` | Streams | `features/streams/hooks/use-streams.ts` | Low |
| 35 | `useCreateSchool` | School | `features/school/hooks/use-schools.ts` | Low |
| 36 | `useCreateScaleProfile` | Assessments | `features/assessments/hooks/use-assessments.ts` | Medium |
| 37 | `useCreateSession` | Assessments | `features/assessments/hooks/use-assessments.ts` | Medium |
| 38 | `useCreateWeightConfig` | Assessments | `features/assessments/hooks/use-assessments.ts` | Low |
| 39 | `useCreateLearningArea` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 40 | `useCreateStrand` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 41 | `useCreateSubStrand` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 42 | `useCreatePerformanceIndicator` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 43 | `useCreateFeeCategory` | Fee Categories | `features/fee-categories/hooks/use-fee-categories.ts` | Low |
| 44 | `useCreateFeeTemplate` | Fee Templates | `features/fee-templates/hooks/use-fee-templates.ts` | Low |
| 45 | `useCreateClassTeacher` | Class Teachers | `features/classteachers/hooks/use-classteachers.ts` | Low |
| 46 | `useCreateTimeBlock` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Medium |
| 47 | `useBatchCreateTimeBlocks` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | High — batch |
| 48 | `useCreateSlot` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Medium |
| 49 | `useBatchCreateSlots` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | High — batch |
| 50 | `useCreateAcademicYear` | Academic Years | `features/academic-years/hooks/use-academic-years.ts` | Low |
| 51 | `useCreateTerm` | Academic Years | `features/academic-years/hooks/use-academic-years.ts` | Low |
| 52 | `useGenerateInvoice` | Invoices | `features/finance-invoices/hooks/use-finance-invoices.ts` | Medium |
| 53 | `useRecordPayment` | Invoices | `features/finance-invoices/hooks/use-finance-invoices.ts` | Medium |
| 54 | `useUpsertHealthProfile` | Health | `features/health/hooks/use-health.ts` | Medium |

### Phase 4 — Updates (patch in-place in list/detail cache)

| # | Hook | Feature | File | Complexity |
|---|------|---------|------|------------|
| 55 | `useUpdateStudent` | Students | `features/students/hooks/use-student-detail.ts` | Medium — list + detail caches |
| 56 | `useUpdateTeacher` | Teachers | `features/teachers/hooks/use-teachers.ts` | Medium |
| 57 | `useUpdateAdmin` | Admins | `features/admin/hooks/use-admins.ts` | Medium |
| 58 | `useUpdateNurse` | Nurses | `features/nurses/hooks/use-nurses.ts` | Medium |
| 59 | `useUpdateFinance` | Finance | `features/finance/hooks/use-finance.ts` | Medium |
| 60 | `useUpdateParent` | Parents | `features/parents/hooks/use-parents.ts` | Medium |
| 61 | `useUpdateBehaviorCategory` | Behavior | `features/behavior/hooks/use-behavior.ts` | Medium |
| 62 | `useReviewBehaviorNote` | Behavior | `features/behavior/hooks/use-behavior.ts` | Medium — status change |
| 63 | `useUpdateMedicalIncident` | Health | `features/health/hooks/use-health.ts` | Medium |
| 64 | `useUpdateStream` | Streams | `features/streams/hooks/use-streams.ts` | Low |
| 65 | `useUpdateSchool` | School | `features/school/hooks/use-schools.ts` | Low |
| 66 | `useSetActiveSchool` | School | `features/school/hooks/use-schools.ts` | Medium — cross-cache (auth) |
| 67 | `useUpdateLearningArea` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 68 | `useUpdateStrand` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 69 | `useUpdateSubStrand` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 70 | `useUpdatePerformanceIndicator` | Curriculum | `features/curriculum/hooks/use-curriculum.ts` | Medium |
| 71 | `useUpdateFeeCategory` | Fee Categories | `features/fee-categories/hooks/use-fee-categories.ts` | Low |
| 72 | `useUpdateFeeTemplate` | Fee Templates | `features/fee-templates/hooks/use-fee-templates.ts` | Low |
| 73 | `useUpdateTimeBlock` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Medium |
| 74 | `useUpdateSlot` | Timetable | `features/timetable-structure/hooks/use-timetable-structure.ts` | Medium |
| 75 | `useUpdateAcademicYear` | Academic Years | `features/academic-years/hooks/use-academic-years.ts` | Medium |
| 76 | `useSetCurrentYear` | Academic Years | `features/academic-years/hooks/use-academic-years.ts` | Medium |
| 77 | `useUpdateTerm` | Academic Years | `features/academic-years/hooks/use-academic-years.ts` | Medium |
| 78 | `useWaiveInvoice` | Invoices | `features/finance-invoices/hooks/use-finance-invoices.ts` | Medium |

### Phase 5 — Special Cases

| # | Hook | Feature | Notes |
|---|------|---------|-------|
| 79 | `useLinkStudent` | Parents | Mutates parent detail + student parent list |
| 80 | `useUnlinkStudent` | Parents | Same cross-cache mutation |
| 81 | `useBatchEnrollStudents` | Students | Bulk — needs students + classes cache update |
| 82 | `useSubmitSession` | Assessments | Status transition (DRAFT → PENDING_APPROVAL) |
| 83 | `useApproveSession` | Assessments | Status transition |
| 84 | `useRejectSession` | Assessments | Status transition |
| 85 | `useBulkSetScaleRanges` | Assessments | Bulk ranges under a profile |
| 86 | `useBulkUpsertScores` | Assessments | Bulk scores under a session |
| 87 | `useBulkUpsertOutcomeGrades` | Assessments | Bulk grades under a session |
| 88 | `useReplicateDay` | Timetable | Batch day replication |
| 89 | `useCancelImportJob` | Import Jobs | Status update on job detail |

---

## Reference: Existing Optimistic Updates

These 4 hooks already have the pattern and serve as implementation reference:

| Hook | File |
|------|------|
| `useToggleTeacherActive` | `features/teachers/hooks/use-teachers.ts` |
| `useToggleAdminActive` | `features/admin/hooks/use-admins.ts` |
| `useToggleNurseActive` | `features/nurses/hooks/use-nurses.ts` |
| `useToggleFinanceActive` | `features/finance/hooks/use-finance.ts` |
