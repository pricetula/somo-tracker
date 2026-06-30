/**
 * Students layout — renders the main content alongside the @modal parallel slot.
 *
 * The @modal slot can intercept import routes from within /students,
 * rendering them as a dialog overlay while keeping the listing page mounted underneath.
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
