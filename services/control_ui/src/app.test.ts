import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountControlUi } from './app'

const BASE_URL = 'https://server.example.ts.net'

function statusElement(): HTMLElement {
  return document.querySelector('#reachability-status')!
}

function stubFetch(status: number): ReturnType<typeof vi.fn> {
  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('mountControlUi', () => {
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

    unmount = mountControlUi(container, { baseUrl: BASE_URL })

    expect(statusElement().textContent).toBe('Checking…')
    expect(statusElement().dataset.state).toBe('checking')
  })

  it('shows Reachable once the health check succeeds', async () => {
    stubFetch(200)

    unmount = mountControlUi(container, { baseUrl: BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(statusElement().textContent).toBe('Reachable')
    expect(fetch).toHaveBeenCalledWith(`${BASE_URL}/health`)
  })

  it('shows Unreachable when the health check fails to connect', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network error')))

    unmount = mountControlUi(container, { baseUrl: BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(statusElement().textContent).toBe('Unreachable')
  })

  it('shows Unreachable when the health check responds with a non-2xx status', async () => {
    stubFetch(502)

    unmount = mountControlUi(container, { baseUrl: BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(statusElement().textContent).toBe('Unreachable')
  })

  it('polls again on the configured interval without a page refresh', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountControlUi(container, { baseUrl: BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchMock).toHaveBeenCalledTimes(2)

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('stops polling once unmounted', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountControlUi(container, { baseUrl: BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    unmount()
    await vi.advanceTimersByTimeAsync(30_000)

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
