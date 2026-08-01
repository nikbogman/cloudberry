# Graph Report - homelab  (2026-07-26)

## Corpus Check
- 96 files · ~47,381 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 538 nodes · 723 edges · 64 communities (33 shown, 31 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 60 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a901f618`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Server API Reachability
- Shared Git-Pull Deploy Helper
- Pi API Wake-on-LAN
- Graphify Skill Exports
- Deploy Domain Concepts
- Caddy Deploy ADR Decisions
- Homelab Domain Glossary
- Wake Plugin Go Internals
- Control Plane Shared Middleware
- Frontend TS Config
- Control Plane ADR Decisions
- Frontend Test Tooling
- Tailnet Identity Auth
- Domain Docs Conventions
- Control UI App Mounting
- Graphify Query Traversal
- Wayfinder Issue Tracker
- Pyinfra Change Detection
- WSGI Bootstrap Tests
- Graphify Incremental Update
- WSGI Bootstrap Tests
- Wake Plugin Handler Tests
- Graphify Watch Mode
- Typed Settings Pattern
- Issue Tracker File Convention
- Control Plane Services
- Graphify URL Ingest
- Triage Label Convention
- Deploy Entrypoint
- Token Reduction Benchmark
- WoL Broadcast Domain ADR
- GitHub Issues Deferred
- Deploy Concept
- handle_errors-inside-handle_path bug found & fixed
- Pyinfra Inventory Name
- Control API Deploy File
- Host Role Detection Helpers
- Pyinfra Host Groups
- Control UI Favicon
- AssertTailnetOnlyBind
- github.com/nikbogman/homelab/control-plane
- deploy.sh
- Homelab Control System
- Declarative Provisioning with pyinfra
- 0013-caddy-binary-built-off-device-by-pyinfra.md
- 01 — Pre-existing bug found and fixed: `handle_errors` nested inside `handle_path`
- 01 — Control-plane auth & logging scaffold
- 02 — Reachability status (health check + UI display)
- 03 — Wake action (Control Pi API + UI Wake button)
- 04 — Suspend action (Control server API + UI Suspend button)
- 05 — Auto-wake proxy for workload services
- 01 — Inventory scaffold
- 02 — Tailscale deploy file
- 03 — Docker deploy file
- 04 — Shared git-pull + systemd deploy helper
- 05 — Control Pi API deploy file (+ Control UI static delivery)
- 06 — Control server API deploy file
- 07 — Caddy deploy file
- 08 — Deploy entrypoint
- 09 — Disposable-container test harness & idempotency verification
- ADR-0009: Manual-Only Deploy Trigger
- ADR-0006: Git-Pull Deploy Model, Control UI Built Off-Device
- ADR-0007: systemd Units for Control APIs, Not Docker

## God Nodes (most connected - your core abstractions)
1. `Homelab README.md` - 20 edges
2. `mustNewHandler()` - 19 edges
3. `compilerOptions` - 16 edges
4. `mustNewHandlerWithConfig()` - 15 edges
5. `mustNewHandler()` - 14 edges
6. `NewHandler()` - 13 edges
7. `authedRequest()` - 12 edges
8. `NewHandler()` - 9 edges
9. `Handler` - 8 edges
10. `NewGrafanaCloudLogger()` - 8 edges

## Surprising Connections (you probably didn't know these)
- `Homelab CLAUDE.md (Agent Skills Config)` --references--> `CONTEXT.md (Domain Glossary)`  [EXTRACTED]
  CLAUDE.md → CONTEXT.md
- `Homelab README.md` --references--> `CONTEXT.md (Domain Glossary)`  [EXTRACTED]
  README.md → CONTEXT.md
- `Homelab README.md` --references--> `ADR-0001: Direct Browser-to-API Calls, No Pi-Side Relay`  [EXTRACTED]
  README.md → docs/adr/0001-direct-browser-to-api-no-relay.md
- `Homelab README.md` --references--> `ADR-0002: Suspend-to-RAM Only, No Full Shutdown`  [EXTRACTED]
  README.md → docs/adr/0002-suspend-only-no-shutdown.md
- `ADR-0002: Suspend-to-RAM Only, No Full Shutdown` --rationale_for--> `Suspend (concept)`  [INFERRED]
  docs/adr/0002-suspend-only-no-shutdown.md → CONTEXT.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **graphify /graphify Pipeline Steps** — claude_skills_graphify_skill_step0_github_clone, claude_skills_graphify_skill_step1_ensure_installed, claude_skills_graphify_skill_step2_detect_files, claude_skills_graphify_skill_step3_extraction, claude_skills_graphify_skill_step4_build_graph, claude_skills_graphify_skill_step5_label_communities, claude_skills_graphify_skill_step6_obsidian_html, claude_skills_graphify_skill_step9_manifest_cleanup [EXTRACTED 1.00]

## Communities (64 total, 31 thin omitted)

### Community 0 - "Server API Reachability"
Cohesion: 0.08
Nodes (32): Homelab CLAUDE.md (Agent Skills Config), Control UI (concept), CONTEXT.md (Domain Glossary), Identity Header (concept), Pi API (concept), Pi Proxy (concept), Reachable (concept), Server API (concept) (+24 more)

### Community 1 - "Shared Git-Pull Deploy Helper"
Cohesion: 0.07
Nodes (31): BaseSettings, DpkgArchitecture, has_device_role(), linux_codename(), linux_distro_id(), FactBase, Helpers shared by more than one Deploy file.  Not a Deploy file itself -- define, Native apt architecture (e.g. "amd64", "arm64") per `dpkg     --print-architectu (+23 more)

### Community 2 - "Pi API Wake-on-LAN"
Cohesion: 0.14
Nodes (26): Request, ResponseWriter, NewHandler(), EventLogger, Handler, T, mustNewHandler(), mustNewHandlerWithConfig() (+18 more)

### Community 3 - "Graphify Skill Exports"
Cohesion: 0.06
Nodes (37): Graphify Slash-Command Trigger, FalkorDB Export, MCP stdio Server, Neo4j Export, Wiki Export, Confidence Score Rubric, Hyperedges Rule, Node ID Format Rule (+29 more)

### Community 4 - "Deploy Domain Concepts"
Cohesion: 0.67
Nodes (3): Deploy (concept), Deploy File (concept), Host Group (concept)

### Community 5 - "Caddy Deploy ADR Decisions"
Cohesion: 0.40
Nodes (5): @deploy decorator, Deploy file, files.template operation, git.repo operation, systemd.service operation

### Community 6 - "Homelab Domain Glossary"
Cohesion: 0.20
Nodes (12): EventLogger, Handler, reachabilityTracker, Suspender, Request, ResponseWriter, NewHandler(), Copy() (+4 more)

### Community 7 - "Wake Plugin Go Internals"
Cohesion: 0.22
Nodes (9): Handler, Request, ResponseWriter, newComputeProxy(), retryUntilReachable(), autoWakeThrottler, Mutex, RoundTripper (+1 more)

### Community 9 - "Control Plane Shared Middleware"
Cohesion: 0.37
Nodes (14): closedPortTarget(), T, splitHostPort(), TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking(), TestProxy_AutoWakeDoesNotThrottleOnceTheWindowHasElapsed(), TestProxy_AutoWakeThrottlesRepeatCallsWithinTheWindow(), TestProxy_DoesNotFallBackToIndexHTMLForAnUnknownPath(), TestProxy_FailsOpenWhenTheWakerErrors() (+6 more)

### Community 10 - "Frontend TS Config"
Cohesion: 0.09
Nodes (21): compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, moduleResolution (+13 more)

### Community 11 - "Control Plane ADR Decisions"
Cohesion: 0.19
Nodes (23): fakeLogger, fakeSuspender, loggedEvent, authedRequest(), EventLogger, Handler, Request, T (+15 more)

### Community 12 - "Frontend Test Tooling"
Cohesion: 0.10
Nodes (19): devDependencies, jsdom, typescript, vite, vitest, name, private, scripts (+11 more)

### Community 13 - "Tailnet Identity Auth"
Cohesion: 0.14
Nodes (17): Client, main(), main(), EnvOr(), MustEnv(), MustEnvInt(), NewGrafanaCloudLogger(), T (+9 more)

### Community 14 - "Domain Docs Conventions"
Cohesion: 0.25
Nodes (8): CONTEXT.md, docs/adr/ (ADR directory), ADR conflict flagging practice, /domain-modeling skill, Glossary vocabulary discipline, /grill-with-docs skill, /improve-codebase-architecture skill, Single-context repo convention

### Community 15 - "Control UI App Mounting"
Cohesion: 0.18
Nodes (4): MountOptions, mountUi(), ReachabilityState, STATUS_LABELS

### Community 16 - "Graphify Query Traversal"
Cohesion: 0.29
Nodes (7): BFS/DFS Traversal Modes, graphify reflect / LESSONS.md, save-result Feedback Loop, Constrained Query Expansion, /graphify explain, /graphify path, /graphify query

### Community 17 - "Wayfinder Issue Tracker"
Cohesion: 0.50
Nodes (5): Blocked by: NN mechanism, Child ticket (wayfinder), Frontier scan (open/unblocked/unclaimed tickets), map.md (wayfinder map file), /wayfinder

### Community 18 - "Pyinfra Change Detection"
Cohesion: 0.50
Nodes (5): .will_change vs .did_change() distinction, OperationMeta.did_change(), @docker connector, Host group, OperationMeta.will_change

### Community 19 - "WSGI Bootstrap Tests"
Cohesion: 0.32
Nodes (9): BuildMagicPacket(), NewWakeOnLanSender(), T, TestBuildMagicPacketAcceptsHyphenSeparatedMac(), TestBuildMagicPacketAcceptsLowercaseMac(), TestBuildMagicPacketHasSixLeadingFFBytes(), TestBuildMagicPacketRejectsInvalidMac(), TestBuildMagicPacketRepeatsMacSixteenTimes() (+1 more)

### Community 20 - "Graphify Incremental Update"
Cohesion: 0.50
Nodes (3): Code-Only Fast Path (skip LLM), detect_incremental(), --update flag

### Community 21 - "WSGI Bootstrap Tests"
Cohesion: 0.32
Nodes (11): SystemSuspender, NewSystemSuspender(), NewSystemSuspenderWithCommand(), T, TestConstructorRejectsAnEmptyCommand(), TestDefaultCommandIsSystemctlSuspendOnly(), TestSuspendPropagatesErrorFromExitCode(), TestSuspendPropagatesErrorWhenCommandIsNotFound() (+3 more)

### Community 23 - "Graphify Watch Mode"
Cohesion: 0.67
Nodes (3): Debounce (3s default), graphify.watch background watcher, --watch flag

### Community 24 - "Typed Settings Pattern"
Cohesion: 0.33
Nodes (5): Behavior, Configuration, Deployment, Development, UI

### Community 25 - "Issue Tracker File Convention"
Cohesion: 0.67
Nodes (3): issues/<NN>-<slug>.md ticket files, .scratch/<feature-slug>/ convention, spec.md (per-feature spec/PRD)

### Community 26 - "Control Plane Services"
Cohesion: 0.16
Nodes (21): GetCallerIdentity(), Handler, Request, remoteHost(), RequireTailnetIdentity(), RequireTailnetIdentityOrLoopback(), Handler, Request (+13 more)

### Community 29 - "Deploy Entrypoint"
Cohesion: 0.67
Nodes (3): Caddyfile.j2 validated with real caddy binary (ticket 07), git_systemd_service live demo (ticket 04), "Highest available seam" testing philosophy

### Community 36 - "Control API Deploy File"
Cohesion: 0.22
Nodes (8): Custom Caddy wake plugin (replaces caddy-wol), Further Notes, Implementation Decisions, Out of Scope, Problem Statement, Solution, Testing Decisions, User Stories

### Community 41 - "AssertTailnetOnlyBind"
Cohesion: 0.31
Nodes (8): AssertTailnetOnlyBind(), appFactory(), T, TestAllowsLoopbackAndTailnetAddresses(), TestAppFactoryRefusesToStartOnOffTailnetHost(), TestAppFactoryStartsOnLoopbackHost(), TestRejectsOffTailnetAddresses(), BindOffTailnetError

### Community 44 - "Homelab Control System"
Cohesion: 0.22
Nodes (8): Further Notes, Homelab Control System, Implementation Decisions, Out of Scope, Problem Statement, Solution, Testing Decisions, User Stories

### Community 45 - "Declarative Provisioning with pyinfra"
Cohesion: 0.22
Nodes (8): Declarative Provisioning with pyinfra, Further Notes, Implementation Decisions, Out of Scope, Problem Statement, Solution, Testing Decisions, User Stories

### Community 46 - "0013-caddy-binary-built-off-device-by-pyinfra.md"
Cohesion: 0.06
Nodes (25): Compute API, Control-plane services (Go), Development, Gateway API, `internal/eventlog`, `internal/tailnet`, The Caddy binary is cross-compiled off-device and shipped by pyinfra, not hand-built on the Pi, Control-plane events ship straight to Grafana Cloud's Loki endpoint — no self-hosted agent (+17 more)

### Community 47 - "01 — Pre-existing bug found and fixed: `handle_errors` nested inside `handle_path`"
Cohesion: 0.40
Nodes (4): 01 — Pre-existing bug found and fixed: `handle_errors` nested inside `handle_path`, Fix, Verification, What was found

## Knowledge Gaps
- **152 isolated node(s):** `github.com/nikbogman/homelab/control-plane`, `lokiLine`, `contextKey`, `name`, `private` (+147 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **31 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewHandler()` connect `Pi API Wake-on-LAN` to `AssertTailnetOnlyBind`, `Control Plane Services`, `WSGI Bootstrap Tests`, `Wake Plugin Go Internals`?**
  _High betweenness centrality (0.079) - this node is a cross-community bridge._
- **Why does `BuildMagicPacket()` connect `WSGI Bootstrap Tests` to `Pi API Wake-on-LAN`?**
  _High betweenness centrality (0.047) - this node is a cross-community bridge._
- **Why does `AssertTailnetOnlyBind()` connect `AssertTailnetOnlyBind` to `Pi API Wake-on-LAN`, `Homelab Domain Glossary`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `mustNewHandlerWithConfig()` (e.g. with `NewHandler()` and `TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking()`) actually correct?**
  _`mustNewHandlerWithConfig()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/nikbogman/homelab/control-plane`, `lokiLine`, `contextKey` to the rest of the system?**
  _152 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Server API Reachability` be split into smaller, more focused modules?**
  _Cohesion score 0.0846774193548387 - nodes in this community are weakly interconnected._
- **Should `Shared Git-Pull Deploy Helper` be split into smaller, more focused modules?**
  _Cohesion score 0.07474747474747474 - nodes in this community are weakly interconnected._