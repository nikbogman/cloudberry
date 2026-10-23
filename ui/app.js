const STATUS_LABELS = {
  checking: 'Checking…',
  reachable: 'Reachable',
  unreachable: 'Unreachable',
}

// Lucide icon paths (ISC license), inlined so there's nothing to fetch.
const ICONS = {
  server:
    '<rect width="20" height="8" x="2" y="2" rx="2"/><rect width="20" height="8" x="2" y="14" rx="2"/><path d="M6 6h.01M6 18h.01"/>',
  power: '<path d="M12 2v10"/><path d="M18.4 6.6a9 9 0 1 1-12.77.04"/>',
  moon: '<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>',
  refresh:
    '<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>',
  alert: '<circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/>',
}
const icon = (name) => `<svg class="icon" viewBox="0 0 24 24" aria-hidden="true">${ICONS[name]}</svg>`

const DEFAULT_POLL_INTERVAL_MS = 12_000
// Caps how long a poll can hang in "Checking…" if the host is powered off.
const HEALTH_CHECK_TIMEOUT_MS = 5_000

/**
 * Renders the UI into `container` and starts polling. Returns a cleanup
 * function that stops polling.
 */
export function mountUi(container, options) {
  const { hostdBaseUrl, gatewayBaseUrl, intervalMs = DEFAULT_POLL_INTERVAL_MS } = options

  container.innerHTML = `
    <header>
      <img src="icons/logo.png" alt="" />
      <h1>Cloudberry</h1>
    </header>
    <main class="card" data-state="checking">
      <div class="host">
        <div class="host-icon">${icon('server')}</div>
        <div class="host-text">
          <p class="host-name">blackberry</p>
          <p id="reachability-status" aria-live="polite">${STATUS_LABELS.checking}</p>
          <p id="last-checked">Not checked yet</p>
        </div>
        <button id="refresh-button" type="button" aria-label="Refresh">${icon('refresh')}</button>
      </div>
      <div class="actions">
        <button id="wake-button" type="button">${icon('power')}<span>Wake</span></button>
        <button id="suspend-button" type="button">${icon('moon')}<span>Suspend</span></button>
      </div>
      <p id="action-error" role="alert" hidden>${icon('alert')}<span></span></p>
    </main>
    <footer>Works only from a device on the tailnet</footer>
  `
  const cardElement = container.querySelector('.card')
  const statusElement = container.querySelector('#reachability-status')
  const lastCheckedElement = container.querySelector('#last-checked')
  const wakeButton = container.querySelector('#wake-button')
  const suspendButton = container.querySelector('#suspend-button')
  const refreshButton = container.querySelector('#refresh-button')
  const errorElement = container.querySelector('#action-error')
  const errorText = errorElement.querySelector('span')

  let reachability = 'checking'
  let pollInFlight = false
  let actionPending = false

  // Disabled when the action wouldn't make sense for current reachability,
  // or when any action/poll is already in flight.
  const applyButtonState = () => {
    wakeButton.disabled = actionPending || reachability === 'reachable'
    suspendButton.disabled = actionPending || reachability === 'unreachable'
    refreshButton.disabled = actionPending || pollInFlight
    refreshButton.setAttribute('aria-busy', String(pollInFlight))
  }

  const setState = (state) => {
    reachability = state
    statusElement.textContent = STATUS_LABELS[state]
    cardElement.dataset.state = state
    applyButtonState()
  }

  const clearError = () => {
    errorElement.hidden = true
    errorText.textContent = ''
  }

  const showError = (message) => {
    errorElement.hidden = false
    errorText.textContent = message
  }

  const poll = async () => {
    pollInFlight = true
    setState('checking')
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), HEALTH_CHECK_TIMEOUT_MS)
    try {
      const response = await fetch(`${hostdBaseUrl}/health`, { signal: controller.signal })
      setState(response.ok ? 'reachable' : 'unreachable')
    } catch {
      setState('unreachable')
    } finally {
      lastCheckedElement.textContent = `Checked ${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`
      clearTimeout(timeoutId)
      pollInFlight = false
      applyButtonState()
    }
  }

  const runAction = async (button, idleLabel, pendingLabel, request) => {
    clearError()
    const label = button.querySelector('span')
    actionPending = true
    label.textContent = pendingLabel
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
      label.textContent = idleLabel
      applyButtonState()
    }
  }

  wakeButton.addEventListener('click', () => {
    void runAction(wakeButton, 'Wake', 'Waking…', () => fetch(`${gatewayBaseUrl}/wake`, { method: 'POST' }))
  })

  suspendButton.addEventListener('click', () => {
    // Suspend lives on hostd (not gateway) -- unlike Wake.
    void runAction(suspendButton, 'Suspend', 'Suspending…', () =>
      fetch(`${hostdBaseUrl}/suspend`, { method: 'POST' }),
    )
  })

  refreshButton.addEventListener('click', () => {
    void poll()
  })

  void poll()
  const timerId = setInterval(poll, intervalMs)

  return () => clearInterval(timerId)
}
