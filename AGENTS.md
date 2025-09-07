# Project rules and instructions

- Do not add unnecessary comments.
- The project uses go for the server and client and svelte for the UI.
- The admin server now resides in tunnel/internal/server/admin directory. The admin/ at the root is deprecated and will be removed in the v1 release.
- Use github.com/charmbracelet/log for structured logging.
- When making any changes, make sure to add/update tests for the same code.

## Directory structure

- tunnel/ contains all the code for the v1 implementation (admin server + tunnel server + portr cli)
- tunnel/internal/client/dashboard/ui - code for the client dashboard, written in svelte
- tunnel/internal/admin/web - code for the old admin dashboard, written in svelte
- tunnel/internal/admin/web-v2 - code for the new admin dashboard, written in react + shadcn
