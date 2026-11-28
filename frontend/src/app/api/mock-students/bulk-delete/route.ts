import { NextResponse } from "next/server";

export async function POST(request: Request) {
  const body = (await request.json()) as { ids?: string[] };
  const ids: string[] = Array.isArray(body.ids) ? body.ids : [];

  // Simulated business constraint: graduated students have an active dependency
  // and cannot be deleted.
  const deleted_ids: string[] = [];
  const failed: { id: string; code: string; message: string }[] = [];

  for (const id of ids) {
    const n = Number(id.replace("std_", ""));
    const status = ["active", "inactive", "graduated"][n % 3];
    if (status === "graduated") {
      failed.push({
        id,
        code: "DEPENDENCY_EXISTS",
        message: "Student has an active dependency.",
      });
    } else {
      deleted_ids.push(id);
    }
  }

  return NextResponse.json({ deleted_ids, failed });
}
