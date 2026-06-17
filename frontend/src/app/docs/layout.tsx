import Link from "next/link";
import { getAllDocMetadata } from "@/lib/docs";
import { BookOpen } from "lucide-react";

export default function DocsLayout({ children }: { children: React.ReactNode }) {
    const docs = getAllDocMetadata();

    return (
        <div className="flex min-h-screen">
            {/* Sidebar */}
            <aside className="border-border bg-muted/20 hidden w-64 shrink-0 border-r md:block">
                <div className="sticky top-0 flex h-screen flex-col">
                    <div className="border-border border-b p-4">
                        <Link
                            href="/"
                            className="text-foreground flex items-center gap-2 text-sm font-semibold"
                        >
                            <BookOpen className="h-4 w-4" />
                            Somotracker Docs
                        </Link>
                    </div>
                    <nav className="flex-1 space-y-1 overflow-y-auto p-3">
                        {docs.map((doc) => (
                            <Link
                                key={doc.slug}
                                href={`/docs/${doc.slug}`}
                                className="text-muted-foreground hover:bg-muted hover:text-foreground block rounded-md px-3 py-2 text-sm transition-colors"
                            >
                                {doc.title}
                            </Link>
                        ))}
                    </nav>
                </div>
            </aside>

            {/* Main content */}
            <main className="min-w-0 flex-1">
                <div className="mx-auto max-w-3xl px-6 py-10">{children}</div>
            </main>
        </div>
    );
}
