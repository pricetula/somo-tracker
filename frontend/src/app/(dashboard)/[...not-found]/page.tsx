// eslint-disable-next-line @typescript-eslint/no-unused-vars
import * as React from "react";
import { notFound } from "next/navigation";

export default function CatchAll() {
    notFound(); // bubbles up to (dashboard)/not-found.tsx ✅
}
