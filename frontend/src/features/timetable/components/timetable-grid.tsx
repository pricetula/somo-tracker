import React from "react";
import Link from "next/link";
import { Button, buttonVariants } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { TimeSlotRow } from "./time-slot-row";
import type { ClassTimetableSlotWithDetails, TimeSlotDraft } from "../types/timetable-template";
import { formatDateString } from "@/lib/utils/date";
import { Plus, Trash2 } from "lucide-react";
import { useDeleteClassTimetableSlot } from "../hooks/use-delete-class-timetable-slot";
import { useAssignSlot } from "../hooks/use-assign-slot";
import { useQuery } from "@tanstack/react-query";
import { getClass } from "@/features/classes/services/api";
import { SubjectsCombobox } from "@/features/curriculum/components/subjects-combobox";
import { TeachersCombobox } from "@/features/teachers/components/teachers-combobox";
import { RoomsCombobox } from "@/features/rooms/components/rooms-combobox";
import { toast } from "sonner";

type Props = {
    slots: TimeSlotDraft[];
    editable?: boolean;
    isLoading?: boolean;
    onSlotChange?: (id: string, patch: Partial<TimeSlotDraft>) => void;
    onSlotDelete?: (id: string) => void;
    onAddSlot?: () => void;
    days?: string[];
    templateId?: string;
    selectedIds?: { classId: string };
    assignments?: ClassTimetableSlotWithDetails[];
};

const DEFAULT_DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

