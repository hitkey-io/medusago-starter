# MedusaGo backend

The backend and admin of your store, built with [MedusaGo](https://github.com/hitkey-io/medusago),
the Go implementation of Medusa.

## Commands

Run them from this directory:

| Command                            | Description                                     |
| ---------------------------------- | ----------------------------------------------- |
| `go run . develop`                 | Start the development server (restarts on code changes) |
| `go run . db:setup`                | Create the database, run migrations and sync links |
| `go run . db:migrate`              | Run pending migrations                          |
| `go run . user -e <email> -p <pw>` | Create an admin user                            |
| `go run . build`                   | Build the production server into `.medusa/server` |
| `go run . start`                   | Start the production server                     |

The server runs at `http://localhost:9000` and the admin at `http://localhost:9000/app`.

## Configuration

`medusa-config.go` is the equivalent of Medusa's `medusa-config.ts`; values
come from `.env` (see `.env.template`).

## Customizations

`src/` mirrors a Medusa project (`api`, `jobs`, `links`, `modules`, `scripts`,
`subscribers`, `workflows`, `admin`). Go doesn't load files at run time, so
`main.go` registers what the project adds to the CLI.

The project comes with:

- `src/migration-scripts`: the data migration scripts that `db:migrate` runs
  once. `initial-data-seed` creates the store, a region, shipping options and
  four demo products; `region-store-policies` adds the store policy values
  that the storefront reads from each region's metadata.
- `src/search/product.go`: the `product` search index that the storefront
  searches and filters, with a price per currency.
- `src/api/middlewares.go`: opens the `product` index to `/store/search`.
