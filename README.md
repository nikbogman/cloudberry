<div align="center">

<img src="ui/icons/metal-awake.svg" width="88" alt="">

# metal

My self-hosting platform: a home server that sleeps until one of its apps is opened, deploys its apps on a git push, and keeps itself up to date.

![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![Tailscale](https://img.shields.io/badge/Tailscale-242424?logo=tailscale&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker_Compose-2496ED?logo=docker&logoColor=white)
![Raspberry Pi](https://img.shields.io/badge/Raspberry_Pi-C51A4A?logo=raspberrypi&logoColor=white)
![Ubuntu](https://img.shields.io/badge/Ubuntu-E95420?logo=ubuntu&logoColor=white)
![Cloudflare Pages](https://img.shields.io/badge/Cloudflare_Pages-F38020?logo=cloudflarepages&logoColor=white)
![GitHub Actions](https://img.shields.io/badge/GitHub_Actions-2088FF?logo=githubactions&logoColor=white)
![SOPS + age](https://img.shields.io/badge/SOPS_+_age-3A3F47)
![Grafana Loki](https://img.shields.io/badge/Grafana_Loki-F46800?logo=grafana&logoColor=white)

</div>

> [!NOTE]
> This is a personal project, built for my two machines and my tailnet. It isn't meant for anyone else to run: names, addresses and choices are shaped around my setup. It's written for me and the AI agents I work with; agent instructions are in [CLAUDE.md](CLAUDE.md).

## What it does

- **Wake on request.** Open an app and the server wakes up: gateway, on an always-on Pi, holds the request, sends Wake-on-LAN, and proxies once the app answers, in about 7 seconds. When nothing has used it for a while, it goes back to sleep, but only if gateway woke it.
- **Push to deploy.** Apps are plain Docker Compose stacks in the [deploys](https://github.com/nikbogman/deploys) repo. Pushing to `main` deploys them: the server pulls the repo and brings up what changed.
- **Secrets in git, encrypted.** Each stack's secrets are SOPS-encrypted for two age keys, the dev machine's and the server's, and decrypted only where they're used.
- **Self-updating, with rollback.** Tagging a release ships both services: each device installs the new binary on its own, and puts the old one back if the new one won't start.
- **One hostname per app, tailnet only.** Every app gets `https://<app>.<tailnet>.ts.net`, with TLS and the caller's identity from Tailscale. A firewall lets only the Pi reach an app's port on the LAN.
- **A control page.** metal, the UI, shows whether the server is awake, what woke it, when it sleeps again, today's wakes and suspends, and the apps, with Wake, Suspend and the auto-wake and auto-suspend switches. Its ingot glows when the server is on.

## How it works

![Architecture: a phone or laptop on the tailnet loads the metal UI from Cloudflare Pages and calls gateway on raspberry (wake, status) and hostd on blackberry (suspend). gateway sends Wake-on-LAN to blackberry, proxies to its stacks, and asks hostd for routes and suspend. hostd pulls the deploys repo and both services pull releases from GitHub. Both send events to Grafana Cloud.](docs/architecture.svg)

Two machines, named for their hardware (both are berries); services are named for what they do:

| Machine | Hardware | Role |
|---|---|---|
| **blackberry** | Ryzen 3 3250U, 8GB RAM, Ubuntu Server, Ethernet | The server. Runs the stacks. Suspended until needed. |
| **raspberry** | Raspberry Pi Zero W, Raspbian Lite, Wi-Fi | Always on, low power. On the same LAN as blackberry, so its Wake-on-LAN broadcast lands. |

### gateway <img src="https://cdn.simpleicons.org/raspberrypi/C51A4A" height="16" alt="">

A Go service on raspberry, and its whole tailnet presence: no tailscaled, just in-process tsnet nodes. One, `raspberry`, serves the UI's `/wake`, `/settings` and `/status`; each app gets its own.

1. **Wake.** A request to an app while blackberry sleeps sends one magic packet and holds the request until the app's port answers on the LAN.
2. **Proxy.** Then it reverse-proxies, with the caller's tailnet identity checked.
3. **Suspend.** After each app's `idle_minutes` without traffic, it asks hostd to suspend, if it did the waking.
4. **Status.** It probes blackberry every 15 seconds and keeps two days of wakes and suspends for the UI.

Routes come from each stack's `x-route` block, which hostd serves at `/routes`. [More](cmd/gateway/README.md).

### hostd <img src="https://cdn.simpleicons.org/go/00ADD8" height="16" alt="">

A Go service on blackberry, outside Docker so it stays in control while Docker redeploys. It handles suspend and the health check, and deploys the stacks: every few minutes while awake, it pulls deploys, decrypts each stack's secrets, runs `docker compose up` for the folders that changed (or `down` for removed ones), and keeps the firewall's port list current. [More](cmd/hostd/README.md).

### Stacks <img src="https://cdn.simpleicons.org/docker/2496ED" height="16" alt="">

Plain Compose projects in [deploys](https://github.com/nikbogman/deploys): today Immich, and a shared Postgres (with pgvector) and Valkey for every stack. Data lives under `/srv/stacks/`, and the photo library on the HDD, which spins down when idle. [More](https://github.com/nikbogman/deploys/blob/main/README.md).

### The UI <img src="https://cdn.simpleicons.org/cloudflarepages/F38020" height="16" alt="">

Static files on Cloudflare Pages, no build step. It works only from a device on the tailnet, since the browser calls gateway and hostd directly at their tailnet URLs: tailnet membership is the access control. [More](ui/README.md).

### Releases <img src="https://cdn.simpleicons.org/githubactions/2088FF" height="16" alt="">

CI vets, tests and cross-builds on every push. A `vX.Y.Z` tag publishes a release; both services poll for it, check its `SHA256SUMS`, swap themselves, and are rolled back by systemd if they fail to start three times. Events from both go to Grafana Cloud's Loki.

## Run it

Notes for me, or an AI agent, to set it up again. Everything goes through [Task](https://taskfile.dev) over ssh; details in [DEPLOY.md](DEPLOY.md).

```sh
cp .env.example .env    # hosts, MAC address, tailnet URLs, Grafana token
task                    # converge both machines: Docker, HDD spindown, network,
                        # stack sync setup, firewall, gateway, hostd
task ui                 # the UI, to Cloudflare Pages
```

Checks:

```sh
go vet ./... && go test ./...
node --test ui/app.test.mjs
```

After that, day to day:

- **Deploy an app:** push to deploys' `main`.
- **Change a secret:** `sops <stack>/secrets.env` in deploys, then push.
- **Ship gateway and hostd:** tag `vX.Y.Z` here and push the tag.

## Docs

- [CONTEXT.md](CONTEXT.md): domain glossary, the vocabulary everything else uses
- [DESIGN.md](DESIGN.md): design decisions, request paths, security, limitations
- [DEPLOY.md](DEPLOY.md): deploying both machines and the UI, releases, stack sync setup
- A README next to each component's code: [gateway](cmd/gateway/README.md), [hostd](cmd/hostd/README.md), [ui](ui/README.md), [stacks](https://github.com/nikbogman/deploys/blob/main/README.md)

## Notes

- Built for one person, me: one operator, one server, one tailnet.
- The UI is public static files but grants nothing: it reveals the tailnet's hostnames, which only tailnet members can reach.
- Wake-on-LAN needs both machines on one broadcast domain; a cold wake takes about 7 seconds, most of it firmware resume and the link training.
