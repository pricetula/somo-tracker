"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableBulkActionContext } from "@/components/shared/data-table/types";
import { studentsColumns, getStudentId } from "../columns";
import { studentFilters } from "../filters";
import { listStudents, deleteStudents } from "../api/students.api";
import type { Student } from "../api/students.api";

export function StudentsTable() {
  const fetchPage = async (params: {
    cursor: string | null;
    limit: number;
    search?: string;
    sorting: { id: string; desc: boolean }[];
    filters: Record<string, unknown>;
  }) => listStudents(params);

  const onDelete = async (ids: string[]) => deleteStudents(ids);

  const renderRowActions = (_context: {
    row: Student;
    rowId: string;
    refresh: () => void;
  }) => {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger>
          <Button variant="ghost" size="icon-sm" className="size-6" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem>View</DropdownMenuItem>
          <DropdownMenuItem>Edit</DropdownMenuItem>
          <DropdownMenuItem>Deactivate</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  };

  const renderBulkActions = (context: DataTableBulkActionContext) => {
    return (
      <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button
          variant="destructive"
          size="sm"
          onClick={() => {
            // Handled by the shared delete flow; this is a custom action slot.
          }}
        >
          Delete {context.selectedCount}
        </Button>
        <Button variant="outline" size="sm" onClick={context.clearSelection}>
          Clear selection
        </Button>
        <Button variant="outline" size="sm" onClick={context.refresh}>
          Refresh
        </Button>
      </div>
    );
  };

  return (
    <div className="w-full">
      <DataTable<Student>
        queryKey={["students"]}
        columns={studentsColumns}
        fetchPage={fetchPage}
        getRowId={getStudentId}
        initialFilters={{ status: [] }}
        initialSorting={[{ id: "created_at", desc: true }]}
        filters={studentFilters}
        pageSize={20}
        enableRowSelection
        enableVirtualization
        renderBulkActions={renderBulkActions}
        renderRowActions={renderRowActions}
        onDelete={onDelete}
      />
    </div>
  );
}
