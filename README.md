# MedusaGo Starter

A [MedusaGo](https://github.com/hitkey-io/medusago) store: the Go implementation
of [Medusa](https://medusajs.com), API-compatible with Medusa v2.

- `apps/backend`: the MedusaGo server and admin dashboard (Go module).
- `apps/storefront`: the Medusa Next.js Starter Storefront.

Create a project from this starter with the installer:

```sh
go run github.com/hitkey-io/medusago/packages/cli/create-medusago-app@latest my-medusa-store
```

Then start the backend from `apps/backend` with `go run . develop` and open
`http://localhost:9000/app`.

The storefront is the original Medusa Next.js starter (MIT); run it from
`apps/storefront` with `npm install && npm run dev` (port 8000).
