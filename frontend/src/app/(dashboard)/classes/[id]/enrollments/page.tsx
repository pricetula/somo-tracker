import { EnrollmentsPageContent } from "@/features/enrollments";

export default async function EnrollmentsPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;
    return (
        <div className="p-6">
            <EnrollmentsPageContent classId={id} />
        </div>
    );
}
