import { defineConfig } from 'astro/config'
import cloudflare from '@astrojs/cloudflare';

import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  // Cloudflare adapter typically requires server or hybrid output
  output: 'server',

  adapter: cloudflare(),
  site: 'https://somotracker.com/',
  base: '/',
  session: false,

  vite: {
    plugins: [tailwindcss()]
  }
})