import { NextResponse } from "next/server";

const STATUSES = ["active", "inactive", "graduated"] as const;

function makeStudent(index: number) {
  const status = STATUSES[index % STATUSES.length];
  return {
    id: `std_${String(index + 1).padStart(3, "0")}`,
    first_name: ["Ashley", "Benny", "Carlos", "Diana", "Elias", "Fiona", "Gabi", "Hank"][index % 8],
    last_name: ["Example", "Testwood", "Sampleton", "Demovsky", "Illustration", "Prototypal", "Mockingham", "Dummy"][index % 8],
    status,
    email: `student${index + 1}@example.school`,
    created_at: new Date(Date.now() - (200 - index) * 86400000).toISOString(),
  };
}

const STUDENTS = Array.from({ length: 200 }, (_, i) => makeStudent(i));

function encodeCursor({ id, created_at }: { id: string; created_at: string }) {
  return Buffer.from(JSON.stringify({ id, created_at })).toString("base64");
}

function decodeCursor(cursor: string) {
  try {
    return JSON.parse(Buffer.from(cursor, "base64").toString("utf8")) as {
      id: string;
      created_at: string;
    };
  } catch {
    return null;
  }
}

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const limit = Math.min(Math.max(Number(searchParams.get("limit")) || 50, 1), 100);
  const cursor = searchParams.get("cursor") || null;
  const search = (searchParams.get("search") || "").toLowerCase();
  const sort = searchParams.get("sort") || "created_at";
  const order = searchParams.get("order") || "desc";
  const status = searchParams.get("status") || null;

  let rows = [...STUDENTS];

  // Resource filters validated by the backend (allowlist).
  if (status && STATUSES.includes(status as typeof STATUSES[number])) {
    rows = rows.filter((r) => r.status === status);
  }

  // Server-side search.
  if (search) {
    rows = rows.filter(
      (r) =>
        r.first_name.toLowerCase().includes(search) ||
        r.last_name.toLowerCase().includes(search) ||
        r.email.toLowerCase().includes(search),
    );
  }

  // Deterministic ordering with a unique tie-breaker (id), per ADR-0027 §3.3.
  rows.sort((a, b) => {
    const aVal = a[sort as keyof typeof a];
    const bVal = b[sort as keyof typeof b];
    let cmp = 0;
    if (aVal < bVal) cmp = -1;
    else if (aVal > bVal) cmp = 1;
    if (cmp === 0) {
      cmp = a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
    }
    return order === "desc" ? -cmp : cmp;
  });

  // Opaque cursor pagination.
  let startIndex = 0;
  if (cursor) {
    const decoded = decodeCursor(cursor);
    if (decoded) {
      startIndex = rows.findIndex((r) => r.id === decoded.id);
      if (startIndex === -1) startIndex = 0;
      else startIndex += 1;
    }
  }

  const page = rows.slice(startIndex, startIndex + limit);
  const hasMore = startIndex + limit < rows.length;
  const nextCursor = hasMore ? encodeCursor(page[page.length - 1]) : null;

  return NextResponse.json({
    data: page,
    meta: { next_cursor: nextCursor, has_more: hasMore },
  });
}
