# Suspend while work is in flight

Suspend never waits for in-flight work (see `cmd/hostd/README.md`). Two systemd-side
fixes, neither of which makes hostd learn about Docker. Inhibitors are held per
command, which suits one-shot jobs, not always-up containers.

- **Seconds of grace**: run the job under `systemd-inhibit --mode=delay
  --what=sleep`, and raise `InhibitDelayMaxSec` (default 5s) in `logind.conf`.
  Suspend waits for the job, then proceeds anyway once the cap expires.
- **Refuse while busy**: run the job under `systemd-inhibit --what=sleep`
  (block mode) and add `--check-inhibitors=yes` to `DefaultSuspendCommand`;
  `systemctl suspend` defaults to `auto`, which honors block inhibitors only
  when invoked from a TTY. The UI already renders the resulting 500 as
  "Suspend failed". Pair it with `CombinedOutput()` in `suspend.go` so the log
  can tell busy from broken.
