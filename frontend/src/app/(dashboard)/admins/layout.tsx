/**
 * Admins layout — renders the main content alongside the @modal parallel slot.
 *
 * The @modal slot intercepts /admins/invitations when navigated from within /admins,
 * rendering the import form as a dialog overlay while keeping the listing
 * page mounted underneath.
 *
 * Keep this layout thin — no data fetching, no providers, just slot composition.
 */

export default function AdminsLayout({
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
