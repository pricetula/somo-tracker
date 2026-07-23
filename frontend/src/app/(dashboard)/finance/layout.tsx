/**
 * Finance layout — renders the main content alongside the @modal parallel slot.
 *
 * The @modal slot intercepts /finance/import and /finance/[id] when navigated
 * from within /finance, rendering as a dialog overlay while keeping the
 * listing page mounted underneath.
 *
 * Keep this layout thin — no data fetching, no providers, just slot composition.
 */

export default function FinanceLayout({
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
