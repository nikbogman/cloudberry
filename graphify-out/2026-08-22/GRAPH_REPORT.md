# Graph Report - homelab  (2026-08-22)

## Corpus Check
- 71 files · ~39,897 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 457 nodes · 789 edges · 39 communities (24 shown, 15 thin omitted)
- Extraction: 89% EXTRACTED · 11% INFERRED · 0% AMBIGUOUS · INFERRED: 90 edges (avg confidence: 0.82)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f76c9b0d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Provisioning Facts & Deploy
- Project Docs & Domain Glossary
- Compute Handler Tests
- Entrypoints & Event Logging
- Tailnet Auth
- UI TypeScript Config
- Gateway Handler Tests
- UI Package Dependencies
- Compute Handler Implementation
- Graphify Skill Documentation
- Gateway Proxy Tests
- Compute Suspend Logic
- UI App Component
- Wake-on-LAN Sender
- Gateway Handler Implementation
- Gateway Proxy & Auto-Wake
- Tailnet Bind Safety
- Homelab Architecture Diagram
- Root Deploy Script
- Provisioning Deploy (Python)
- Provisioning Deploy (Shell)
- Favicon Asset (192x192)
- Favicon Asset (512x512)
- Favicon Asset (Apple Touch)
- Favicon Asset (16x16)
- Favicon Asset (32x32)
- Go Module Root
- Pyinfra Module Root
- Architecture Decisions
- Spec: Compute API autonomous suspend
- Domain Docs
- Issue tracker: Local Markdown
- triage-labels.md
- 01-compute-routes-to-containers-by-label.md
- 02-backup-scripts-can-hold-compute-awake.md
- 03-compute-suspends-itself-when-idle.md
- 04-validate-idle-watcher-without-suspending.md
- hold

