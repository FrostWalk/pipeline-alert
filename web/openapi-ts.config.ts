import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: '../openapi/openapi.yaml',
  output: {
    path: 'src/client',
  },
  plugins: [
    '@hey-api/typescript',
    '@hey-api/client-fetch',
    {
      name: '@hey-api/sdk',
      operations: { nesting: 'operationId' },
    },
  ],
})
