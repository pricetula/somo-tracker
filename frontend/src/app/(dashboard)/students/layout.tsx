/**
 * Students layout — renders the main content alongside the @modal parallel slot.
 *
 * Keep this layout thin — no data fetching, no providers, just slot composition.
 */

export default function StudentsLayout({
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
