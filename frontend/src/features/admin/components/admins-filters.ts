export const adminFilterGroups = [
    {
        id: "invitation",
        label: "Invitation Status",
        items: [
            {
                id: "status",
                label: "Status",
                type: "sub_menu_single" as const,
                submenu: [
                    { id: "all", label: "All", value: "all" },
                    { id: "invited", label: "Invited", value: "invited" },
                    { id: "accepted", label: "Accepted", value: "accepted" },
                ],
            },
        ],
    },
];

export function mapAdminFiltersToParams(filters?: Record<string, string | string[]>) {
    const status = typeof filters?.status === "string" ? filters.status : undefined;
    return {
        invitation_status:
            status === "all" || !status ? undefined : (status as "invited" | "accepted"),
    };
}