## God Nodes (most connected - your core abstractions)
1. `mustNewHandler()` - 27 edges
2. `authedRequest()` - 18 edges
3. `CONTEXT.md (Domain Glossary)` - 17 edges
4. `compilerOptions` - 16 edges
5. `Handler` - 15 edges
6. `mustNewHandlerWithConfig()` - 15 edges
7. `mustNewHandler()` - 14 edges
8. `ARCHITECTURE.md (Homelab Control Plane)` - 14 edges
9. `NewHandler()` - 13 edges
10. `mustNewHandlerWithRuntime()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `Honesty Rules` --semantically_similar_to--> `Known Limitations (Flagged Transparency)`  [INFERRED] [semantically similar]
  .claude/skills/graphify/SKILL.md → ARCHITECTURE.md
- `Honesty Rules` --semantically_similar_to--> `Known Gaps (Flagged, Not Silently Dropped)`  [INFERRED] [semantically similar]
  .claude/skills/graphify/SKILL.md → provisioning/README.md
- `Graphify Project Integration Rules (root CLAUDE.md)` --references--> `/graphify Skill Pipeline`  [INFERRED]
  CLAUDE.md → .claude/skills/graphify/SKILL.md
- `reachabilityTracker` --conceptually_related_to--> `Reachable`  [INFERRED]
  control-plane/README.md → CONTEXT.md
- `autoWakeThrottler` --conceptually_related_to--> `Wake`  [INFERRED]
  control-plane/README.md → CONTEXT.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Graphify Slash-Command Activation Flow** — claude_claude_graphify_directive, claude_graphify_project_rules, claude_skills_graphify_skill_graphify_command [INFERRED 0.85]
- **Automatic Wake Trigger Mechanism** — context_wake, context_gateway_api, control_plane_readme_dowake, control_plane_readme_autowakethrottler [INFERRED 0.85]
- **Homelab Core Documentation Cross-reference Hub** — architecture_doc, context_doc, readme_doc, control_plane_readme_doc [INFERRED 0.75]

## Communities (39 total, 15 thin omitted)

### Community 0 - "Provisioning Facts & Deploy"
Cohesion: 0.07
Nodes (31): BaseSettings, DpkgArchitecture, has_device_role(), linux_codename(), linux_distro_id(), FactBase, Fact-derived helpers and the `device_role` guard shared by Deploy files. Importe, `dpkg --print-architecture` output (e.g. "amd64") -- pyinfra has no     built-in (+23 more)

### Community 1 - "Project Docs & Domain Glossary"
Cohesion: 0.20
Nodes (25): Deploys Are Manually Triggered Only, ARCHITECTURE.md (Homelab Control Plane), Known Limitations (Flagged Transparency), Honesty Rules, Compute, Compute API, Deploy, Deploy file (+17 more)

### Community 2 - "Compute Handler Tests"
Cohesion: 0.17
Nodes (31): fakeLogger, fakeSuspender, loggedEvent, authedRequest(), Mutex, Request, T, loopbackRequest() (+23 more)

### Community 3 - "Entrypoints & Event Logging"
Cohesion: 0.14
Nodes (17): main(), main(), EnvOr(), MustEnv(), MustEnvInt(), Client, NewGrafanaCloudLogger(), T (+9 more)

### Community 4 - "Tailnet Auth"
Cohesion: 0.16
Nodes (20): Handler, Request, remoteHost(), RequireTailnetIdentity(), RequireTailnetIdentityOrLoopback(), Handler, Request, ResponseWriter (+12 more)

### Community 5 - "UI TypeScript Config"
Cohesion: 0.09
Nodes (21): compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, moduleResolution (+13 more)

### Community 6 - "Gateway Handler Tests"
Cohesion: 0.13
Nodes (25): Request, ResponseWriter, NewHandler(), EventLogger, Handler, T, mustNewHandler(), TestNewHandlerRefusesOffTailnetBindHost() (+17 more)

### Community 7 - "UI Package Dependencies"
Cohesion: 0.10
Nodes (19): devDependencies, jsdom, typescript, vite, vitest, name, private, scripts (+11 more)

### Community 8 - "Compute Handler Implementation"
Cohesion: 0.10
Nodes (30): ContainerRuntime, EventLogger, lastProxied, reachabilityTracker, Suspender, Handler, Request, ResponseWriter (+22 more)

### Community 9 - "Graphify Skill Documentation"
Cohesion: 0.13
Nodes (16): Graphify Trigger Directive (user-level), Graphify Project Integration Rules (root CLAUDE.md), /graphify add & --watch Reference, Extra Exports & Benchmark Reference, Confidence Score Rubric, Extraction Subagent Prompt Spec, Node ID Format Rule, GitHub Clone & Cross-repo Merge Reference (+8 more)

### Community 10 - "Gateway Proxy Tests"
Cohesion: 0.38
Nodes (15): mustNewHandlerWithConfig(), closedPortTarget(), T, splitHostPort(), TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking(), TestProxy_AutoWakeDoesNotThrottleOnceTheWindowHasElapsed(), TestProxy_AutoWakeThrottlesRepeatCallsWithinTheWindow(), TestProxy_DoesNotFallBackToIndexHTMLForAnUnknownPath() (+7 more)

### Community 11 - "Compute Suspend Logic"
Cohesion: 0.32
Nodes (11): SystemSuspender, NewSystemSuspender(), NewSystemSuspenderWithCommand(), T, TestConstructorRejectsAnEmptyCommand(), TestDefaultCommandIsSystemctlSuspendOnly(), TestSuspendPropagatesErrorFromExitCode(), TestSuspendPropagatesErrorWhenCommandIsNotFound() (+3 more)

### Community 12 - "UI App Component"
Cohesion: 0.18
Nodes (4): MountOptions, mountUi(), ReachabilityState, STATUS_LABELS

### Community 13 - "Wake-on-LAN Sender"
Cohesion: 0.32
Nodes (9): BuildMagicPacket(), NewWakeOnLanSender(), T, TestBuildMagicPacketAcceptsHyphenSeparatedMac(), TestBuildMagicPacketAcceptsLowercaseMac(), TestBuildMagicPacketHasSixLeadingFFBytes(), TestBuildMagicPacketRejectsInvalidMac(), TestBuildMagicPacketRepeatsMacSixteenTimes() (+1 more)

### Community 14 - "Gateway Handler Implementation"
Cohesion: 0.22
Nodes (9): Handler, Mutex, Request, ResponseWriter, Time, newComputeProxy(), retryUntilReachable(), autoWakeThrottler (+1 more)

### Community 15 - "Gateway Proxy & Auto-Wake"
Cohesion: 0.13
Nodes (15): DockerRuntime, fakeContainerRuntime, RoutableContainer, Client, NewDockerRuntime(), routableFromContainers(), T, TestRoutableFromContainersMatchesByLabel() (+7 more)

### Community 16 - "Tailnet Bind Safety"
Cohesion: 0.31
Nodes (8): AssertTailnetOnlyBind(), appFactory(), T, TestAllowsLoopbackAndTailnetAddresses(), TestAppFactoryRefusesToStartOnOffTailnetHost(), TestAppFactoryStartsOnLoopbackHost(), TestRejectsOffTailnetAddresses(), BindOffTailnetError

### Community 17 - "Homelab Architecture Diagram"
Cohesion: 0.39
Nodes (8): Browser (tailnet), Compute API, Compute host (sleeps to RAM), Downstream workload proxy, Gateway (Pi Zero, always-on), Gateway API, Grafana Cloud (Loki push), UI (static SPA)

### Community 29 - "Architecture Decisions"
Cohesion: 0.14
Nodes (13): Architecture Decisions, Automatic suspend reuses the existing in-process Suspender, Compute API becomes Compute's own reverse proxy, Event log: automatic suspend and holds get their own `suspend_*` event types, Hold only blocks automatic suspend, never the manual `POST /suspend` button, Holds are HTTP-API-driven with a mandatory, fixed TTL — never indefinite, Idle signal combines container activity, HTTP traffic, and explicit holds — not container running-state alone, Idle watcher gets a dry-run mode, gated on whether it calls the Suspender — not on the suspend command (+5 more)

### Community 30 - "Spec: Compute API autonomous suspend"
Cohesion: 0.22
Nodes (8): Further Notes, Implementation Decisions, Out of Scope, Problem Statement, Solution, Spec: Compute API autonomous suspend, Testing Decisions, User Stories

### Community 31 - "Domain Docs"
Cohesion: 0.33
Nodes (5): Before exploring, read these, Domain Docs, File structure, Flag ADR conflicts, Use the glossary's vocabulary

### Community 32 - "Issue tracker: Local Markdown"
Cohesion: 0.33
Nodes (5): Conventions, Issue tracker: Local Markdown, Wayfinding operations, When a skill says "fetch the relevant ticket", When a skill says "publish to the issue tracker"

### Community 38 - "hold"
Cohesion: 0.43
Nodes (3): hold, EventLogger, Mutex

## Knowledge Gaps
- **83 isolated node(s):** `github.com/nikbogman/homelab/control-plane`, `Handler`, `lokiLine`, `contextKey`, `name` (+78 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **15 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewHandler()` connect `Gateway Handler Tests` to `Tailnet Auth`, `Gateway Proxy Tests`, `Wake-on-LAN Sender`, `Gateway Handler Implementation`, `Tailnet Bind Safety`?**
  _High betweenness centrality (0.130) - this node is a cross-community bridge._
- **Why does `NewHandler()` connect `Compute Handler Implementation` to `Tailnet Bind Safety`, `Compute Handler Tests`, `Tailnet Auth`?**
  _High betweenness centrality (0.104) - this node is a cross-community bridge._
- **Why does `RequireTailnetIdentityOrLoopback()` connect `Tailnet Auth` to `Compute Handler Implementation`, `Gateway Handler Tests`?**
  _High betweenness centrality (0.059) - this node is a cross-community bridge._
- **What connects `github.com/nikbogman/homelab/control-plane`, `Handler`, `lokiLine` to the rest of the system?**
  _83 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Provisioning Facts & Deploy` be split into smaller, more focused modules?**
  _Cohesion score 0.07474747474747474 - nodes in this community are weakly interconnected._
- **Should `Entrypoints & Event Logging` be split into smaller, more focused modules?**
  _Cohesion score 0.1422924901185771 - nodes in this community are weakly interconnected._
- **Should `UI TypeScript Config` be split into smaller, more focused modules?**
  _Cohesion score 0.09090909090909091 - nodes in this community are weakly interconnected._