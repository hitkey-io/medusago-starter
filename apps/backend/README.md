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
`subscribers`, `workflows`, `admin`).
