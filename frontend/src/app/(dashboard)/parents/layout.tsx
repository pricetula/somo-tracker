/**
 * Parents layout — renders the main content alongside the @modal parallel slot.
 *
 * The @modal slot intercepts /parents/import when navigated from within /parents,
 * rendering the import form as a dialog overlay while keeping the listing
 * page mounted underneath.
 *
 * Keep this layout thin — no data fetching, no providers, just slot composition.
 */

export default function ParentsLayout({
    children,
    modal,
}: {
    children: React.ReactNode;
    modal: React.ReactNode;
}) {
    return (
        <>
            {children}
            {modal}
        </>
    );
}
