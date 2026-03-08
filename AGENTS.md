# Project rules and instructions

- Do not add comments unless absolutely necessary.
- The project uses go for the server and client, svelte for the client dashboard, and react + shadcn for the admin UI.
- The admin server resides in internal/server/admin.
- Use github.com/charmbracelet/log for structured logging.
- When making any changes, make sure to add/update tests for the same code.

## Directory structure

- cmd/ contains the portr cli and portrd entrypoints
- internal/client/dashboard/ui - code for the client dashboard, written in svelte
- internal/server/admin/web-v2 - code for the admin dashboard, written in react + shadcn
- migrations/ contains the database migrations for the server
