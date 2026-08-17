/**
 * Default fallback for the `@modal` parallel slot.
 *
 * Rendered when the current URL does not match any route inside a role's
 * `@modal` folder (e.g. visiting `/admins` instead of `/admins/add`).
 * Returns null so nothing overlays the page.
 */
export default function ModalDefault() {
    return null;
}
