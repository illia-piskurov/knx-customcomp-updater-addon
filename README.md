# KNX Custom Component Updater (Home Assistant Add-on)

This repository is a Home Assistant add-on repository and includes an add-on for managing custom KNX components in `knxN` format (`knx2`, `knx3`, ...).

## Features

- Lists installed `knxN` domains from `/config/custom_components`
- Creates a new domain (`knxN`) from the built-in KNX integration
- Updates a single domain or all domains at once
- Deletes a domain (regular hard delete)
- Provides a simple Ingress web UI

## Repository structure

- `repository.json` - Home Assistant add-on repository metadata
- `addon/` - add-on source code (Go backend, web UI, `config.yaml`, `Dockerfile`)
- `knx_update.py` - legacy helper script kept for reference

## Install in Home Assistant

1. In Home Assistant, open `Settings -> Add-ons -> Add-on Store`.
2. Open the top-right menu and select `Repositories`.
3. Add this repository URL:
   `https://github.com/illia-piskurov/knx-customcomp-updater-addon`
4. Open the `KNX Manager` add-on, install it, then start it.
5. Optionally enable `Show in sidebar`, then open the add-on web UI.

## Important notes

- Allowed domains must match `^knx[0-9]+$`.
- Home Assistant version is read via Supervisor API (`SUPERVISOR_TOKEN`).
- If HA version cannot be retrieved, the UI displays an unavailable status.
- Job state is stored in memory only (no persistent storage).

## Local development

```bash
cd addon
go mod tidy
go test ./...
go run ./cmd/server
```

After startup, open `http://localhost:8080`.