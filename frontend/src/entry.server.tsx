/**
 * Server entry point with authentication middleware
 * Layer A: Middleware for document requests (server-side redirect before HTML render)
 */

import {
  createStartHandler,
  defaultStreamHandler,
} from '@tanstack/react-start/server';
import { routePolicy } from '@/features/auth/utils/route-policy';

// Create a handler that wraps the default stream handler with auth middleware
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const authHandler = async (ctx: any) => {
  const { request } = ctx;

  // Only apply to document requests (HTML navigation)
  const accept = request.headers.get('accept') || '';
  const secFetchDest = request.headers.get('sec-fetch-dest') || '';

  const isDocumentRequest =
    accept.includes('text/html') && secFetchDest === 'document';

  if (isDocumentRequest) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    // Read session cookie from request headers
    const cookieHeader = request.headers.get('cookie') || '';
    const hasSession = cookieHeader.includes('__Secure-session_token');

    // Apply route policy
    const decision = routePolicy(pathname, hasSession);

    switch (decision.type) {
      case 'allow':
        break;
      case 'redirect':
        return Response.redirect(new URL(decision.to, request.url), 302);
      case 'redirect-with-next': {
        const redirectUrl = new URL(decision.to, request.url);
        redirectUrl.searchParams.set('next', decision.next);
        return Response.redirect(redirectUrl, 302);
      }
    }
  }

  // Call the default stream handler
  return defaultStreamHandler(ctx);
};

const handler = createStartHandler(authHandler);

export default handler;