export function TimetableGrid({
    slots,
    editable = false,
    isLoading = false,
    onSlotChange,
    onSlotDelete,
    onAddSlot,
    days = DEFAULT_DAYS,
    templateId,
    selectedIds,
    assignments = [],
}: Props) {
    const deleteMutation = useDeleteClassTimetableSlot(
        templateId || "",
        selectedIds?.classId || ""
    );
    const { data: classData } = useQuery({
        queryKey: ["class", selectedIds?.classId],
        queryFn: () => getClass(selectedIds?.classId ?? ""),
        enabled: !!selectedIds?.classId,
    });
    const gradeId = classData?.gradeId ?? "";
    const { mutate: assignMutate, isPending: isAssignPending } = useAssignSlot();
    const [bulkMode, setBulkMode] = React.useState(false);
    const [selectedKeys, setSelectedKeys] = React.useState<Set<string>>(new Set());
    const [bulkSubjectId, setBulkSubjectId] = React.useState("");
    const [bulkTeacherId, setBulkTeacherId] = React.useState("");
    const [bulkRoomId, setBulkRoomId] = React.useState("");

    const assignmentMap = React.useMemo(() => {
        const map = new Map<string, ClassTimetableSlotWithDetails>();
        if (!assignments) return map;
        for (const a of assignments) {
            const timeSlotId = (a.time_slot_id || "").toLowerCase();
            const day = a.day_of_week === 0 ? 7 : a.day_of_week;
            const key = `${timeSlotId}_${day}`;
            map.set(key, a);
        }
        return map;
    }, [assignments]);

    return (
        <div className="space-y-4">
            {templateId && selectedIds?.classId && (
                <div className="flex items-center justify-between">
                    <Button
                        type="button"
                        variant={bulkMode ? "default" : "outline"}
                        onClick={() => {
                            setBulkMode(!bulkMode);
                            setSelectedKeys(new Set());
                        }}
                    >
                        {bulkMode ? "Exit Bulk Assign" : "Bulk Assign"}
                    </Button>
                    {bulkMode && selectedKeys.size > 0 && (
                        <div className="text-muted-foreground text-sm">
                            {selectedKeys.size} slot{selectedKeys.size > 1 ? "s" : ""} selected
                        </div>
                    )}
                </div>
            )}
            {bulkMode && templateId && selectedIds?.classId && (
                <div className="space-y-3 rounded-md border p-4">
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                        <SubjectsCombobox
                            gradeId={gradeId}
                            value={bulkSubjectId}
                            onChange={setBulkSubjectId}
                            placeholder="Subject"
                        />
                        <TeachersCombobox
                            value={bulkTeacherId}
                            onChange={setBulkTeacherId}
                            placeholder="Teacher"
                        />
                        <RoomsCombobox
                            value={bulkRoomId}
                            onChange={setBulkRoomId}
                            placeholder="Room (optional)"
                        />
                    </div>
                    <div className="flex justify-end gap-2">
                        <Button
                            variant="outline"
                            onClick={() => {
                                setSelectedKeys(new Set());
                                setBulkSubjectId("");
                                setBulkTeacherId("");
                                setBulkRoomId("");
                            }}
                        >
                            Clear
                        </Button>
                        <Button
                            disabled={
                                !bulkSubjectId ||
                                !bulkTeacherId ||
                                selectedKeys.size === 0 ||
                                isAssignPending
                            }
                            onClick={() => {
                                const promises = Array.from(selectedKeys).map((key) => {
                                    const [tsId, dayStr] = key.split("_");
                                    const dayNumber = Number(dayStr);
                                    return new Promise<void>((resolve) => {
                                        assignMutate(
                                            {
                                                class_room_id: selectedIds.classId,
                                                day_of_week: dayNumber,
                                                time_slot_id: tsId,
                                                subject_id: bulkSubjectId,
                                                teacher_membership_id: bulkTeacherId,
                                                room_id: bulkRoomId || undefined,
                                            },
                                            {
                                                onSuccess: () => resolve(),
                                                onError: () => resolve(),
                                            }
                                        );
                                    });
                                });
                                Promise.all(promises).then(() => {
                                    toast.success(
                                        `Assigned ${selectedKeys.size} slot${selectedKeys.size > 1 ? "s" : ""}`
                                    );
                                    setBulkMode(false);
                                    setSelectedKeys(new Set());
                                });
                            }}
                        >
                            Assign to {selectedKeys.size} slot{selectedKeys.size !== 1 ? "s" : ""}
                        </Button>
                    </div>
                </div>
            )}
            <div className="overflow-x-auto rounded-md border">
                <table className="w-full">
                    <thead>
                        <tr className="border-b">
                            <th className="bg-background text-muted-foreground sticky top-0 left-0 z-30 w-96 border-r px-4 py-3 text-left font-medium">
                                Time Slot
                            </th>
                            {days.map((d) => (
                                <th
                                    key={d}
                                    className="bg-background sticky top-0 z-20 min-w-56 border-r px-4 py-3 text-left font-medium"
                                >
                                    {d}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {isLoading
                            ? Array.from({ length: 3 }).map((_, i) => (
                                  <tr
                                      key={`skeleton-${i}`}
                                      className="animate-pulse border-b align-top"
                                  >
                                      <td className="bg-muted/30 sticky left-0 z-10 w-96 border-r px-4 py-3 align-top">
                                          <div className="min-h-31 w-48 space-y-1 p-3 pt-4">
                                              <div className="bg-muted mb-4 h-4 w-32 rounded" />
                                              <div className="bg-muted h-3 w-24 rounded" />
                                          </div>
                                      </td>
                                      {days.map((d) => (
                                          <td
                                              key={d}
                                              className="bg-muted/20 min-w-56 border-r px-4 py-4 align-top"
                                          />
                                      ))}
                                  </tr>
                              ))
                            : slots.map((slot) => (
                                  <tr key={slot.id} className="border-b align-top">
                                      <td className="bg-background sticky left-0 z-10 border-r">
                                          {editable && onSlotChange && onSlotDelete ? (
                                              <TimeSlotRow
                                                  slot={slot}
                                                  onChange={(patch) => onSlotChange(slot.id, patch)}
                                                  onDelete={() => onSlotDelete(slot.id)}
                                                  canDelete={slots.length > 1}
                                              />
                                          ) : (
                                              <div className="min-h-31 w-48 space-y-1 p-3 pt-4">
                                                  <div className="mb-4 font-medium">
                                                      {slot.name}
                                                  </div>
                                                  <div className="text-muted-foreground text-xs">
                                                      {`${formatDateString(slot.start_time, {
                                                          inputFormat: "HH:mm",
                                                          outputFormat: "HH:mm a",
                                                      })} — ${formatDateString(slot.end_time, {
                                                          inputFormat: "HH:mm",
                                                          outputFormat: "HH:mm a",
                                                      })}`}
                                                  </div>
                                              </div>
                                          )}
                                      </td>
                                      {days.map((d, dayIdx) => {
                                          const dayNumber = dayIdx + 1;
                                          if (!slot.is_instructional) {
                                              return (
                                                  <td
                                                      key={d}
                                                      className="bg-row-disabled border-r p-4 text-center align-middle"
                                                  >
                                                      {slot.name}
                                                  </td>
                                              );
                                          }
                                          const timeSlotId = (slot.id || "").toLowerCase();
                                          const assignment = assignmentMap.get(
                                              `${timeSlotId}_${dayNumber}`
                                          );
                                          if (assignment) {
                                              return (
                                                  <td
                                                      key={d}
                                                      className="border-r px-4 py-4 align-top"
                                                  >
                                                      <div className="flex justify-between">
                                                          <div className="flex flex-col items-start space-y-2">
                                                              {(assignment.subject_name && (
                                                                  <Link
                                                                      href={`/subjects/${assignment.subject_id}`}
                                                                      className="block font-medium"
                                                                  >
                                                                      {assignment.subject_name}
                                                                  </Link>
                                                              )) || (
                                                                  <span className="font-medium">
                                                                      Unassigned subject
                                                                  </span>
                                                              )}

                                                              {(assignment.teacher_name && (
                                                                  <Link
                                                                      href={`/teachers/${assignment.teacher_membership_id}`}
                                                                      className="text-muted-foreground block"
                                                                  >
                                                                      {assignment.teacher_name}
                                                                  </Link>
                                                              )) || (
                                                                  <span className="text-muted-foreground">
                                                                      Unassigned teacher
                                                                  </span>
                                                              )}

                                                              {assignment.room_name && (
                                                                  <div className="text-muted-foreground text-xs">
                                                                      Room: {assignment.room_name}
                                                                  </div>
                                                              )}
                                                              <div className="mt-2 flex gap-2">
                                                                  <Link
                                                                      href={`/timetable/${templateId}/assign?classId=${selectedIds?.classId}&day=${dayNumber}&slot=${slot.id}&subjectId=${assignment.subject_id}&teacherId=${assignment.teacher_membership_id}&roomId=${assignment.room_id ?? ""}&assignmentId=${assignment.id}`}
                                                                      className={buttonVariants({
                                                                          variant: "outline",
                                                                          size: "xs",
                                                                      })}
                                                                  >
                                                                      Edit
                                                                  </Link>
                                                                  <Button
                                                                      variant="destructive"
                                                                      size="xs"
                                                                      onClick={() =>
                                                                          deleteMutation.mutate(
                                                                              assignment.id
                                                                          )
                                                                      }
                                                                  >
                                                                      <Trash2 className="h-4 w-4" />
                                                                      <span>Delete</span>
                                                                  </Button>
                                                              </div>
                                                          </div>
                                                          <div
                                                              className="h-10 w-1"
                                                              style={{
                                                                  backgroundColor:
                                                                      assignment?.subject_color ??
                                                                      "",
                                                              }}
                                                          />
                                                      </div>
                                                  </td>
                                              );
                                          }
                                          const cellKey = `${timeSlotId}_${dayNumber}`;
                                          const isSelected = selectedKeys.has(cellKey);
                                          return (
                                              <td
                                                  key={d}
                                                  className="border-r px-4 py-4 text-center align-middle"
                                              >
                                                  {bulkMode ? (
                                                      <Checkbox
                                                          checked={isSelected}
                                                          onCheckedChange={(checked) => {
                                                              const next = new Set(selectedKeys);
                                                              if (checked) {
                                                                  next.add(cellKey);
                                                              } else {
                                                                  next.delete(cellKey);
                                                              }
                                                              setSelectedKeys(next);
                                                          }}
                                                      />
                                                  ) : templateId && selectedIds?.classId ? (
                                                      <Link
                                                          href={`/timetable/${templateId}/assign?classId=${selectedIds.classId}&day=${dayNumber}&slot=${slot.id}`}
                                                          className={buttonVariants({
                                                              variant: "outline",
                                                              size: "sm",
                                                          })}
                                                      >
                                                          <Plus />
                                                          <span>Assign</span>
                                                      </Link>
                                                  ) : null}
                                              </td>
                                          );
                                      })}
                                  </tr>
                              ))}
                    </tbody>
                </table>
            </div>

            {editable && onAddSlot && (
                <div className="flex items-center justify-between">
                    <div>
                        <Button type="button" variant="outline" onClick={onAddSlot}>
                            + Add Time Slot
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}
