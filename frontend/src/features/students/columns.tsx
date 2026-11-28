import type { ColumnDef } from "@tanstack/react-table";
import type { Student } from "./api/students.api";

export const studentsColumns: ColumnDef<any, Student, unknown>[] = [
  {
    id: "first_name",
    accessorKey: "first_name",
    header: "First name",
    meta: { width: "150px" },
  },
  {
    id: "last_name",
    accessorKey: "last_name",
    header: "Last name",
    meta: { width: "150px" },
  },
  { id: "email", accessorKey: "email", header: "Email", meta: { width: "200px" } },
  {
    id: "status",
    accessorKey: "status",
    header: "Status",
    meta: { width: "120px" },
    cell: ({ row }: { row: { original: Student } }) => row.original.status,
  },
  {
    id: "created_at",
    accessorKey: "created_at",
    header: "Created at",
    meta: { width: "160px" },
    cell: ({ row }: { row: { original: Student } }) => {
      return new Date(row.original.created_at).toLocaleDateString();
    },
  },
];

export function getStudentId(row: Student) {
  return row.id;
}
