import { api } from "@/lib/api/client";

export async function createAcademicPeriod(data: {
    year: number;
    terms: { name: string; startDate: string; endDate: string }[];
}) {
    const payload = {
        year: data.year,
        terms: data.terms.map((t) => ({
            name: t.name,
            start_date: t.startDate,
            end_date: t.endDate,
        })),
    };
    return api.post("/api/school/academic-period", payload);
}
