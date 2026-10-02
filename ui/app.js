const STATUS_LABELS = {
  checking: 'Checking…',
  reachable: 'Reachable',
  unreachable: 'Unreachable',
}

const DEFAULT_POLL_INTERVAL_MS = 12_000
// Caps how long a poll can hang in "Checking…" if the host is powered off.
const HEALTH_CHECK_TIMEOUT_MS = 5_000

/**
 * Renders the UI into `container` and starts polling. Returns a cleanup
 * function that stops polling.
 */
export function mountUi(container, options) {
  const { hostdBaseUrl, wakerBaseUrl, intervalMs = DEFAULT_POLL_INTERVAL_MS } = options

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
  const statusElement = container.querySelector('#reachability-status')
  const wakeButton = container.querySelector('#wake-button')
  const suspendButton = container.querySelector('#suspend-button')
  const refreshButton = container.querySelector('#refresh-button')
  const errorElement = container.querySelector('#action-error')

  let reachability = 'checking'
  let pollInFlight = false
  let actionPending = false

  // Disabled when the action wouldn't make sense for current reachability,
  // or when any action/poll is already in flight.
  const applyButtonState = () => {
    wakeButton.disabled = actionPending || reachability === 'reachable'
    suspendButton.disabled = actionPending || reachability === 'unreachable'
    refreshButton.disabled = actionPending || pollInFlight
  }

  const setState = (state) => {
    reachability = state
    statusElement.textContent = STATUS_LABELS[state]
    statusElement.dataset.state = state
    applyButtonState()
  }

  const clearError = () => {
    errorElement.hidden = true
    errorElement.textContent = ''
  }

  const showError = (message) => {
    errorElement.hidden = false
    errorElement.textContent = message
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
      clearTimeout(timeoutId)
      pollInFlight = false
      applyButtonState()
    }
  }

  const runAction = async (button, idleLabel, pendingLabel, request) => {
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
    void runAction(wakeButton, 'Wake', 'Waking…', () => fetch(`${wakerBaseUrl}/wake`, { method: 'POST' }))
  })

  suspendButton.addEventListener('click', () => {
    // Suspend lives on hostd (not waker) -- unlike Wake.
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
