"use client";

import { useCreateEnrollments } from "../hooks/use-enrollments";
import { DataTable } from "@/components/shared/data-table";
import { Button } from "@/components/ui/button";
import { listUnassigned, type UnassignedStudent } from "../services/api";
import type { CreateEnrollmentRequest } from "../types/enrollment";

export function EnrollmentsPageContent({ classId }: { classId: string }) {
    const createMutation = useCreateEnrollments(classId);

    return (
        <div className="space-y-6">
            <div className="space-y-4">
                <h2 className="text-lg font-semibold">Unassigned Students</h2>
                <DataTable<
                    UnassignedStudent,
                    Record<string, never>,
                    { items: UnassignedStudent[]; total: number }
                >
                    queryKey={["unassigned-students"]}
                    queryFn={(params) =>
                        listUnassigned({ page: params.page ?? 1, limit: params.limit ?? 50 })
                    }
                    params={{}}
                    getRowId={(row) => row.student_id}
                    isCheckable
                    pageSize={50}
                    height={500}
                    columns={[
                        {
                            id: "full_name",
                            header: "Student",
                            cell: (row) => row.full_name,
                            width: "1fr",
                        },
                        {
                            id: "admission_number",
                            header: "Admission",
                            cell: (row) => row.admission_number,
                            width: "200px",
                        },
                    ]}
                    renderToolBarComponents={(selectedIds) => (
                        <Button
                            size="sm"
                            disabled={selectedIds.size === 0 || createMutation.isPending}
                            onClick={() => {
                                const today = new Date().toISOString().slice(0, 10);
                                const payload: CreateEnrollmentRequest[] = Array.from(
                                    selectedIds
                                ).map((id) => ({
                                    student_id: id,
                                    enrollment_date: today,
                                }));
                                createMutation.mutate(payload);
                            }}
                        >
                            Enroll {selectedIds.size} student{selectedIds.size !== 1 ? "s" : ""}
                        </Button>
                    )}
                    emptyState={
                        <div className="text-muted-foreground p-4 text-center">
                            No unassigned students
                        </div>
                    }
                />
            </div>
        </div>
    );
}
