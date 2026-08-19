/**
 * Default fallback for the `@modal` parallel slot.
 *
 * Rendered when the current URL does not match an intercepted add route
 * (e.g. visiting `/admins` instead of `/admins/add`, or hard-navigating to
 * `/admins/add`, where the `(.)` intercept does not apply).
 * Returns null so nothing overlays the page.
 */
export default function ModalDefault() {
    return null;
}
