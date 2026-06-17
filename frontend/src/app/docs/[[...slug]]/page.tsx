import React from "react";
import { notFound } from "next/navigation";
import Link from "next/link";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { ArrowLeft, FileText } from "lucide-react";
import { getDocData, getAllDocSlugs, getAllDocMetadata } from "@/lib/docs";

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
                <h1 className="mb-2 text-3xl font-bold tracking-tight">Documentation</h1>
                <p className="text-muted-foreground mb-8 text-base leading-relaxed">
                    Browse the Somotracker documentation to learn about features, setup, and
                    configuration.
                </p>

                <div className="grid gap-4">
                    {docs.map((doc) => (
                        <Link
                            key={doc.slug}
                            href={`/docs/${doc.slug}`}
                            className="group border-border hover:border-primary/50 hover:bg-muted/30 block rounded-lg border p-5 transition-all"
                        >
                            <div className="flex items-start gap-3">
                                <FileText className="text-muted-foreground mt-0.5 h-5 w-5 shrink-0" />
                                <div>
                                    <h2 className="text-foreground group-hover:text-primary font-semibold transition-colors">
                                        {doc.title}
                                    </h2>
                                    <p className="text-muted-foreground mt-1 text-sm">
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
    const docSlug = slug.join("/");
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
                className="text-muted-foreground hover:text-foreground mb-6 inline-flex items-center gap-1.5 text-sm no-underline transition-colors"
            >
                <ArrowLeft className="h-4 w-4" />
                Back to docs
            </Link>

            {/* Title */}
            <h1 className="mb-2 text-3xl font-bold tracking-tight">{metadata.title}</h1>

            {/* Description */}
            {metadata.description && (
                <p className="text-muted-foreground mb-8 text-base leading-relaxed">
                    {metadata.description}
                </p>
            )}

            {/* MDX Content rendered as Markdown */}
            <div className="markdown-content">
                <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={{
                        h2: ({ children }) => {
                            const text = extractTextContent(children);
                            const anchorMatch = text.match(/\{#([^}]+)\}$/);
                            const id = anchorMatch ? anchorMatch[1] : undefined;
                            const cleanText = anchorMatch
                                ? text.replace(/\s*\{#[^}]+\}$/, "")
                                : text;
                            return (
                                <h2 id={id} className="group scroll-mt-20">
                                    <span className="mt-10 mb-4 block text-xl font-semibold">
                                        {cleanText}
                                    </span>
                                </h2>
                            );
                        },
                        a: ({ href, children, ...props }) => (
                            <a
                                href={href}
                                className="text-primary font-medium hover:underline"
                                {...props}
                            >
                                {children}
                            </a>
                        ),
                        code: ({ children, ...props }) => (
                            <code
                                className="bg-muted text-foreground rounded px-1.5 py-0.5 font-mono text-sm"
                                {...props}
                            >
                                {children}
                            </code>
                        ),
                        pre: ({ children, ...props }) => (
                            <pre
                                className="bg-muted overflow-x-auto rounded-lg p-4 text-sm"
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
    if (typeof children === "string") return children;
    if (typeof children === "number") return String(children);
    if (Array.isArray(children)) return children.map(extractTextContent).join("");
    if (React.isValidElement(children)) {
        const props = children.props as { children?: React.ReactNode };
        return extractTextContent(props.children);
    }
    return "";
}
