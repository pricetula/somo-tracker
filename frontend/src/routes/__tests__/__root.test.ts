/**
 * Frontend-only test for rootLoader — verifies /me is called when page renders.
 * No backend involved — mocks getAuthState and get-me modules.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { rootLoader } from '../__root';

vi.mock('@/features/auth/api/get-auth-state', () => ({
  getAuthState: vi.fn(),
}));
vi.mock('@/features/auth/api/get-me', () => ({
  getMe: vi.fn(),
}));

import { getAuthState } from '@/features/auth/api/get-auth-state';
import { getMe } from '@/features/auth/api/get-me';

const mockGetAuthState = vi.mocked(getAuthState);
const mockGetMe = vi.mocked(getMe);

describe('rootLoader', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('calls getAuthState and getMe when hasSession is true', async () => {
    mockGetAuthState.mockResolvedValue({
      hasSession: true,
    });
    mockGetMe.mockResolvedValue({
      user_id: 'user-123',
      full_name: 'Test User',
      email: 'test@example.com',
      role: 'HEADTEACHER',
      active_school: { id: 'school-1', name: 'School A' },
      schools: [{ id: 'school-1', name: 'School A' }],
    });

    const result = await rootLoader();

    expect(mockGetAuthState).toHaveBeenCalledTimes(1);
    expect(mockGetMe).toHaveBeenCalledTimes(1);
    expect(result.hasSession).toBe(true);
    expect(result.me).not.toBeNull();
    expect(result.me?.full_name).toBe('Test User');
    expect(result.me?.role).toBe('HEADTEACHER');
  });

  it('does NOT call getMe when hasSession is false (no session)', async () => {
    mockGetAuthState.mockResolvedValue({
      hasSession: false,
    });

    const result = await rootLoader();

    expect(mockGetAuthState).toHaveBeenCalledTimes(1);
    expect(mockGetMe).not.toHaveBeenCalled();
    expect(result.hasSession).toBe(false);
    expect(result.me).toBeNull();
  });

  it('handles getMe failure gracefully — me is null, hasSession stays true', async () => {
    mockGetAuthState.mockResolvedValue({
      hasSession: true,
    });
    mockGetMe.mockRejectedValue(new Error('API error'));

    const result = await rootLoader();

    expect(mockGetAuthState).toHaveBeenCalledTimes(1);
    expect(mockGetMe).toHaveBeenCalledTimes(1);
    expect(result.hasSession).toBe(true);
    expect(result.me).toBeNull();
  });
});
