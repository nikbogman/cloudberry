import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountUi } from './app'

const COMPUTE_API_BASE_URL = 'https://compute.example.ts.net'
const GATEWAY_API_BASE_URL = 'https://gateway.example.ts.net'

function statusElement(): HTMLElement {
  return document.querySelector('#reachability-status')!
}

function wakeButton(): HTMLButtonElement {
  return document.querySelector('#wake-button')!
}

function suspendButton(): HTMLButtonElement {
  return document.querySelector('#suspend-button')!
}

function refreshButton(): HTMLButtonElement {
  return document.querySelector('#refresh-button')!
}

function actionError(): HTMLElement {
  return document.querySelector('#action-error')!
}

function stubFetch(status: number): ReturnType<typeof vi.fn> {
  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('mountUi', () => {
  let container: HTMLDivElement
  let unmount: () => void

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
    vi.useFakeTimers()
  })

  afterEach(() => {
    unmount?.()
    container.remove()
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('shows Checking before the first poll resolves', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })

    expect(statusElement().textContent).toBe('Checking…')
    expect(statusElement().dataset.state).toBe('checking')
  })

  it('shows Reachable once the health check succeeds', async () => {
    stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(statusElement().textContent).toBe('Reachable')
    expect(fetch).toHaveBeenCalledWith(`${COMPUTE_API_BASE_URL}/health`, { signal: expect.any(AbortSignal) })
  })

  it('shows Unreachable when the health check fails to connect', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network error')))

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(statusElement().textContent).toBe('Unreachable')
  })

  it('shows Unreachable when the health check responds with a non-2xx status', async () => {
    stubFetch(502)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(statusElement().textContent).toBe('Unreachable')
  })

  it('polls again on the configured interval without a page refresh', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchMock).toHaveBeenCalledTimes(2)

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('stops polling once unmounted', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    unmount()
    await vi.advanceTimersByTimeAsync(30_000)

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('enables the wake button before the first poll resolves', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })

    expect(wakeButton().disabled).toBe(false)
  })

  it('disables the wake button once the server is reachable', async () => {
    stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(wakeButton().disabled).toBe(true)
  })

  it('re-enables the wake button once the server goes unreachable again', async () => {
    const fetchMock = stubFetch(200)
    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))
    expect(wakeButton().disabled).toBe(true)

    fetchMock.mockResolvedValue(new Response(null, { status: 502 }))
    await vi.advanceTimersByTimeAsync(10_000)
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(wakeButton().disabled).toBe(false)
  })

  it('calls the Gateway API when the wake button is clicked', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, {
      computeApiBaseUrl: COMPUTE_API_BASE_URL,
      gatewayApiBaseUrl: GATEWAY_API_BASE_URL,
    })
    wakeButton().click()

    expect(fetchMock).toHaveBeenCalledWith(`${GATEWAY_API_BASE_URL}/wake`, { method: 'POST' })
  })

  it('calls the Gateway API same-origin (relative path) when no gatewayApiBaseUrl is given', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    wakeButton().click()

    expect(fetchMock).toHaveBeenCalledWith('/wake', { method: 'POST' })
  })

  it('enables the suspend button before the first poll resolves', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })

    expect(suspendButton().disabled).toBe(false)
  })

  it('enables the suspend button once the server is reachable', async () => {
    stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(suspendButton().disabled).toBe(false)
  })

  it('disables the suspend button once the server goes unreachable', async () => {
    const fetchMock = stubFetch(200)
    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))
    expect(suspendButton().disabled).toBe(false)

    fetchMock.mockResolvedValue(new Response(null, { status: 502 }))
    await vi.advanceTimersByTimeAsync(10_000)
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(suspendButton().disabled).toBe(true)
  })

  it('calls the Compute API when the suspend button is clicked', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    suspendButton().click()

    expect(fetchMock).toHaveBeenCalledWith(`${COMPUTE_API_BASE_URL}/suspend`, { method: 'POST' })
  })

  it('shows a loading label and disables the button while wake is in flight', async () => {
    let resolveWake!: (value: Response) => void
    const fetchMock = vi.fn((url: string) => {
      if (url.endsWith('/wake')) return new Promise<Response>((resolve) => (resolveWake = resolve))
      return new Promise<Response>(() => {}) // health check never resolves; keep state 'checking'
    })
    vi.stubGlobal('fetch', fetchMock)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    expect(statusElement().dataset.state).toBe('checking')

    wakeButton().click()
    expect(wakeButton().textContent).toBe('Waking…')
    expect(wakeButton().disabled).toBe(true)
    expect(suspendButton().disabled).toBe(true)
    expect(refreshButton().disabled).toBe(true)

    resolveWake(new Response(null, { status: 200 }))
    await vi.waitFor(() => expect(wakeButton().textContent).toBe('Wake'))
    expect(wakeButton().disabled).toBe(false)
  })

  it('shows a visible error when the wake request fails to reach the server', async () => {
    stubFetch(502)
    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL, gatewayApiBaseUrl: GATEWAY_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    const fetchMock = vi.fn().mockRejectedValue(new Error('network error'))
    vi.stubGlobal('fetch', fetchMock)

    wakeButton().click()
    await vi.waitFor(() => expect(actionError().hidden).toBe(false))
    expect(actionError().textContent).toBe('Wake failed: could not reach the server.')
  })

  it('shows a visible error when the suspend request returns a non-2xx status', async () => {
    stubFetch(200)
    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 500 }))
    vi.stubGlobal('fetch', fetchMock)

    suspendButton().click()
    await vi.waitFor(() => expect(actionError().hidden).toBe(false))
    expect(actionError().textContent).toBe('Suspend failed (HTTP 500).')
  })

  it('clears a previous action error on the next action attempt', async () => {
    stubFetch(200)
    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network error')))
    suspendButton().click()
    await vi.waitFor(() => expect(actionError().hidden).toBe(false))

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 200 })))
    suspendButton().click()
    expect(actionError().hidden).toBe(true)
  })

  it('re-polls immediately when the refresh button is clicked', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL, intervalMs: 60_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    refreshButton().click()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })

  it('aborts a health check that hangs past the timeout, marking the server unreachable', async () => {
    const abortedSignals: AbortSignal[] = []
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      const signal = init?.signal as AbortSignal
      return new Promise<Response>((_resolve, reject) => {
        signal.addEventListener('abort', () => {
          abortedSignals.push(signal)
          reject(new DOMException('aborted', 'AbortError'))
        })
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    unmount = mountUi(container, { computeApiBaseUrl: COMPUTE_API_BASE_URL })
    expect(statusElement().dataset.state).toBe('checking')

    await vi.advanceTimersByTimeAsync(5_000)
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))
    expect(abortedSignals).toHaveLength(1)
  })
})
