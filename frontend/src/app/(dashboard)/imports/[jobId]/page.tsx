/**
 * Import Job detail page — view progress, results, and failures.
 */

import { ImportJobDetail } from "@/features/import-jobs";

interface ImportJobDetailPageProps {
    params: Promise<{ jobId: string }>;
}

export default async function ImportJobDetailPage({ params }: ImportJobDetailPageProps) {
    const { jobId } = await params;
    return <ImportJobDetail jobId={jobId} />;
}
