# Graph Report - .  (2026-08-01)

## Corpus Check
- Corpus is ~33,598 words - fits in a single context window. You may not need a graph.

## Summary
- 351 nodes · 610 edges · 29 communities (19 shown, 10 thin omitted)
- Extraction: 88% EXTRACTED · 12% INFERRED · 0% AMBIGUOUS · INFERRED: 74 edges (avg confidence: 0.82)
- Token cost: 279,924 input · 0 output

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

## God Nodes (most connected - your core abstractions)
1. `mustNewHandler()` - 19 edges
2. `CONTEXT.md (Domain Glossary)` - 17 edges
3. `compilerOptions` - 16 edges
4. `mustNewHandlerWithConfig()` - 15 edges
5. `mustNewHandler()` - 14 edges
6. `ARCHITECTURE.md (Homelab Control Plane)` - 14 edges
7. `NewHandler()` - 13 edges
8. `/graphify Skill Pipeline` - 13 edges
9. `authedRequest()` - 12 edges
10. `NewHandler()` - 9 edges

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

## Communities (29 total, 10 thin omitted)

### Community 0 - "Provisioning Facts & Deploy"
Cohesion: 0.07
Nodes (31): BaseSettings, DpkgArchitecture, has_device_role(), linux_codename(), linux_distro_id(), FactBase, Helpers shared by more than one Deploy file.  Not a Deploy file itself -- define, Native apt architecture (e.g. "amd64", "arm64") per `dpkg     --print-architectu (+23 more)

### Community 1 - "Project Docs & Domain Glossary"
Cohesion: 0.20
Nodes (25): Deploys Are Manually Triggered Only, ARCHITECTURE.md (Homelab Control Plane), Known Limitations (Flagged Transparency), Honesty Rules, Compute, Compute API, Deploy, Deploy file (+17 more)

### Community 2 - "Compute Handler Tests"
Cohesion: 0.19
Nodes (23): fakeLogger, fakeSuspender, loggedEvent, authedRequest(), EventLogger, Handler, Request, T (+15 more)

### Community 3 - "Entrypoints & Event Logging"
Cohesion: 0.14
Nodes (17): Client, main(), main(), EnvOr(), MustEnv(), MustEnvInt(), NewGrafanaCloudLogger(), T (+9 more)

### Community 4 - "Tailnet Auth"
Cohesion: 0.16
Nodes (21): GetCallerIdentity(), Handler, Request, remoteHost(), RequireTailnetIdentity(), RequireTailnetIdentityOrLoopback(), Handler, Request (+13 more)

### Community 5 - "UI TypeScript Config"
Cohesion: 0.09
Nodes (21): compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, moduleResolution (+13 more)

### Community 6 - "Gateway Handler Tests"
Cohesion: 0.20
Nodes (17): EventLogger, Handler, T, mustNewHandler(), TestNewHandlerRefusesOffTailnetBindHost(), TestNewHandlerRejectsAMalformedMacAddressAtStartup(), TestWakeAllowsALoopbackCallerWithNoIdentityHeader(), TestWakeDoesNotGuardAgainstRepeatedRequests() (+9 more)

### Community 7 - "UI Package Dependencies"
Cohesion: 0.10
Nodes (19): devDependencies, jsdom, typescript, vite, vitest, name, private, scripts (+11 more)

### Community 8 - "Compute Handler Implementation"
Cohesion: 0.20
Nodes (12): EventLogger, Handler, reachabilityTracker, Suspender, Request, ResponseWriter, NewHandler(), Copy() (+4 more)

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
Cohesion: 0.33
Nodes (8): Request, ResponseWriter, NewHandler(), FS, Config, EventLogger, Handler, Waker

### Community 15 - "Gateway Proxy & Auto-Wake"
Cohesion: 0.22
Nodes (9): Handler, Request, ResponseWriter, newComputeProxy(), retryUntilReachable(), autoWakeThrottler, Mutex, RoundTripper (+1 more)

### Community 16 - "Tailnet Bind Safety"
Cohesion: 0.31
Nodes (8): AssertTailnetOnlyBind(), appFactory(), T, TestAllowsLoopbackAndTailnetAddresses(), TestAppFactoryRefusesToStartOnOffTailnetHost(), TestAppFactoryStartsOnLoopbackHost(), TestRejectsOffTailnetAddresses(), BindOffTailnetError

### Community 17 - "Homelab Architecture Diagram"
Cohesion: 0.39
Nodes (8): Browser (tailnet), Compute API, Compute host (sleeps to RAM), Downstream workload proxy, Gateway (Pi Zero, always-on), Gateway API, Grafana Cloud (Loki push), UI (static SPA)

## Knowledge Gaps
- **50 isolated node(s):** `github.com/nikbogman/homelab/control-plane`, `lokiLine`, `contextKey`, `name`, `private` (+45 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewHandler()` connect `Gateway Handler Implementation` to `Tailnet Auth`, `Gateway Handler Tests`, `Gateway Proxy Tests`, `Wake-on-LAN Sender`, `Gateway Proxy & Auto-Wake`, `Tailnet Bind Safety`?**
  _High betweenness centrality (0.186) - this node is a cross-community bridge._
- **Why does `BuildMagicPacket()` connect `Wake-on-LAN Sender` to `Gateway Handler Implementation`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Why does `AssertTailnetOnlyBind()` connect `Tailnet Bind Safety` to `Compute Handler Implementation`, `Gateway Handler Implementation`?**
  _High betweenness centrality (0.091) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `mustNewHandlerWithConfig()` (e.g. with `NewHandler()` and `TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking()`) actually correct?**
  _`mustNewHandlerWithConfig()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/nikbogman/homelab/control-plane`, `lokiLine`, `contextKey` to the rest of the system?**
  _50 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Provisioning Facts & Deploy` be split into smaller, more focused modules?**
  _Cohesion score 0.07474747474747474 - nodes in this community are weakly interconnected._
- **Should `Entrypoints & Event Logging` be split into smaller, more focused modules?**
  _Cohesion score 0.1422924901185771 - nodes in this community are weakly interconnected._