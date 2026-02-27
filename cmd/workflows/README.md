# Workflow Scripts

This folder stores reusable developer workflow scripts.

## Build + Restart Dev

- Windows CMD: `cmd\workflows\build-restart.cmd`
- Bash (macOS/Linux): `cmd/workflows/build-restart.sh`

Both scripts execute the same workflow:

1. `pnpm build` in `web`
2. stop processes listening on ports `9880` and `9881`
3. start `wails dev`

The scripts auto-detect the repository root by walking up from the script location and finding:

- `go.mod`
- `web/package.json`

So they continue to work even if the script folder is moved deeper under the repo.