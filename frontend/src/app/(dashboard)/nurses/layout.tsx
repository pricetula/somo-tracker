/**
 * Nurses layout — renders the main content alongside the @modal parallel slot.
 *
 * The @modal slot intercepts /nurses/add when navigated from within /nurses,
 * rendering the import form as a dialog overlay while keeping the listing
 * page mounted underneath.
 */

export default function NursesLayout({
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
