# Typed web client from OpenAPI

Contract: [openapi/openapi.yaml](openapi/openapi.yaml) (`openapi: 3.1.0`).

## TypeScript (openapi-typescript + openapi-fetch)

```bash
npx openapi-typescript openapi/openapi.yaml -o web/src/api/schema.d.ts
```

```bash
npm add openapi-fetch
```

Example:

```ts
import createClient from "openapi-fetch";
import type { paths } from "./api/schema";

const client = createClient<paths>({ baseUrl: "" });
const { data, error } = await client.POST("/auth/login", {
  body: { username: "admin", password: process.env.ADMIN_PASSWORD! },
});
```

SSE: `EventSource` cannot set `Authorization`. Use query param `accessToken=<jwt>` on `/api/logs/server/stream` and `/api/logs/pi/stream` as documented in the OpenAPI file.

## Go (optional)

Use [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) against `openapi/openapi.yaml` if you want generated Gin stubs or Go client types in-repo.
