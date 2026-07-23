# Graph Report - .  (2026-07-23)

## Corpus Check
- Corpus is ~41,806 words - fits in a single context window. You may not need a graph.

## Summary
- 554 nodes · 746 edges · 44 communities (31 shown, 13 thin omitted)
- Extraction: 94% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 41 edges (avg confidence: 0.8)
- Token cost: 362,791 input · 0 output

## Community Hubs (Navigation)
- Server API Reachability
- Shared Git-Pull Deploy Helper
- Pi API Wake-on-LAN
- Graphify Skill Exports
- Deploy Domain Concepts
- Caddy Deploy ADR Decisions
- Homelab Domain Glossary
- Wake Plugin Go Internals
- Caddy handle_errors Fix
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
- Graphify Incremental Update
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
- Wake Plugin Go Module
- Pyinfra Inventory Name
- Control API Deploy File
- Host Role Detection Helpers
- Pyinfra Host Groups
- Control UI Favicon

## God Nodes (most connected - your core abstractions)
1. `Homelab README.md` - 24 edges
2. `compilerOptions` - 16 edges
3. `pyinfra Provisioning Spec` - 16 edges
4. `SystemSuspender` - 14 edges
5. `wake_plugin Go module` - 12 edges
6. `Issue 07: Caddy Deploy File` - 12 edges
7. `create_app()` - 10 edges
8. `build_magic_packet()` - 10 edges
9. `newWakeCaller()` - 10 edges
10. `AlloyLogger` - 10 edges

## Surprising Connections (you probably didn't know these)
- `pyinfra/deploy_tailscale.py` --references--> `TailscaleBackendState`  [EXTRACTED]
  .scratch/pyinfra-provisioning/issues/02-tailscale.md → pyinfra/deploy_tailscale.py
- `pyinfra Provisioning Spec` --references--> `ADR-0008: Full Caddyfile as Single Declarative Source of Truth`  [EXTRACTED]
  .scratch/pyinfra-provisioning/spec.md → docs/adr/0008-full-caddyfile-single-source-of-truth.md
- `Homelab README.md` --references--> `pyinfra/deploy.py`  [INFERRED]
  README.md → .scratch/pyinfra-provisioning/issues/08-deploy-entrypoint.md
- `Verification performed (go test, caddy validate/run)` --references--> `docs/agents/pyinfra-demo.md`  [EXTRACTED]
  services/pi-proxy/wake_plugin/README.md → .scratch/pyinfra-provisioning/issues/04-control-api-shared-deploy-helper.md
- `git_systemd_service live demo (ticket 04)` --references--> `api_deploy.py (shared git+systemd @deploy helper)`  [AMBIGUOUS]
  docs/agents/pyinfra-demo.md → pyinfra/README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **graphify /graphify Pipeline Steps** — claude_skills_graphify_skill_step0_github_clone, claude_skills_graphify_skill_step1_ensure_installed, claude_skills_graphify_skill_step2_detect_files, claude_skills_graphify_skill_step3_extraction, claude_skills_graphify_skill_step4_build_graph, claude_skills_graphify_skill_step5_label_communities, claude_skills_graphify_skill_step6_obsidian_html, claude_skills_graphify_skill_step9_manifest_cleanup [EXTRACTED 1.00]
