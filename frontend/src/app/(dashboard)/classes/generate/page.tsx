"use client";

import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ClassStreamGenerator } from "@/features/classes";

export default function ClassesGeneratePage() {
  const router = useRouter();

  return (
    <div className="min-h-screen p-6">
      {/* Back navigation */}
      <div className="mb-6">
        <Button
          variant="ghost"
          onClick={() => router.push("/dashboard")}
          className="gap-2"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Dashboard
        </Button>
      </div>

      {/* Centered form */}
      <div className="mx-auto max-w-3xl">
        <ClassStreamGenerator onSuccess={() => router.push("/dashboard")} />
      </div>
    </div>
  );
}
