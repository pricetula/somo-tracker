import type { ColumnDef } from "@tanstack/react-table";
import type { Student } from "./api/students.api";

export const studentsColumns: ColumnDef<any, Student, unknown>[] = [
  {
    accessorKey: "first_name",
    header: "First name",
    meta: { width: "150px" },
  },
  {
    accessorKey: "last_name",
    header: "Last name",
    meta: { width: "150px" },
  },
  { accessorKey: "email", header: "Email", meta: { width: "200px" } },
  {
    accessorKey: "status",
    header: "Status",
    meta: { width: "120px" },
    cell: ({ row }: { row: { original: Student } }) => row.original.status,
  },
  {
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