- **Control Plane Ticket Build Sequence (01-05)** — scratch_homelab_control_system_issues_01_control_plane_auth_logging_scaffold_identity_middleware, scratch_homelab_control_system_issues_02_reachability_status_health_check_endpoint, scratch_homelab_control_system_issues_03_wake_action_wake_button, scratch_homelab_control_system_issues_04_suspend_action_suspend_button, scratch_homelab_control_system_issues_05_auto_wake_proxy_caddy_wol_config [EXTRACTED 1.00]
- **Auto-Wake Proxy Redesign (caddy-wol -> wake_plugin)** — scratch_caddy_wake_plugin_spec_caddy_wol, scratch_caddy_wake_plugin_spec_wake_plugin_module, scratch_caddy_wake_plugin_spec_adr_0012, scratch_homelab_control_system_issues_05_auto_wake_proxy_caddy_wol_config [INFERRED 0.85]
- **Deploy Entrypoint Composes All Deploy Files** — pyinfra_deploy_module, pyinfra_deploy_tailscale_module, pyinfra_deploy_docker_module, pyinfra_deploy_control_pi_api_module, pyinfra_deploy_control_server_api_module, pyinfra_deploy_caddy_module [EXTRACTED 1.00]
- **Shared git-pull+systemd Helper Reused by Both Control API Deploy Files** — pyinfra_control_api_deploy_git_systemd_service, pyinfra_deploy_control_pi_api_module, pyinfra_deploy_control_server_api_module [EXTRACTED 1.00]
- **Three-Tier Infrastructure Testing Pattern (--check, disposable containers, idempotency)** — pyinfra_readme_doc, scratch_pyinfra_provisioning_issues_09_disposable_container_test_harness_issue, pyinfra_inventory_module, docs_agents_pyinfra_demo_doc [EXTRACTED 0.90]
- **pyinfra's three-tier testing pattern (--dry, disposable containers, idempotency)** — pyinfra_readme_three_tier_testing, docs_agents_pyinfra_dry_flag, docs_agents_pyinfra_idempotency_mechanics, docs_agents_pyinfra_demo_highest_available_seam [EXTRACTED 1.00]
- **Server wake trigger flow (Control UI Wake button + Pi proxy auto-wake, both hitting Pi API /wake)** — services_control_ui_readme_wake_button, services_pi_api_readme_wake_endpoint, services_pi_proxy_wake_plugin_readme_call_wake_api_directive, docs_adr_0012_auto_wake_proxy_calls_control_pi_api_doc [EXTRACTED 1.00]
- **Tailnet-only bind safety pattern shared by Pi API and Server API** — services_shared_readme_bind_safety_py, services_pi_api_readme_pi_api_host, services_server_api_readme_server_api_host, docs_adr_0004_tailnet_membership_authorization_doc [EXTRACTED 1.00]

## Communities (44 total, 13 thin omitted)

### Community 0 - "Server API Reachability"
Cohesion: 0.05
Nodes (30): test_create_app_refuses_off_tailnet_bind_host(), create_app(), Flask, Server API: exposes a Reachable health check and a Suspend action.  Runs on the, Logs a `reachability_changed` event the first time this process     observes its, ReachabilityTracker, Suspend-to-RAM: runs the configured system command as a subprocess. This is the, Suspends the host to RAM by running the configured command. (+22 more)

