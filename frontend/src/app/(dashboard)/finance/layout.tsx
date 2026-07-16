/**
 * Finance layout — renders sub-navigation tabs and content.
 *
 * The @modal parallel slot intercepts /finance/invitations when navigated
 * from within /finance, rendering the invite form as a dialog overlay.
 */

"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const financeTabs = [
    { label: "Staff", href: "/finance" },
    { label: "Fee Categories", href: "/finance/fee-categories" },
    { label: "Fee Templates", href: "/finance/fee-templates" },
    { label: "Invoices", href: "/finance/invoices" },
];

export default function FinanceLayout({
    children,
    modal,
}: {
    children: React.ReactNode;
    modal: React.ReactNode;
}) {
    const pathname = usePathname();

    function isActive(href: string) {
        if (href === "/finance") return pathname === "/finance";
        return pathname.startsWith(href);
    }

    return (
        <div className="space-y-6">
            <nav className="flex gap-4 border-b">
                {financeTabs.map((tab) => (
                    <Link
                        key={tab.href}
                        href={tab.href}
                        className={`border-b-2 px-1 pb-2 font-medium transition-colors ${
                            isActive(tab.href)
                                ? "text-foreground border-primary"
                                : "text-muted-foreground hover:text-foreground hover:border-foreground border-transparent"
                        }`}
                    >
                        {tab.label}
                    </Link>
                ))}
            </nav>
            {children}
            {modal}
        </div>
    );
}
