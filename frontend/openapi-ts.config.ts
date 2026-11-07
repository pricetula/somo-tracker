import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: '../docs/api/openapi.yaml',
  output: {
    path: './src/features/auth/api/generated',
    format: 'prettier',
    lint: 'eslint',
  },
  plugins: [
    '@hey-api/typescript',
    '@hey-api/zod',
    '@hey-api/client-fetch',
  ],
  typescript: {
    enums: 'typescript',
    dates: true,
  },
  client: '@hey-api/client-fetch',
});