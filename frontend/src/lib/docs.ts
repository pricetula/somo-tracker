import fs from "fs";
import path from "path";
import matter from "gray-matter";

const DOCS_DIRECTORY = path.join(process.cwd(), "content/docs");

interface DocMetadata {
    title: string;
    description: string;
    tooltipSummary?: string;
}

export function getDocData(slug: string) {
    const fullPath = path.join(DOCS_DIRECTORY, `${slug}.mdx`);

    if (!fs.existsSync(fullPath)) {
        return null;
    }

    const fileContents = fs.readFileSync(fullPath, "utf8");
    const { data, content } = matter(fileContents);

    return {
        slug,
        metadata: data as DocMetadata,
        content,
    };
}

export function getTooltipContent(slug: string): string {
    const doc = getDocData(slug);
    return doc?.metadata?.tooltipSummary || "Learn more in our documentation.";
}

export function getAllDocSlugs(): string[] {
    if (!fs.existsSync(DOCS_DIRECTORY)) return [];
    return fs
        .readdirSync(DOCS_DIRECTORY)
        .filter((file) => file.endsWith(".mdx"))
        .map((file) => file.replace(/\.mdx$/, ""));
}

export function getAllDocMetadata() {
    const slugs = getAllDocSlugs();
    return slugs
        .map((slug) => {
            const doc = getDocData(slug);
            if (!doc) return null;
            return {
                slug,
                title: doc.metadata.title,
                description: doc.metadata.description,
            };
        })
        .filter(Boolean) as { slug: string; title: string; description: string }[];
}