### Community 1 - "Shared Git-Pull Deploy Helper"
Cohesion: 0.07
Nodes (33): BaseSettings, git_systemd_service(), Shared git-pull + systemd deploy helper.  The pattern common to both the Pi API, Pull `repo_url`@`ref` to `dest`, install/enable a systemd unit named     `unit_n, has_device_role(), linux_codename(), linux_distro_id(), Helpers shared by more than one Deploy file.  Not a Deploy file itself -- define (+25 more)

### Community 2 - "Pi API Wake-on-LAN"
Cohesion: 0.07
Nodes (21): create_app(), Flask, Pi API: sends a Wake-on-LAN packet to the main server on request.  Runs on the P, build_magic_packet(), Wake-on-LAN: builds and broadcasts the magic packet that wakes the main server., Broadcasts a WoL magic packet over UDP on the local L2 segment., WakeOnLanSender, Real entrypoint: wires `create_app` to environment-provided config so this can a (+13 more)

### Community 3 - "Graphify Skill Exports"
Cohesion: 0.06
Nodes (37): Graphify Slash-Command Trigger, FalkorDB Export, MCP stdio Server, Neo4j Export, Wiki Export, Confidence Score Rubric, Hyperedges Rule, Node ID Format Rule (+29 more)

### Community 4 - "Deploy Domain Concepts"
Cohesion: 0.13
Nodes (36): Deploy (concept), Deploy File (concept), Host Group (concept), ADR-0001: Direct Browser-to-API Calls, No Pi-Side Relay, ADR-0006: Git-Pull Deploy Model, Control UI Built Off-Device, ADR-0007: systemd Units for Control APIs, Not Docker, ADR-0009: Manual-Only Deploy Trigger, ADR-0010: Secrets via Dev-Machine Environment Variables (+28 more)

### Community 5 - "Caddy Deploy ADR Decisions"
Cohesion: 0.08
Nodes (36): ADR-0005: Dual wake paths (superseded), ADR-0008 (templated Caddyfile.j2 supersedes hand-edited Caddyfile), ADR-0009 (deploy.py never runs automatically), ADR-0011 (/server* route via server-side workload proxy), ADR-0012: Auto-wake proxy calls Control Pi API, @deploy decorator, Caddyfile.j2 validated with real caddy binary (ticket 07), git_systemd_service live demo (ticket 04) (+28 more)

### Community 6 - "Homelab Domain Glossary"
Cohesion: 0.08
Nodes (34): Homelab CLAUDE.md (Agent Skills Config), Control UI (concept), CONTEXT.md (Domain Glossary), Identity Header (concept), Pi API (concept), Pi Proxy (concept), Reachable (concept), Server API (concept) (+26 more)

### Community 7 - "Wake Plugin Go Internals"
Cohesion: 0.10
Nodes (24): Client, Context, Dispenser, Helper, MiddlewareHandler, ModuleInfo, Mutex, Request (+16 more)

### Community 8 - "Caddy handle_errors Fix"
Cohesion: 0.06
Nodes (34): ADR-0011, caddy validate Verification, handle_errors Site-Level Fix, handle_errors directive, handle_path directive, docs/agents/pyinfra-demo.md, ADR-0004 Loopback Auth Amendment, ADR-0005 (superseded framing) (+26 more)

### Community 9 - "Control Plane Shared Middleware"
Cohesion: 0.11
Nodes (26): Grafana Alloy Logging Client, Off-Tailnet Bind Assertion, control_plane_shared package (services/shared), Identity-Header Auth Decorator/Middleware, Control UI Polling (10-15s), CORS Allow-List, Health-Check Endpoint, Control UI Wake Button (+18 more)

### Community 10 - "Frontend TS Config"
Cohesion: 0.09
Nodes (21): DOM, ES2023, src, vite/client, compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, erasableSyntaxOnly (+13 more)

### Community 11 - "Control Plane ADR Decisions"
Cohesion: 0.14
Nodes (20): ADR-0001: Direct browser-to-API, no relay, ADR-0002: Suspend-only, no shutdown, ADR-0003: WoL same broadcast domain, ADR-0004: Tailnet membership authorization, ADR-0006: Git-pull deploy with off-device UI build, ADR-0007: systemd not Docker for control APIs, Off-device build + static dist/ deploy, GET /health polling (12s default) (+12 more)

### Community 12 - "Frontend Test Tooling"
Cohesion: 0.10
Nodes (19): jsdom, devDependencies, jsdom, typescript, vite, vitest, name, private (+11 more)

### Community 13 - "Tailnet Identity Auth"
Cohesion: 0.11
Nodes (7): get_caller_identity(), Tailnet identity-header auth for the control plane's Flask apps.  Both the Pi AP, Reject requests missing the Tailscale identity header with a 401., Return the caller's identity captured by `require_tailnet_identity`., Like `require_tailnet_identity`, but also accepts a caller on     127.0.0.1 with, require_tailnet_identity(), require_tailnet_identity_or_loopback()

### Community 14 - "Domain Docs Conventions"
Cohesion: 0.15
Nodes (13): CONTEXT.md, docs/adr/ (ADR directory), ADR conflict flagging practice, /domain-modeling skill, Glossary vocabulary discipline, /grill-with-docs skill, /improve-codebase-architecture skill, Single-context repo convention (+5 more)

### Community 15 - "Control UI App Mounting"
Cohesion: 0.22
Nodes (4): mountControlUi(), MountOptions, ReachabilityState, STATUS_LABELS

### Community 16 - "Graphify Query Traversal"
Cohesion: 0.29
Nodes (7): BFS/DFS Traversal Modes, graphify reflect / LESSONS.md, save-result Feedback Loop, Constrained Query Expansion, /graphify explain, /graphify path, /graphify query

### Community 17 - "Wayfinder Issue Tracker"
Cohesion: 0.50
Nodes (5): Blocked by: NN mechanism, Child ticket (wayfinder), Frontier scan (open/unblocked/unclaimed tickets), map.md (wayfinder map file), /wayfinder

### Community 18 - "Pyinfra Change Detection"
Cohesion: 0.50
Nodes (5): .will_change vs .did_change() distinction, OperationMeta.did_change(), @docker connector, Host group, OperationMeta.will_change

### Community 20 - "Graphify Incremental Update"
Cohesion: 0.50
Nodes (3): Code-Only Fast Path (skip LLM), detect_incremental(), --update flag

### Community 22 - "Wake Plugin Handler Tests"
Cohesion: 0.67
Nodes (3): T, TestHandler_ProvisionRequiresAWakeURL(), TestHandler_ServeHTTPCallsTheWakeURLAndAlwaysContinues()

### Community 23 - "Graphify Watch Mode"
Cohesion: 0.67
Nodes (3): Debounce (3s default), graphify.watch background watcher, --watch flag

### Community 24 - "Typed Settings Pattern"
Cohesion: 0.67
Nodes (3): ADR-0010 (typed env-var settings, no file-based secrets store), settings.py (typed pydantic-settings config), Typed pydantic-settings config pattern

### Community 25 - "Issue Tracker File Convention"
Cohesion: 0.67
Nodes (3): issues/<NN>-<slug>.md ticket files, .scratch/<feature-slug>/ convention, spec.md (per-feature spec/PRD)

### Community 26 - "Control Plane Services"
Cohesion: 0.67
Nodes (3): control-plane-shared, pi-api, server-api

## Ambiguous Edges - Review These
- `Control UI` → `.scratch/ Local Markdown Issue Tracker`  [AMBIGUOUS]
  .scratch/homelab-control-system/spec.md · relation: conceptually_related_to
- `git_systemd_service live demo (ticket 04)` → `api_deploy.py (shared git+systemd @deploy helper)`  [AMBIGUOUS]
  docs/agents/pyinfra-demo.md · relation: references

## Knowledge Gaps
- **127 isolated node(s):** `homelab-pyinfra`, `name`, `private`, `version`, `type` (+122 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **13 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Control UI` and `.scratch/ Local Markdown Issue Tracker`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `git_systemd_service live demo (ticket 04)` and `api_deploy.py (shared git+systemd @deploy helper)`?**
  _Edge tagged AMBIGUOUS (relation: references) - confidence is low._
- **Why does `Homelab README.md` connect `Homelab Domain Glossary` to `Deploy Domain Concepts`, `Caddy Deploy ADR Decisions`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **Why does `TailscaleBackendState` connect `Shared Git-Pull Deploy Helper` to `Deploy Domain Concepts`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Why does `pyinfra/deploy_tailscale.py` connect `Deploy Domain Concepts` to `Shared Git-Pull Deploy Helper`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `SystemSuspender` (e.g. with `ReachabilityTracker` and `system_suspender()`) actually correct?**
  _`SystemSuspender` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `wake_plugin Go module` (e.g. with `wol.py` and `Caddy WoL Plugin Config (initial)`) actually correct?**
  _`wake_plugin Go module` has 2 INFERRED edges - model-reasoned connections that need verification._