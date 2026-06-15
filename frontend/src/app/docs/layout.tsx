import Link from 'next/link';
import { getAllDocMetadata } from '@/lib/docs';
import { cn } from '@/lib/utils';
import { BookOpen } from 'lucide-react';

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const docs = getAllDocMetadata();

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside className="w-64 shrink-0 border-r border-border bg-muted/20 hidden md:block">
        <div className="sticky top-0 flex flex-col h-screen">
          <div className="p-4 border-b border-border">
            <Link href="/" className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <BookOpen className="h-4 w-4" />
              Somotracker Docs
            </Link>
          </div>
          <nav className="flex-1 overflow-y-auto p-3 space-y-1">
            {docs.map((doc) => (
              <Link
                key={doc.slug}
                href={`/docs/${doc.slug}`}
                className="block rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              >
                {doc.title}
              </Link>
            ))}
          </nav>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 min-w-0">
        <div className="mx-auto max-w-3xl px-6 py-10">
          {children}
        </div>
      </main>
    </div>
  );
}
