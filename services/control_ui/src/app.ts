export type ReachabilityState = 'checking' | 'reachable' | 'unreachable'

const STATUS_LABELS: Record<ReachabilityState, string> = {
  checking: 'Checking…',
  reachable: 'Reachable',
  unreachable: 'Unreachable',
}

const DEFAULT_POLL_INTERVAL_MS = 12_000

export interface MountOptions {
  baseUrl: string
  intervalMs?: number
}

/** Renders the Control UI into `container` and starts polling the Control
 * server API's health check. Returns a cleanup function that stops polling. */
export function mountControlUi(container: HTMLElement, options: MountOptions): () => void {
  const { baseUrl, intervalMs = DEFAULT_POLL_INTERVAL_MS } = options

  container.innerHTML = `
    <h1>Homelab Control</h1>
    <p id="reachability-status" data-state="checking">${STATUS_LABELS.checking}</p>
  `
  const statusElement = container.querySelector<HTMLParagraphElement>('#reachability-status')!

  const setState = (state: ReachabilityState) => {
    statusElement.textContent = STATUS_LABELS[state]
    statusElement.dataset.state = state
  }

  const poll = async () => {
    try {
      const response = await fetch(`${baseUrl}/health`)
      setState(response.ok ? 'reachable' : 'unreachable')
    } catch {
      setState('unreachable')
    }
  }

  void poll()
  const timerId = window.setInterval(poll, intervalMs)

  return () => window.clearInterval(timerId)
}
