"use client";

interface SummaryCardShellProps {
    title: string;
    href: string;
    children?: React.ReactNode;
    actions?: React.ReactNode;
}

export function SummaryCardShell({ title, href, children, actions }: SummaryCardShellProps) {
    return (
        <article className="flex min-h-[7.5rem] flex-col gap-3">
            <header className="space-y-2">
                <h2 className="text-xl font-bold">
                    <a href={href}>{title}</a>
                </h2>
                {children}
            </header>
            <div className="mt-auto space-y-2">{actions}</div>
        </article>
    );
}
