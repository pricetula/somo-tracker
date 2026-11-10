/**
 * Client entry point with QueryClientProvider
 */

import { StrictMode, startTransition } from 'react';
import { hydrateRoot } from 'react-dom/client';
import { StartClient } from '@tanstack/react-start/client';
import { QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'sonner';
import { queryClient } from '@/lib/query-client';
import './styles/globals.css';

startTransition(() => {
  hydrateRoot(
    document,
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <StartClient />
        <Toaster position="top-right" />
      </QueryClientProvider>
    </StrictMode>,
  );
});