import { Plus } from "lucide-react";
import Link from "next/link";

const summaries = [
    {
        label: "Students",
        value: "412",
        sub: "52% boys · 48% girls",
        link: "/students",
        addLink: "/students/add",
        addLinkLabel: "Add students",
    },
    {
        label: "Teachers",
        value: "24",
        sub: "2 invites pending",
        link: "/teachers",
        addLink: "/teachers/invite",
        addLinkLabel: "Invite teachers",
    },
    {
        label: "Guardians",
        value: "389",
        sub: "17 invites pending",
        link: "/guardians",
        addLink: "/guardians/invite",
        addLinkLabel: "Invite guardians",
    },
    {
        label: "Finance",
        value: "3",
        sub: "",
        link: "/finance",
        addLink: "/finance/invite",
        addLinkLabel: "Invite finance",
    },
    {
        label: "Admins",
        value: "2",
        sub: "",
        link: "/admins",
        addLink: "/admins/invite",
        addLinkLabel: "Invite admins",
    },
];

export function SummaryCards() {
    return (
        <div className="mb-20 grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5">
            {summaries.map((s) => (
                <div key={s.label} className="flex flex-col border p-3">
                    <Link className="mb-2 space-x-1" href={s.link}>
                        <span className="text-2xl">{s.value}</span>
                        <span className="text-muted-foreground">{s.label}</span>
                    </Link>
                    <Link
                        className="text-muted-foreground flex items-center gap-0.5"
                        href={s.addLink}
                    >
                        <Plus size={12} />
                        <span>{s.addLinkLabel}</span>
                    </Link>
                    {s.sub && <p className="text-muted-foreground mt-1">{s.sub}</p>}
                </div>
            ))}
        </div>
    );
}
