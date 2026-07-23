import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountUi } from './app'

const SERVER_API_BASE_URL = 'https://server.example.ts.net'
const PI_API_BASE_URL = 'https://pi.example.ts.net'

function statusElement(): HTMLElement {
  return document.querySelector('#reachability-status')!
}

function wakeButton(): HTMLButtonElement {
  return document.querySelector('#wake-button')!
}

function suspendButton(): HTMLButtonElement {
  return document.querySelector('#suspend-button')!
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

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })

    expect(statusElement().textContent).toBe('Checking…')
    expect(statusElement().dataset.state).toBe('checking')
  })

  it('shows Reachable once the health check succeeds', async () => {
    stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(statusElement().textContent).toBe('Reachable')
    expect(fetch).toHaveBeenCalledWith(`${SERVER_API_BASE_URL}/health`)
  })

  it('shows Unreachable when the health check fails to connect', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network error')))

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(statusElement().textContent).toBe('Unreachable')
  })

  it('shows Unreachable when the health check responds with a non-2xx status', async () => {
    stubFetch(502)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(statusElement().textContent).toBe('Unreachable')
  })

  it('polls again on the configured interval without a page refresh', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchMock).toHaveBeenCalledTimes(2)

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('stops polling once unmounted', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    unmount()
    await vi.advanceTimersByTimeAsync(30_000)

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('enables the wake button before the first poll resolves', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })

    expect(wakeButton().disabled).toBe(false)
  })

  it('disables the wake button once the server is reachable', async () => {
    stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(wakeButton().disabled).toBe(true)
  })

  it('re-enables the wake button once the server goes unreachable again', async () => {
    const fetchMock = stubFetch(200)
    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))
    expect(wakeButton().disabled).toBe(true)

    fetchMock.mockResolvedValue(new Response(null, { status: 502 }))
    await vi.advanceTimersByTimeAsync(10_000)
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(wakeButton().disabled).toBe(false)
  })

  it('calls the Pi API when the wake button is clicked', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, {
      serverApiBaseUrl: SERVER_API_BASE_URL,
      piApiBaseUrl: PI_API_BASE_URL,
    })
    wakeButton().click()

    expect(fetchMock).toHaveBeenCalledWith(`${PI_API_BASE_URL}/wake`, { method: 'POST' })
  })

  it('calls the Pi API same-origin (relative path) when no piApiBaseUrl is given', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    wakeButton().click()

    expect(fetchMock).toHaveBeenCalledWith('/wake', { method: 'POST' })
  })

  it('enables the suspend button before the first poll resolves', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })

    expect(suspendButton().disabled).toBe(false)
  })

  it('enables the suspend button once the server is reachable', async () => {
    stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))

    expect(suspendButton().disabled).toBe(false)
  })

  it('disables the suspend button once the server goes unreachable', async () => {
    const fetchMock = stubFetch(200)
    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL, intervalMs: 10_000 })
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('reachable'))
    expect(suspendButton().disabled).toBe(false)

    fetchMock.mockResolvedValue(new Response(null, { status: 502 }))
    await vi.advanceTimersByTimeAsync(10_000)
    await vi.waitFor(() => expect(statusElement().dataset.state).toBe('unreachable'))

    expect(suspendButton().disabled).toBe(true)
  })

  it('calls the Server API when the suspend button is clicked', async () => {
    const fetchMock = stubFetch(200)

    unmount = mountUi(container, { serverApiBaseUrl: SERVER_API_BASE_URL })
    suspendButton().click()

    expect(fetchMock).toHaveBeenCalledWith(`${SERVER_API_BASE_URL}/suspend`, { method: 'POST' })
  })
})
