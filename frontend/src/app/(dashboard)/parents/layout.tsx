/**
 * Parents layout — renders the main content.
 *
 * Keep this layout thin — no data fetching, no providers, just slot composition.
 */

export default function ParentsLayout({ children }: { children: React.ReactNode }) {
    return <>{children}</>;
}
