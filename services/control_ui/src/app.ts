export type ReachabilityState = 'checking' | 'reachable' | 'unreachable'

const STATUS_LABELS: Record<ReachabilityState, string> = {
  checking: 'Checking…',
  reachable: 'Reachable',
  unreachable: 'Unreachable',
}

const DEFAULT_POLL_INTERVAL_MS = 12_000

export interface MountOptions {
  /** Server API origin. */
  serverApiBaseUrl: string
  /** Pi API origin — same-origin as the Control UI by default, so '' (relative) works. */
  piApiBaseUrl?: string
  intervalMs?: number
}

/** Renders the Control UI into `container`, starts polling the Server
 * API's health check, and wires the Wake button (Pi API)
 * and the Suspend button (Server API). Returns a cleanup function
 * that stops polling. */
export function mountControlUi(container: HTMLElement, options: MountOptions): () => void {
  const { serverApiBaseUrl, piApiBaseUrl = '', intervalMs = DEFAULT_POLL_INTERVAL_MS } = options

  container.innerHTML = `
    <h1>Homelab Control</h1>
    <p id="reachability-status" data-state="checking">${STATUS_LABELS.checking}</p>
    <button id="wake-button" type="button">Wake</button>
    <button id="suspend-button" type="button">Suspend</button>
  `
  const statusElement = container.querySelector<HTMLParagraphElement>('#reachability-status')!
  const wakeButton = container.querySelector<HTMLButtonElement>('#wake-button')!
  const suspendButton = container.querySelector<HTMLButtonElement>('#suspend-button')!

  const setState = (state: ReachabilityState) => {
    statusElement.textContent = STATUS_LABELS[state]
    statusElement.dataset.state = state
    wakeButton.disabled = state === 'reachable'
    suspendButton.disabled = state === 'unreachable'
  }

  const poll = async () => {
    try {
      const response = await fetch(`${serverApiBaseUrl}/health`)
      setState(response.ok ? 'reachable' : 'unreachable')
    } catch {
      setState('unreachable')
    }
  }

  wakeButton.addEventListener('click', () => {
    // Fire-and-forget; swallow rejections so a network error here can't
    // surface as an unhandled promise rejection.
    void fetch(`${piApiBaseUrl}/wake`, { method: 'POST' }).catch(() => {})
  })

  suspendButton.addEventListener('click', () => {
    // Fire-and-forget, same rationale as the Wake button above. Suspend
    // lives on the Server API (not the Pi API) — unlike Wake.
    void fetch(`${serverApiBaseUrl}/suspend`, { method: 'POST' }).catch(() => {})
  })

  void poll()
  const timerId = window.setInterval(poll, intervalMs)

  return () => window.clearInterval(timerId)
}
