export type ReachabilityState = 'checking' | 'reachable' | 'unreachable'

const STATUS_LABELS: Record<ReachabilityState, string> = {
  checking: 'Checking…',
  reachable: 'Reachable',
  unreachable: 'Unreachable',
}

const DEFAULT_POLL_INTERVAL_MS = 12_000
// Bounds how long a poll can sit in "Checking…": without it, a fully powered-off
// host can leave the TCP connect hanging far longer than the poll interval.
const HEALTH_CHECK_TIMEOUT_MS = 5_000

export interface MountOptions {
  computeApiBaseUrl: string
  /** Same-origin as the UI by default, so '' (relative) works. */
  gatewayApiBaseUrl?: string
  intervalMs?: number
}

/** Returns a cleanup function that stops polling. */
export function mountUi(container: HTMLElement, options: MountOptions): () => void {
  const { computeApiBaseUrl, gatewayApiBaseUrl = '', intervalMs = DEFAULT_POLL_INTERVAL_MS } = options

  container.innerHTML = `
    <h1>Homelab Control</h1>
    <p id="reachability-status" data-state="checking">${STATUS_LABELS.checking}</p>
    <div class="actions">
      <button id="wake-button" type="button">Wake</button>
      <button id="suspend-button" type="button">Suspend</button>
      <button id="refresh-button" type="button">Refresh</button>
    </div>
    <p id="action-error" role="alert" hidden></p>
  `
  const statusElement = container.querySelector<HTMLParagraphElement>('#reachability-status')!
  const wakeButton = container.querySelector<HTMLButtonElement>('#wake-button')!
  const suspendButton = container.querySelector<HTMLButtonElement>('#suspend-button')!
  const refreshButton = container.querySelector<HTMLButtonElement>('#refresh-button')!
  const errorElement = container.querySelector<HTMLParagraphElement>('#action-error')!

  let reachability: ReachabilityState = 'checking'
  let pollInFlight = false
  let actionPending = false

  // Each button is disabled whenever its action wouldn't make sense given
  // the last known reachability state, plus whenever any action/poll is
  // already in flight (actionPending and pollInFlight are tracked
  // separately since a poll may land while an action is pending, or vice versa).
  const applyButtonState = () => {
    wakeButton.disabled = actionPending || reachability === 'reachable'
    suspendButton.disabled = actionPending || reachability === 'unreachable'
    refreshButton.disabled = actionPending || pollInFlight
  }

  const setState = (state: ReachabilityState) => {
    reachability = state
    statusElement.textContent = STATUS_LABELS[state]
    statusElement.dataset.state = state
    applyButtonState()
  }

  const clearError = () => {
    errorElement.hidden = true
    errorElement.textContent = ''
  }

  const showError = (message: string) => {
    errorElement.hidden = false
    errorElement.textContent = message
  }

  const poll = async () => {
    pollInFlight = true
    setState('checking')
    const controller = new AbortController()
    const timeoutId = window.setTimeout(() => controller.abort(), HEALTH_CHECK_TIMEOUT_MS)
    try {
      const response = await fetch(`${computeApiBaseUrl}/health`, { signal: controller.signal })
      setState(response.ok ? 'reachable' : 'unreachable')
    } catch {
      setState('unreachable')
    } finally {
      window.clearTimeout(timeoutId)
      pollInFlight = false
      applyButtonState()
    }
  }

  const runAction = async (
    button: HTMLButtonElement,
    idleLabel: string,
    pendingLabel: string,
    request: () => Promise<Response>,
  ) => {
    clearError()
    actionPending = true
    button.textContent = pendingLabel
    applyButtonState()
    try {
      const response = await request()
      if (!response.ok) {
        showError(`${idleLabel} failed (HTTP ${response.status}).`)
      }
    } catch {
      showError(`${idleLabel} failed: could not reach the server.`)
    } finally {
      actionPending = false
      button.textContent = idleLabel
      applyButtonState()
    }
  }

  wakeButton.addEventListener('click', () => {
    void runAction(wakeButton, 'Wake', 'Waking…', () => fetch(`${gatewayApiBaseUrl}/wake`, { method: 'POST' }))
  })

  suspendButton.addEventListener('click', () => {
    // Suspend lives on the Compute API (not the Gateway API) -- unlike Wake.
    void runAction(suspendButton, 'Suspend', 'Suspending…', () =>
      fetch(`${computeApiBaseUrl}/suspend`, { method: 'POST' }),
    )
  })

  refreshButton.addEventListener('click', () => {
    void poll()
  })

  void poll()
  const timerId = window.setInterval(poll, intervalMs)

  return () => window.clearInterval(timerId)
}
