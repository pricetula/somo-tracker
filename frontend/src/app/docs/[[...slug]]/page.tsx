import React from 'react';
import { notFound } from 'next/navigation';
import Link from 'next/link';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { ArrowLeft, FileText } from 'lucide-react';
import { getDocData, getAllDocSlugs, getAllDocMetadata } from '@/lib/docs';

interface Props {
  params: Promise<{ slug?: string[] }>;
}

export async function generateStaticParams() {
  const slugs = getAllDocSlugs();
  return slugs.map((slug) => ({ slug: [slug] }));
}

export default async function DocPage({ params }: Props) {
  const { slug } = await params;

  // ── Root /docs — show docs listing ──────────────────────────────────
  if (!slug || slug.length === 0) {
    const docs = getAllDocMetadata();

    return (
      <div>
        <h1 className="text-3xl font-bold tracking-tight mb-2">Documentation</h1>
        <p className="text-base text-muted-foreground mb-8 leading-relaxed">
          Browse the Somotracker documentation to learn about features, setup, and configuration.
        </p>

        <div className="grid gap-4">
          {docs.map((doc) => (
            <Link
              key={doc.slug}
              href={`/docs/${doc.slug}`}
              className="group block rounded-lg border border-border p-5 hover:border-primary/50 hover:bg-muted/30 transition-all"
            >
              <div className="flex items-start gap-3">
                <FileText className="h-5 w-5 mt-0.5 text-muted-foreground shrink-0" />
                <div>
                  <h2 className="font-semibold text-foreground group-hover:text-primary transition-colors">
                    {doc.title}
                  </h2>
                  <p className="text-sm text-muted-foreground mt-1">
                    {doc.description}
                  </p>
                </div>
              </div>
            </Link>
          ))}
        </div>
      </div>
    );
  }

  // ── Specific doc page ───────────────────────────────────────────────
  const docSlug = slug.join('/');
  const doc = getDocData(docSlug);

  if (!doc) {
    notFound();
  }

  const { metadata, content } = doc;

  return (
    <article className="prose prose-gray dark:prose-invert max-w-none">
      {/* Back link */}
      <Link
        href="/docs"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-6 no-underline"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to docs
      </Link>

      {/* Title */}
      <h1 className="text-3xl font-bold tracking-tight mb-2">{metadata.title}</h1>

      {/* Description */}
      {metadata.description && (
        <p className="text-base text-muted-foreground mb-8 leading-relaxed">
          {metadata.description}
        </p>
      )}

      {/* MDX Content rendered as Markdown */}
      <div className="markdown-content">
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            h2: ({ children, ...props }) => {
              const text = extractTextContent(children);
              const anchorMatch = text.match(/\{#([^}]+)\}$/);
              const id = anchorMatch ? anchorMatch[1] : undefined;
              const cleanText = anchorMatch ? text.replace(/\s*\{#[^}]+\}$/, '') : text;
              return (
                <h2 id={id} className="group scroll-mt-20">
                  <span className="text-xl font-semibold mt-10 mb-4 block">{cleanText}</span>
                </h2>
              );
            },
            a: ({ href, children, ...props }) => (
              <a
                href={href}
                className="text-primary hover:underline font-medium"
                {...props}
              >
                {children}
              </a>
            ),
            code: ({ children, ...props }) => (
              <code
                className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono text-foreground"
                {...props}
              >
                {children}
              </code>
            ),
            pre: ({ children, ...props }) => (
              <pre
                className="rounded-lg bg-muted p-4 overflow-x-auto text-sm"
                {...props}
              >
                {children}
              </pre>
            ),
          }}
        >
          {content}
        </ReactMarkdown>
      </div>
    </article>
  );
}

/** Recursively extract plain text from React children. */
function extractTextContent(children: React.ReactNode): string {
  if (typeof children === 'string') return children;
  if (typeof children === 'number') return String(children);
  if (Array.isArray(children)) return children.map(extractTextContent).join('');
  if (React.isValidElement(children)) {
    const props = children.props as { children?: React.ReactNode };
    return extractTextContent(props.children);
  }
  return '';
}
