import { describe, expect, it } from 'vitest'
import { formatKes } from '../format-kes'

describe('formatKes', () => {
  it('formats a positive amount in KES', () => {
    const result = formatKes(15000)
    expect(result).toMatch(/^Ksh\s*150\.00$/)
  })

  it('formats a zero amount', () => {
    const result = formatKes(0)
    expect(result).toMatch(/^Ksh\s*0\.00$/)
  })

  it('formats a decimal amount', () => {
    const result = formatKes(7550)
    expect(result).toMatch(/^Ksh\s*75\.50$/)
  })

  it('accepts a custom currency code', () => {
    const result = formatKes(10000, 'USD')
    expect(result).toMatch(/^US\$/)
  })

  it('throws on non-integer', () => {
    expect(() => formatKes(1.5 as any)).toThrow(TypeError)
  })

  it('formats with two decimal places', () => {
    const result = formatKes(12345)
    expect(result).toMatch(/\.\d{2}$/)
  })
})