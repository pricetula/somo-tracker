import { TimetableDetail } from "@/features/timetable/components/timetable-detail";

type Props = {
    params: Promise<{ id: string }>;
};

export default async function TimetableTemplatePage({ params }: Props) {
    const { id } = await params;
    return <TimetableDetail templateId={id} />;
}
