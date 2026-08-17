import FinancePage from "../page";

/**
 * Children slot for `/finance/add`.
 *
 * Renders the exact same listing page as `/finance` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function FinanceAddPage() {
    return <FinancePage />;
}
