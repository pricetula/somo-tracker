import type { DataTablePage } from "@/components/shared/data-table/types";

type Student = {
  id: string;
  first_name: string;
  last_name: string;
  status: string;
  email: string;
  created_at: string;
};

export type { Student };

interface ListStudentsParams {
  cursor: string | null;
  limit: number;
  search?: string;
  sorting: { id: string; desc: boolean }[];
  filters: Record<string, unknown>;
}

export async function listStudents({
  cursor,
  limit,
  search,
  sorting,
  filters,
}: ListStudentsParams): Promise<DataTablePage<Student>> {
  const params = new URLSearchParams();
  params.set("limit", String(limit));
  if (cursor) params.set("cursor", cursor);
  if (search) params.set("search", search);
  if (sorting.length) {
    params.set("sort", sorting[0].id);
    params.set("order", sorting[0].desc ? "desc" : "asc");
  }
  if (filters.status) {
    const value = filters.status;
    const values = Array.isArray(value) ? value : [value];
    for (const v of values) {
      if (v) params.append("status", String(v));
    }
  }

  const res = await fetch(`/api/mock-students?${params.toString()}`, {
    method: "GET",
    credentials: "include",
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error("Failed to fetch students");
  }

  const raw = await res.json();
  return {
    data: (raw.data ?? []).map((s: Student) => ({
      id: s.id,
      first_name: s.first_name,
      last_name: s.last_name,
      status: s.status,
      email: s.email,
      created_at: s.created_at,
    })),
    nextCursor: raw.meta?.next_cursor ?? null,
    hasMore: !!raw.meta?.has_more,
  };
}

export async function deleteStudents(ids: string[]) {
  const res = await fetch("/api/mock-students/bulk-delete", {
    method: "POST",
    credentials: "include",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ ids }),
  });

  if (!res.ok) {
    throw new Error("Failed to delete students");
  }

  const raw = await res.json();
  return {
    deletedIds: raw.deleted_ids ?? [],
    failed: (raw.failed ?? []).map((f: { id: string; code?: string; message: string }) => ({
      id: f.id,
      code: f.code,
      message: f.message,
    })),
  };
}
