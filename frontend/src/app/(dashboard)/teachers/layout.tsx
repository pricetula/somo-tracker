/**
 * Teachers layout — renders the main content alongside the @modal parallel slot.
 *
 * The @modal slot intercepts /teachers/invitations when navigated from within /teachers,
 * rendering the import form as a dialog overlay while keeping the listing
 * page mounted underneath.
 *
 * Keep this layout thin — no data fetching, no providers, just slot composition.
 */

export default function TeachersLayout({
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
