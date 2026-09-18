import React from "react";
import Link from "next/link";
import { Button, buttonVariants } from "@/components/ui/button";
import { TimeSlotRow } from "./time-slot-row";
import type { ClassTimetableSlotWithDetails, TimeSlotDraft } from "../types/timetable-template";
import { formatDateString } from "@/lib/utils/date";
import { Plus, Trash2 } from "lucide-react";
import { useDeleteClassTimetableSlot } from "../hooks/use-delete-class-timetable-slot";

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
    const assignmentMap = React.useMemo(() => {
        const map = new Map<string, ClassTimetableSlotWithDetails>();
        if (!assignments) return map;
        for (const a of assignments) {
            const key = `${a.time_slot_id}_${a.day_of_week}`;
            map.set(key, a);
        }
        return map;
    }, [assignments]);

    return (
        <div className="space-y-4">
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
                                          const assignment = assignmentMap.get(
                                              `${slot.id}_${dayNumber}`
                                          );
                                          if (assignment) {
                                              return (
                                                  <td
                                                      key={d}
                                                      className="border-r px-4 py-4 align-top"
                                                  >
                                                      <div className="flex flex-col space-y-2">
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
                                                                  className="text-muted-foreground block text-sm"
                                                              >
                                                                  {assignment.teacher_name}
                                                              </Link>
                                                          )) || (
                                                              <span className="text-muted-foreground text-sm">
                                                                  Unassigned teacher
                                                              </span>
                                                          )}

                                                          {assignment.room_name && (
                                                              <div className="text-muted-foreground text-xs">
                                                                  Room: {assignment.room_name}
                                                              </div>
                                                          )}
                                                          <Button
                                                              variant="ghost"
                                                              size="sm"
                                                              className="self-end"
                                                              onClick={() =>
                                                                  deleteMutation.mutate(
                                                                      assignment.id
                                                                  )
                                                              }
                                                          >
                                                              <Trash2 className="h-4 w-4" />
                                                          </Button>
                                                      </div>
                                                  </td>
                                              );
                                          }
                                          return (
                                              <td
                                                  key={d}
                                                  className="border-r px-4 py-4 text-center align-middle"
                                              >
                                                  {templateId && selectedIds?.classId ? (
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
