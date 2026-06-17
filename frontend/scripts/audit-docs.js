/* eslint-disable @typescript-eslint/no-require-imports */
const fs = require("fs");
const path = require("path");
const matter = require("gray-matter");

const DOCS_DIR = path.join(process.cwd(), "content/docs");
const APP_DIR = path.join(process.cwd(), "src/app");

function getFiles(dir, ext) {
    let results = [];
    if (!fs.existsSync(dir)) return results;
    const list = fs.readdirSync(dir);
    list.forEach((file) => {
        const filePath = path.join(dir, file);
        const stat = fs.statSync(filePath);
        if (stat && stat.isDirectory()) {
            results = results.concat(getFiles(filePath, ext));
        } else if (filePath.endsWith(ext)) {
            results.push(filePath);
        }
    });
    return results;
}

function auditDocs() {
    console.log("🔍 Running Documentation & Tooltip Sync Audit...");
    const uiFiles = [...getFiles(APP_DIR, ".tsx"), ...getFiles(APP_DIR, ".ts")];
    let errorsFound = false;

    const helpComponentRegex =
        /<FeatureHelp\s+[^>]*slug=["']([^"']+)["'](?:[^>]*anchorId=["']([^"']+)["'])?[^>]*\/>/g;

    uiFiles.forEach((filePath) => {
        const content = fs.readFileSync(filePath, "utf8");
        let match;

        while ((match = helpComponentRegex.exec(content)) !== null) {
            const [, slug, anchorId] = match;
            const mdxPath = path.join(DOCS_DIR, `${slug}.mdx`);

            if (!fs.existsSync(mdxPath)) {
                console.error(
                    `❌ Error in ${path.relative(process.cwd(), filePath)}: Slug "${slug}" has no corresponding file at content/docs/${slug}.mdx`
                );
                errorsFound = true;
                continue;
            }

            const mdxContent = fs.readFileSync(mdxPath, "utf8");
            const { data, content: body } = matter(mdxContent);

            if (!data.tooltipSummary) {
                console.error(
                    `❌ Error in ${slug}.mdx: Missing required 'tooltipSummary' in YAML frontmatter.`
                );
                errorsFound = true;
            }

            if (anchorId) {
                const anchorRegex = new RegExp(`{#${anchorId}}|id=["']${anchorId}["']`, "i");
                if (!anchorRegex.test(body)) {
                    console.error(
                        `❌ Error in ${path.relative(process.cwd(), filePath)}: anchorId="${anchorId}" does not exist inside '${slug}.mdx'.`
                    );
                    errorsFound = true;
                }
            }
        }
    });

    if (errorsFound) {
        console.log("\n🛑 Audit failed. Fix the synchronization gaps listed above.");
        process.exit(1);
    } else {
        console.log(
            "✅ Audit passed! All UI tooltips match backend MDX content definitions safely."
        );
        process.exit(0);
    }
}

auditDocs();
