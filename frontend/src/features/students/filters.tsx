import type { DataTableFilter } from "@/components/shared/data-table/types";

export const studentFilters: DataTableFilter[] = [
  {
    id: "status",
    label: "Status",
    type: "multi-select",
    placeholder: "All statuses",
    options: [
      { label: "Active", value: "active" },
      { label: "Inactive", value: "inactive" },
      { label: "Graduated", value: "graduated" },
    ],
  },
];
