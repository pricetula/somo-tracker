/**
 * Route policy - Single source of truth for auth navigation decisions
 * Used by both middleware (Layer A) and root beforeLoad (Layer B)
 * Pure function, no side effects, fully unit testable
 */

export type RoutePolicyDecision =
  | { type: 'allow' }
  | { type: 'redirect'; to: string }
  | { type: 'redirect-with-next'; to: string; next: string };

// Public routes that never require authentication
const PUBLIC_ROUTES = new Set<string>([
  '/login',
  '/signup',
  '/auth/callback',
  '/onboarding/tenant-name',
  '/unauthorized',
]);

// Routes that are allowed without cookie but redirect if cookie present
const GUEST_ONLY_ROUTES = new Set<string>([
  '/login',
  '/signup',
  '/onboarding/tenant-name',
]);

// Framework internal routes that should never be redirected
const FRAMEWORK_ROUTES_PREFIXES = [
  '/_tanstack', // TanStack Start internals
  '/@vite', // Vite HMR
  '/@fs', // Vite file system
  '/node_modules',
];

// Asset extensions that should never be redirected
const ASSET_EXTENSIONS = [
  '.js',
  '.css',
  '.map',
  '.ico',
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.svg',
  '.woff',
  '.woff2',
  '.ttf',
  '.eot',
  '.webp',
  '.avif',
];

/**
 * Check if a path is a framework/internal route that should never be redirected
 */
function isFrameworkRoute(pathname: string): boolean {
  return FRAMEWORK_ROUTES_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

/**
 * Check if a path is a static asset that should never be redirected
 */
function isAssetRoute(pathname: string): boolean {
  return ASSET_EXTENSIONS.some((ext) => pathname.endsWith(ext));
}

/**
 * Check if a path is an API/server function route that should never be redirected
 */
function isApiRoute(pathname: string): boolean {
  return pathname.startsWith('/api/') || pathname.startsWith('/server/');
}

/**
 * Normalize pathname by removing trailing slash (except root)
 */
function normalizePath(pathname: string): string {
  if (pathname === '/') return '/';
  return pathname.replace(/\/+$/, '');
}

/**
 * Core route policy function
 * @param pathname - Current pathname (e.g., '/dashboard', '/login')
 * @param hasCookie - Whether the __Secure-session_token cookie is present
 * @returns Decision: allow, redirect, or redirect-with-next
 */
export function routePolicy(
  pathname: string,
  hasCookie: boolean
): RoutePolicyDecision {
  const path = normalizePath(pathname);

  // Never redirect framework internals, assets, or API routes
  if (isFrameworkRoute(path) || isAssetRoute(path) || isApiRoute(path)) {
    return { type: 'allow' };
  }

  // Public routes - always allow
  if (PUBLIC_ROUTES.has(path)) {
    // Guest-only routes: redirect to home if cookie present
    if (GUEST_ONLY_ROUTES.has(path) && hasCookie) {
      return { type: 'redirect', to: '/' };
    }
    return { type: 'allow' };
  }

  // Protected routes
  if (!hasCookie) {
    // No cookie - redirect to login with next parameter
    const nextPath = encodeURIComponent(path);
    return { type: 'redirect-with-next', to: '/login', next: nextPath };
  }

  // Has cookie - allow (backend will validate on API calls)
  return { type: 'allow' };
}

/**
 * Check if a route is public (no auth required)
 */
export function isPublicRoute(pathname: string): boolean {
  const path = normalizePath(pathname);
  return PUBLIC_ROUTES.has(path);
}

/**
 * Check if a route is guest-only (redirects authenticated users)
 */
export function isGuestOnlyRoute(pathname: string): boolean {
  const path = normalizePath(pathname);
  return GUEST_ONLY_ROUTES.has(path);
}