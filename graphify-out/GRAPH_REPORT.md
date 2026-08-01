# Graph Report - homelab  (2026-08-01)

## Corpus Check
- 57 files · ~33,595 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 419 nodes · 629 edges · 33 communities (25 shown, 8 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 57 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `90a922ca`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Server API Reachability
- Shared Git-Pull Deploy Helper
- Pi API Wake-on-LAN
- Graphify Skill Exports
- Deploy Domain Concepts
- Homelab Domain Glossary
- Wake Plugin Go Internals
- Architecture
- Control Plane Shared Middleware
- Frontend TS Config
- Control Plane ADR Decisions
- Frontend Test Tooling
- Tailnet Identity Auth
- Control UI App Mounting
- Graphify Query Traversal
- WSGI Bootstrap Tests
- Graphify Incremental Update
- WSGI Bootstrap Tests
- /graphify command
- Graphify Watch Mode
- Typed Settings Pattern
- Control Plane Services
- Graphify URL Ingest
- Token Reduction Benchmark
- WoL Broadcast Domain ADR
- Pyinfra Inventory Name
- Pyinfra Host Groups
- Control UI Favicon
- AssertTailnetOnlyBind
- github.com/nikbogman/homelab/control-plane
- deploy.sh
- 0013-caddy-binary-built-off-device-by-pyinfra.md

## God Nodes (most connected - your core abstractions)
1. `mustNewHandler()` - 19 edges
2. `compilerOptions` - 16 edges
3. `mustNewHandlerWithConfig()` - 15 edges
4. `Homelab README.md` - 15 edges
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
- `Homelab README.md` --references--> `docs/agents/domain.md`  [EXTRACTED]
  README.md → CLAUDE.md
- `Homelab README.md` --references--> `docs/agents/issue-tracker.md`  [EXTRACTED]
  README.md → CLAUDE.md
- `Homelab README.md` --references--> `docs/agents/triage-labels.md`  [EXTRACTED]
  README.md → CLAUDE.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **graphify /graphify Pipeline Steps** — claude_skills_graphify_skill_step0_github_clone, claude_skills_graphify_skill_step1_ensure_installed, claude_skills_graphify_skill_step2_detect_files, claude_skills_graphify_skill_step3_extraction, claude_skills_graphify_skill_step4_build_graph, claude_skills_graphify_skill_step5_label_communities, claude_skills_graphify_skill_step6_obsidian_html, claude_skills_graphify_skill_step9_manifest_cleanup [EXTRACTED 1.00]

## Communities (33 total, 8 thin omitted)

### Community 0 - "Server API Reachability"
Cohesion: 0.16
Nodes (16): Homelab CLAUDE.md (Agent Skills Config), CONTEXT.md (Domain Glossary), docs/agents/domain.md, docs/agents/issue-tracker.md, docs/agents/triage-labels.md, Homelab README.md, services/control_ui, services/control_ui/README.md (+8 more)

### Community 1 - "Shared Git-Pull Deploy Helper"
Cohesion: 0.07
Nodes (31): BaseSettings, DpkgArchitecture, has_device_role(), linux_codename(), linux_distro_id(), FactBase, Helpers shared by more than one Deploy file.  Not a Deploy file itself -- define, Native apt architecture (e.g. "amd64", "arm64") per `dpkg     --print-architectu (+23 more)

### Community 2 - "Pi API Wake-on-LAN"
Cohesion: 0.20
Nodes (17): EventLogger, Handler, T, mustNewHandler(), TestNewHandlerRefusesOffTailnetBindHost(), TestNewHandlerRejectsAMalformedMacAddressAtStartup(), TestWakeAllowsALoopbackCallerWithNoIdentityHeader(), TestWakeDoesNotGuardAgainstRepeatedRequests() (+9 more)

### Community 3 - "Graphify Skill Exports"
Cohesion: 0.10
Nodes (22): FalkorDB Export, Neo4j Export, Wiki Export, Confidence Score Rubric, Hyperedges Rule, Node ID Format Rule, Extraction Subagent Prompt, graphify claude install (+14 more)

### Community 4 - "Deploy Domain Concepts"
Cohesion: 0.67
Nodes (3): Deploy (concept), Deploy File (concept), Host Group (concept)

### Community 6 - "Homelab Domain Glossary"
Cohesion: 0.20
Nodes (12): EventLogger, Handler, reachabilityTracker, Suspender, Request, ResponseWriter, NewHandler(), Copy() (+4 more)

### Community 7 - "Wake Plugin Go Internals"
Cohesion: 0.13
Nodes (17): Request, ResponseWriter, NewHandler(), Handler, Request, ResponseWriter, newComputeProxy(), retryUntilReachable() (+9 more)

### Community 8 - "Architecture"
Cohesion: 0.07
Nodes (25): Architecture, Architecture, Constraints, Data Flow, Domain Model, Extension Points, External integrations, How they interact (+17 more)

### Community 9 - "Control Plane Shared Middleware"
Cohesion: 0.38
Nodes (15): mustNewHandlerWithConfig(), closedPortTarget(), T, splitHostPort(), TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking(), TestProxy_AutoWakeDoesNotThrottleOnceTheWindowHasElapsed(), TestProxy_AutoWakeThrottlesRepeatCallsWithinTheWindow(), TestProxy_DoesNotFallBackToIndexHTMLForAnUnknownPath() (+7 more)

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

### Community 15 - "Control UI App Mounting"
Cohesion: 0.18
Nodes (4): MountOptions, mountUi(), ReachabilityState, STATUS_LABELS

### Community 16 - "Graphify Query Traversal"
Cohesion: 0.29
Nodes (7): BFS/DFS Traversal Modes, graphify reflect / LESSONS.md, save-result Feedback Loop, Constrained Query Expansion, /graphify explain, /graphify path, /graphify query

### Community 19 - "WSGI Bootstrap Tests"
Cohesion: 0.32
Nodes (9): BuildMagicPacket(), NewWakeOnLanSender(), T, TestBuildMagicPacketAcceptsHyphenSeparatedMac(), TestBuildMagicPacketAcceptsLowercaseMac(), TestBuildMagicPacketHasSixLeadingFFBytes(), TestBuildMagicPacketRejectsInvalidMac(), TestBuildMagicPacketRepeatsMacSixteenTimes() (+1 more)

### Community 20 - "Graphify Incremental Update"
Cohesion: 0.50
Nodes (3): Code-Only Fast Path (skip LLM), detect_incremental(), --update flag

### Community 21 - "WSGI Bootstrap Tests"
Cohesion: 0.32
Nodes (11): SystemSuspender, NewSystemSuspender(), NewSystemSuspenderWithCommand(), T, TestConstructorRejectsAnEmptyCommand(), TestDefaultCommandIsSystemctlSuspendOnly(), TestSuspendPropagatesErrorFromExitCode(), TestSuspendPropagatesErrorWhenCommandIsNotFound() (+3 more)

### Community 22 - "/graphify command"
Cohesion: 0.13
Nodes (15): Graphify Slash-Command Trigger, MCP stdio Server, graphify clone, graphify merge-graphs, Monorepo Multi-Subfolder Flow, Domain-Hint Whisper Prompt, Whisper Transcription, graphify (+7 more)

### Community 23 - "Graphify Watch Mode"
Cohesion: 0.67
Nodes (3): Debounce (3s default), graphify.watch background watcher, --watch flag

### Community 24 - "Typed Settings Pattern"
Cohesion: 0.29
Nodes (7): Control UI (concept), Pi API (concept), Pi Proxy (concept), Reachable (concept), Server API (concept), Suspend (concept), Wake (concept)

### Community 26 - "Control Plane Services"
Cohesion: 0.16
Nodes (21): GetCallerIdentity(), Handler, Request, remoteHost(), RequireTailnetIdentity(), RequireTailnetIdentityOrLoopback(), Handler, Request (+13 more)

### Community 41 - "AssertTailnetOnlyBind"
Cohesion: 0.31
Nodes (8): AssertTailnetOnlyBind(), appFactory(), T, TestAllowsLoopbackAndTailnetAddresses(), TestAppFactoryRefusesToStartOnOffTailnetHost(), TestAppFactoryStartsOnLoopbackHost(), TestRejectsOffTailnetAddresses(), BindOffTailnetError

### Community 46 - "0013-caddy-binary-built-off-device-by-pyinfra.md"
Cohesion: 0.13
Nodes (15): Configuration, deploy_compute_api.py (`ComputeApiSettings`, `ComputeApiSecrets`), deploy_gateway.py (`GatewaySettings`, `GatewaySecrets`), deploy_tailscale.py (`TailscaleSettings`), inventory.py (`InventorySettings`), Known gaps (flagged, not silently dropped), Layout, pyinfra provisioning (+7 more)

## Knowledge Gaps
- **105 isolated node(s):** `github.com/nikbogman/homelab/control-plane`, `lokiLine`, `contextKey`, `name`, `private` (+100 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **8 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewHandler()` connect `Wake Plugin Go Internals` to `Pi API Wake-on-LAN`, `AssertTailnetOnlyBind`, `Control Plane Shared Middleware`, `WSGI Bootstrap Tests`, `Control Plane Services`?**
  _High betweenness centrality (0.130) - this node is a cross-community bridge._
- **Why does `BuildMagicPacket()` connect `WSGI Bootstrap Tests` to `Wake Plugin Go Internals`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **Why does `AssertTailnetOnlyBind()` connect `AssertTailnetOnlyBind` to `Homelab Domain Glossary`, `Wake Plugin Go Internals`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `mustNewHandlerWithConfig()` (e.g. with `NewHandler()` and `TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking()`) actually correct?**
  _`mustNewHandlerWithConfig()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/nikbogman/homelab/control-plane`, `lokiLine`, `contextKey` to the rest of the system?**
  _105 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Shared Git-Pull Deploy Helper` be split into smaller, more focused modules?**
  _Cohesion score 0.07474747474747474 - nodes in this community are weakly interconnected._
- **Should `Graphify Skill Exports` be split into smaller, more focused modules?**
  _Cohesion score 0.09956709956709957 - nodes in this community are weakly interconnected._