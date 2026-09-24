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
    {
        id: "status",
        label: "Status",
        items: [
            {
                id: "without_class",
                label: "Unassigned class",
                type: "button" as const,
                value: "true",
            },
            {
                id: "without_guardian",
                label: "No guardian",
                type: "button" as const,
                value: "true",
            },
        ],
    },
];

export function mapStudentFiltersToParams(filters?: Record<string, string | string[]>) {
    const classId = typeof filters?.class === "string" ? filters.class : undefined;
    const withoutClass = filters?.without_class === "true";
    const withoutGuardian = filters?.without_guardian === "true";
    return { class_id: classId, without_class: withoutClass, without_guardian: withoutGuardian };
}
