import { renderToString } from 'react-dom/server';
import { ServerRouterProvider } from '@tanstack/react-router';
import { router } from './routes/__root';

export function render(url: string) {
  return renderToString(
    <ServerRouterProvider router={router} context={{ url }} />
  );
}
