export const studentFilterGroups = [
    {
        id: "class",
        label: "Class",
        items: [
            {
                id: "class",
                label: "Class",
                type: "sub_menu_single" as const,
                submenu: [] as { id: string; label: string; value: string }[],
            },
        ],
    },
];

export function mapStudentFiltersToParams(filters?: Record<string, string | string[]>) {
    const classId = typeof filters?.class === "string" ? filters.class : undefined;
    return { class_id: classId };
}
