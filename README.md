# Cloudberry

The setup that lets me run my home server only when I need it, and reach its
apps from anywhere.

This repo isn't meant for outside contributors. It's written for me and the
AI agents I work with; agent instructions are in [CLAUDE.md](CLAUDE.md).

## How it works

```mermaid
flowchart LR
  subgraph cf[Cloudflare Pages]
    ui["UI<br/>static files"]
  end
  browser([Browser on the tailnet]) -->|loads| ui

  subgraph lan[Home LAN]
    subgraph rb[raspberry · always on]
      gateway
    end
    subgraph bb[blackberry · suspended until needed]
      hostd
      caddy[Caddy] --> stacks[stacks]
    end
  end

  browser -->|wake| gateway
  browser -->|suspend, health| hostd
  gateway -.->|Wake-on-LAN| bb
```

Machines are named for their hardware (both are berries; `cloudberry` is the
whole thing), services for what they do:

| Machine | Hardware | Role |
|---|---|---|
| **blackberry** | Ryzen 3 3250U, 8GB RAM, Ubuntu Server, Ethernet | The home server. Suspended until needed. |
| **raspberry** | Raspberry Pi Zero, Raspbian Lite, Wi-Fi | Always on, low power. On the same LAN as blackberry, so its Wake-on-LAN broadcast lands. |

The repo is a monorepo holding everything in the diagram:

| Part | Code | Runs on | Shipped as |
|---|---|---|---|
| **UI**: Wake/Suspend page | [ui/](ui/) | Cloudflare Pages | Static files, uploaded by `task ui` |
| **gateway**: sends Wake-on-LAN; wakes blackberry on request and proxies stacks | [cmd/gateway](cmd/gateway/) | raspberry | Go binary cross-compiled for armv6, run by systemd |
| **hostd**: suspend, health check | [cmd/hostd](cmd/hostd/) | blackberry | Go binary, run by systemd |
| **stacks**: the apps, behind Caddy | [stacks/](stacks/) | blackberry | Docker Compose, started with `docker compose` (not `task`) |

The two services are one Go module: each has its entrypoint in `cmd/<name>/`
and its logic in `internal/<name>/`, and the rest of `internal/` is shared
code. The stacks don't use it.

The UI is public static files; it works only from a device on the Tailscale
network, because the browser calls gateway and hostd at their tailnet URLs.
Access control is tailnet membership.

## Docs

- [CONTEXT.md](CONTEXT.md): domain glossary, the vocabulary everything else uses
- [DESIGN.md](DESIGN.md): design decisions, request paths, security, limitations
- [DEPLOY.md](DEPLOY.md): deploying both machines and the UI with `task`
- A README next to each component's code: [gateway](cmd/gateway/README.md), [hostd](cmd/hostd/README.md), [ui](ui/README.md), [stacks](stacks/README.md)
