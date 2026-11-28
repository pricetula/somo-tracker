import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

import { listStudents, deleteStudents, type Student } from "./students.api";

const mockFetch = vi.fn();
global.fetch = mockFetch as any;

const makePage = (overrides: Partial<{
  data: Student[];
  next_cursor: string | null;
  has_more: boolean;
}> = {}) => {
  const data: Student[] = overrides.data ?? [
    { id: "std_001", first_name: "Ashley", last_name: "Example", status: "active", email: "a@example.school", created_at: "2026-01-01T00:00:00Z" },
    { id: "std_002", first_name: "Benny", last_name: "Testwood", status: "inactive", email: "b@example.school", created_at: "2026-01-02T00:00:00Z" },
  ];
  return {
    data,
    meta: { next_cursor: overrides.next_cursor ?? null, has_more: overrides.has_more ?? false },
  };
};

beforeEach(() => {
  mockFetch.mockClear();
  mockFetch.mockResolvedValue(new Response(JSON.stringify(makePage()), {
    status: 200,
    headers: { "content-type": "application/json" },
  }));
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("listStudents", () => {
  it("fetches the first page with default params", async () => {
    await listStudents({ cursor: null, limit: 50, sorting: [], filters: {} });

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url] = mockFetch.mock.calls[0];
    expect(String(url)).toBe("/api/mock-students?limit=50");
  });

  it("passes search, sorting and filters as query params", async () => {
    await listStudents({
      cursor: "abc",
      limit: 20,
      search: "ashley",
      sorting: [{ id: "created_at", desc: true }],
      filters: { status: ["active", "inactive"] },
    });

    const [url] = mockFetch.mock.calls[0];
    const u = new URL(String(url), "http://localhost");
    expect(u.searchParams.get("limit")).toBe("20");
    expect(u.searchParams.get("cursor")).toBe("abc");
    expect(u.searchParams.get("search")).toBe("ashley");
    expect(u.searchParams.get("sort")).toBe("created_at");
    expect(u.searchParams.get("order")).toBe("desc");
    expect(u.searchParams.getAll("status")).toEqual(["active", "inactive"]);
  });

  it("maps the response to DataTablePage shape", async () => {
    mockFetch.mockResolvedValueOnce(new Response(JSON.stringify(makePage({ next_cursor: "cursor-123", has_more: true })), {
      status: 200,
      headers: { "content-type": "application/json" },
    }));

    const result = await listStudents({ cursor: null, limit: 50, sorting: [], filters: {} });

    expect(result.hasMore).toBe(true);
    expect(result.nextCursor).toBe("cursor-123");
    expect(result.data).toHaveLength(2);
    expect(result.data[0].id).toBe("std_001");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce(new Response("error", { status: 500 }));

    await expect(listStudents({ cursor: null, limit: 50, sorting: [], filters: {} })).rejects.toThrow(
      "Failed to fetch students",
    );
  });
});

describe("deleteStudents", () => {
  it("posts ids and returns mapped bulk delete result", async () => {
    mockFetch.mockResolvedValueOnce(new Response(JSON.stringify({
      deleted_ids: ["std_001"],
      failed: [{ id: "std_003", code: "DEPENDENCY_EXISTS", message: "Student has an active dependency." }],
    }), {
      status: 200,
      headers: { "content-type": "application/json" },
    }));

    const result = await deleteStudents(["std_001", "std_003"]);

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url, init] = mockFetch.mock.calls[0];
    expect(String(url)).toBe("/api/mock-students/bulk-delete");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toEqual({ ids: ["std_001", "std_003"] });
    expect(result.deletedIds).toEqual(["std_001"]);
    expect(result.failed).toHaveLength(1);
    expect(result.failed[0].code).toBe("DEPENDENCY_EXISTS");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce(new Response("error", { status: 500 }));
    await expect(deleteStudents(["std_001"])).rejects.toThrow("Failed to delete students");
  });
});
