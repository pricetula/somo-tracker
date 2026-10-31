/**
 * Auth feature public API
 * Single entry point for all auth feature exports
 */

// Types
export * from './types';

// Schemas
export * from './schemas';

// Query keys
export * from './query-keys';

// Utils
export * from './utils/route-policy';

// API
export * from './api/request-magic-link';
export * from './api/authenticate-token';
export * from './api/get-onboarding';
export * from './api/update-onboarding';
export * from './api/complete-onboarding';
export * from './api/logout';
export * from './api/get-auth-state';

// Hooks
export * from './hooks/use-request-magic-link';
export * from './hooks/use-onboarding';
export * from './hooks/use-logout';
export * from './hooks/use-auth-state';

// Components
export * from './components/login-form';
export * from './components/callback-page';
export * from './components/onboarding-tenant-name';
export * from './components/logout-button';