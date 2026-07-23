/**
 * Academic Year detail page — view year info and manage terms.
 */

import { AcademicYearDetail } from "@/features/academic-years";

interface AcademicYearDetailPageProps {
    params: Promise<{ id: string }>;
}

export default async function AcademicYearDetailPage({ params }: AcademicYearDetailPageProps) {
    const { id } = await params;
    return <AcademicYearDetail id={id} />;
}
