import { startRouter } from '@tanstack/react-start';
import { getRouter } from './router';

startRouter({
  router: getRouter(),
});
