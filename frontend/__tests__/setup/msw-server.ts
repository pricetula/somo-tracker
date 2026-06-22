/**
 * MSW server for HTTP and SSE mocking in the Bulk Staff Import tests.
 *
 * Handlers are registered per-test via server.use(...).
 * The server is started in vitest.setup.ts before all tests and reset after each.
 */

import { setupServer } from "msw/node";

export const server = setupServer();
