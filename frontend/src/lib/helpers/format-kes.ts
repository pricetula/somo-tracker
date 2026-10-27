/**
 * Formats an integer amount in minor currency units into a readable currency string.
 * Supports different currencies (e.g. KES, USD) via an optional `currency` param.
 */
export function formatKes(amountMinor: number, currency: string = 'KES'): string {
  if (!Number.isInteger(amountMinor)) {
    throw new TypeError('formatKes expects an integer amount')
  }
  const amount = amountMinor / 100
  return new Intl.NumberFormat('en-KE', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}
