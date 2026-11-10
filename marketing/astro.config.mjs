import { defineConfig } from 'astro/config'
import cloudflare from '@astrojs/cloudflare';

export default defineConfig({
  output: 'server', // Cloudflare adapter typically requires server or hybrid output
  adapter: cloudflare(),
  site: 'https://somotracker.app/',
  base: '/',
  session: false
})
