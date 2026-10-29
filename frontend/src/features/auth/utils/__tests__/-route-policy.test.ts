/**
 * Route policy unit tests
 * Tests all route categories with cookie present/absent
 */

import { describe, it, expect } from 'vitest';
import { routePolicy, isPublicRoute, isGuestOnlyRoute } from '../route-policy';

describe('routePolicy', () => {
  describe('Truly public routes - always allow (even with cookie)', () => {
    const publicRoutes = [
      '/',
      '/auth/callback',
      '/unauthorized',
    ];

    publicRoutes.forEach((route) => {
      it(`allows ${route} without cookie`, () => {
        expect(routePolicy(route, false)).toEqual({ type: 'allow' });
      });

      it(`allows ${route} with cookie`, () => {
        expect(routePolicy(route, true)).toEqual({ type: 'allow' });
      });
    });
  });

  describe('Guest-only routes - redirect if cookie present', () => {
    const guestOnlyRoutes = ['/login', '/signup', '/onboarding/tenant-name'];

    guestOnlyRoutes.forEach((route) => {
      it(`allows ${route} without cookie`, () => {
        expect(routePolicy(route, false)).toEqual({ type: 'allow' });
      });

      it(`redirects ${route} to / with cookie`, () => {
        expect(routePolicy(route, true)).toEqual({ type: 'redirect', to: '/' });
      });
    });
  });

  describe('Protected routes - require cookie', () => {
    const protectedRoutes = [
      '/dashboard',
      '/students',
      '/teachers',
      '/settings',
      '/onboarding/school-details',
      '/onboarding/admin-user',
    ];

    protectedRoutes.forEach((route) => {
      it(`redirects ${route} to /login without cookie`, () => {
        const result = routePolicy(route, false);
        expect(result.type).toBe('redirect-with-next');
        if (result.type === 'redirect-with-next') {
          expect(result.to).toBe('/login');
          // next is URL-encoded
          expect(result.next).toBe(encodeURIComponent(route));
        }
      });

      it(`allows ${route} with cookie`, () => {
        expect(routePolicy(route, true)).toEqual({ type: 'allow' });
      });
    });
  });

  describe('Framework/internal routes - never redirect', () => {
    const frameworkRoutes = [
      '/_tanstack/something',
      '/@vite/client',
      '/@fs/some-file',
      '/node_modules/something',
    ];

    frameworkRoutes.forEach((route) => {
      it(`allows ${route} without cookie`, () => {
        expect(routePolicy(route, false)).toEqual({ type: 'allow' });
      });

      it(`allows ${route} with cookie`, () => {
        expect(routePolicy(route, true)).toEqual({ type: 'allow' });
      });
    });
  });

  describe('Asset routes - never redirect', () => {
    const assetRoutes = [
      '/app.js',
      '/styles.css',
      '/favicon.ico',
      '/logo.png',
      '/font.woff2',
    ];

    assetRoutes.forEach((route) => {
      it(`allows ${route} without cookie`, () => {
        expect(routePolicy(route, false)).toEqual({ type: 'allow' });
      });

      it(`allows ${route} with cookie`, () => {
        expect(routePolicy(route, true)).toEqual({ type: 'allow' });
      });
    });
  });

  describe('API routes - never redirect', () => {
    const apiRoutes = ['/api/v1/students', '/api/v1/auth/session', '/server/some-function'];

    apiRoutes.forEach((route) => {
      it(`allows ${route} without cookie`, () => {
        expect(routePolicy(route, false)).toEqual({ type: 'allow' });
      });

      it(`allows ${route} with cookie`, () => {
        expect(routePolicy(route, true)).toEqual({ type: 'allow' });
      });
    });
  });

  describe('Trailing slash normalization', () => {
    it('normalizes trailing slashes', () => {
      expect(routePolicy('/dashboard/', false)).toEqual(routePolicy('/dashboard', false));
      expect(routePolicy('/login/', true)).toEqual(routePolicy('/login', true));
    });

    it('preserves root slash', () => {
      expect(routePolicy('/', false)).toEqual({ type: 'allow' });
    });
  });
});

describe('isPublicRoute', () => {
  it('returns true for public routes', () => {
    expect(isPublicRoute('/')).toBe(true);
    expect(isPublicRoute('/login')).toBe(true);
    expect(isPublicRoute('/auth/callback')).toBe(true);
  });

  it('returns false for protected routes', () => {
    expect(isPublicRoute('/dashboard')).toBe(false);
    expect(isPublicRoute('/students')).toBe(false);
  });
});

describe('isGuestOnlyRoute', () => {
  it('returns true for guest-only routes', () => {
    expect(isGuestOnlyRoute('/login')).toBe(true);
    expect(isGuestOnlyRoute('/signup')).toBe(true);
    expect(isGuestOnlyRoute('/onboarding/tenant-name')).toBe(true);
  });

  it('returns false for other routes', () => {
    expect(isGuestOnlyRoute('/')).toBe(false);
    expect(isGuestOnlyRoute('/dashboard')).toBe(false);
    expect(isGuestOnlyRoute('/auth/callback')).toBe(false);
  });
});