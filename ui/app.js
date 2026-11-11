// metal: blackberry's control page. Everything shown comes from gateway's
// GET /status (gateway probes blackberry and keeps the last two days of
// wakes and suspends); Wake goes to gateway, Suspend to hostd.

// Lucide icon paths (ISC license), inlined so there's nothing to fetch.
const ICONS = {
  power: '<path d="M12 2v10"/><path d="M18.4 6.6a9 9 0 1 1-12.77.04"/>',
  refresh:
    '<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>',
  external: '<path d="M7 17 17 7"/><path d="M8 7h9v9"/>',
  alert: '<circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/>',
}
const icon = (name) => `<svg class="icon" viewBox="0 0 24 24" aria-hidden="true">${ICONS[name]}</svg>`

// The ingot: a metal bar with a seam that glows while blackberry is awake.
const LOGO = `<svg viewBox="0 0 64 64" aria-hidden="true"><polygon points="6,46 58,46 49,18 15,18" fill="#23262c" stroke="#c9cdd3" stroke-width="3" stroke-linejoin="round"/><line class="seam" x1="17" y1="37" x2="47" y2="37" stroke-width="4" stroke-linecap="round"/></svg>`

const POLL_MS = 12_000
// While a Wake or Suspend is settling, poll fast so the page follows it.
const FAST_POLL_MS = 2_000
const FAST_POLL_FOR_MS = 90_000
const REQUEST_TIMEOUT_MS = 8_000

const MINUTE = 60_000

const clock = (date) => date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })

function duration(ms) {
  const minutes = Math.max(0, Math.round(ms / MINUTE))
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

function ago(date, now) {
  const ms = now - date
  if (ms < MINUTE) return 'just now'
  if (ms < 60 * MINUTE) return `${Math.round(ms / MINUTE)} min ago`
  if (ms < 24 * 60 * MINUTE) return `${Math.round(ms / (60 * MINUTE))} h ago`
  return date.toLocaleDateString()
}

// A login like "nik@github" reads as "nik"; a route name stays as it is.
const who = (by) => by.split('@')[0]

const startOfToday = (now) => {
  const d = new Date(now)
  d.setHours(0, 0, 0, 0)
  return d
}

/**
 * How long blackberry has been awake since local midnight, from gateway's
 * events and its current state. Exported for tests.
 */
export function awakeToday(events, awake, since, now = new Date()) {
  const midnight = startOfToday(now)
  const today = events.map((e) => ({ at: new Date(e.at), type: e.type })).filter((e) => e.at >= midnight)
  // The state at midnight is the opposite of what the first event today made
  // it; with no events today, it's been the current one since `since`.
  let on = today.length ? today[0].type === 'suspended' : Boolean(awake)
  let from = today.length ? midnight : new Date(Math.max(midnight, new Date(since)))
  let total = 0
  for (const e of today) {
    if (on) total += e.at - from
    on = e.type === 'woke'
    from = e.at
  }
  if (on) total += now - from
  return total
}

/**
 * Renders the page into `container` and starts polling. Returns a cleanup
 * function that stops polling.
 */
export function mountUi(container, options) {
  const { hostdBaseUrl, gatewayBaseUrl, intervalMs = POLL_MS } = options

  container.innerHTML = `
    <header>
      <h1 class="brand">${LOGO}<span>metal</span></h1>
      <button id="refresh" class="icon-button" type="button" aria-label="Refresh">${icon('refresh')}</button>
    </header>

    <section class="card status" aria-labelledby="state">
      <p id="state" class="state" aria-live="polite"><span class="dot"></span><span id="state-label">Checking…</span></p>
      <p id="subtitle" class="subtitle"></p>
      <div id="countdown" class="countdown" hidden>
        <div class="countdown-label"><span>Sleeps in</span><span id="countdown-value"></span></div>
        <div class="bar"><div id="countdown-bar"></div></div>
      </div>
      <button id="primary" class="primary" type="button" disabled>${icon('power')}<span>Wake</span></button>
    </section>

    <div class="stats">
      <div class="card stat"><span>Last wake</span><span id="last-wake">—</span></div>
      <div class="card stat"><span>On today</span><span id="on-today">—</span></div>
    </div>

    <section aria-labelledby="stacks-heading">
      <h2 id="stacks-heading">Stacks</h2>
      <div id="stacks" class="list stacks"></div>
    </section>

    <section aria-labelledby="automation-heading">
      <h2 id="automation-heading">Automation</h2>
      <div class="card list">
        <label class="switch"><span>Auto-wake</span><input type="checkbox" role="switch" name="wake" disabled /></label>
        <label class="switch"><span>Auto-suspend</span><input type="checkbox" role="switch" name="suspend" disabled /></label>
      </div>
    </section>

    <section aria-labelledby="today-heading">
      <h2 id="today-heading">Today</h2>
      <ol id="events" class="card list events"></ol>
    </section>

    <p id="action-error" role="alert" hidden>${icon('alert')}<span></span></p>
    <footer>Works only from a device on the tailnet</footer>
  `
  const $ = (selector) => container.querySelector(selector)
  const stateLabel = $('#state-label')
  const subtitle = $('#subtitle')
  const countdown = $('#countdown')
  const countdownValue = $('#countdown-value')
  const countdownBar = $('#countdown-bar')
  const primary = $('#primary')
  const primaryLabel = primary.querySelector('span')
  const refresh = $('#refresh')
  const lastWake = $('#last-wake')
  const onToday = $('#on-today')
  const stacks = $('#stacks')
  const events = $('#events')
  const errorElement = $('#action-error')
  const errorText = errorElement.querySelector('span')
  const switches = container.querySelectorAll('.switch input')

  let status = null // the last GET /status, or null if gateway didn't answer
  let pending = null // 'wake' | 'suspend' while one is settling
  let fastUntil = 0
  let pollInFlight = false
  let timerId = 0

  const showError = (message) => {
    errorElement.hidden = false
    errorText.textContent = message
  }
  const clearError = () => {
    errorElement.hidden = true
  }

  const fetchWithTimeout = (url, init = {}) => {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)
    return fetch(url, { ...init, signal: controller.signal }).finally(() => clearTimeout(timeoutId))
  }

  const render = () => {
    const now = new Date()
    const awake = status?.awake
    const known = status !== null && awake !== undefined && awake !== null
    const waking = pending === 'wake' || (status?.wake_pending && !awake)
    const state = known ? (awake ? 'awake' : 'asleep') : 'unknown'
    container.dataset.state = state

    if (!known) {
      stateLabel.textContent = status === null && !pollInFlight ? 'Unknown' : 'Checking…'
      subtitle.textContent = status === null && !pollInFlight ? "Can't reach gateway on raspberry." : ''
    } else if (waking) {
      stateLabel.textContent = 'Waking…'
      subtitle.textContent = 'Usually takes under 10 seconds.'
    } else if (pending === 'suspend' && awake) {
      stateLabel.textContent = 'Suspending…'
      subtitle.textContent = ''
    } else if (awake) {
      stateLabel.textContent = 'Awake'
      const parts = [duration(now - new Date(status.since))]
      if (status.woken_by) parts.push(`woken by ${who(status.woken_by)}`)
      subtitle.textContent = parts.join(' · ')
    } else {
      stateLabel.textContent = 'Asleep'
      const parts = [`Since ${clock(new Date(status.since))}`]
      if (status.wake && status.routes.some((r) => r.wake)) parts.push('wakes on request')
      subtitle.textContent = parts.join(' · ')
    }

    // Countdown: only when gateway did the waking and will suspend it again.
    const sleepsAt = awake && status.sleeps_at ? new Date(status.sleeps_at) : null
    countdown.hidden = !sleepsAt
    if (sleepsAt) {
      const left = sleepsAt - now
      countdownValue.textContent = left > MINUTE ? duration(left) : 'any moment'
      const window = status.idle_window_seconds * 1000
      countdownBar.style.width = `${window > 0 ? Math.min(100, Math.max(0, (left / window) * 100)) : 0}%`
    } else if (awake && known && !pending) {
      subtitle.textContent += status.suspend ? ' · stays on until you suspend it' : ' · auto-suspend is off'
    }

    const showWake = !known || !awake
    primary.className = showWake ? 'primary wake' : 'primary'
    primaryLabel.textContent = waking ? 'Waking…' : pending === 'suspend' ? 'Suspending…' : showWake ? 'Wake' : 'Suspend'
    primary.disabled = pending !== null || waking || pollInFlight && !known

    lastWake.textContent = status?.last_wake_seconds ? `${status.last_wake_seconds.toFixed(1)}s` : '—'
    onToday.textContent = known ? duration(awakeToday(status.events, awake, status.since, now)) : '—'

    stacks.replaceChildren(
      ...(status?.routes ?? []).map((route) => {
        const row = document.createElement('a')
        row.className = 'card stack'
        row.href = route.url
        row.target = '_blank'
        row.rel = 'noopener'
        const meta = !awake
          ? route.wake
            ? 'sleeping · wakes on open'
            : 'sleeping'
          : route.last_request
            ? ago(new Date(route.last_request), now)
            : 'no requests yet'
        row.innerHTML = `<span class="dot"></span><span class="stack-name"></span><span class="stack-meta"></span>${icon('external')}`
        row.querySelector('.stack-name').textContent = route.name
        row.querySelector('.stack-meta').textContent = meta
        return row
      }),
    )
    if (status && status.routes.length === 0) {
      const empty = document.createElement('p')
      empty.className = 'card empty'
      empty.textContent = 'No stacks are routed through gateway.'
      stacks.replaceChildren(empty)
    }

    const midnight = startOfToday(now)
    const todays = (status?.events ?? []).filter((e) => new Date(e.at) >= midnight).reverse()
    events.replaceChildren(
      ...todays.map((e) => {
        const li = document.createElement('li')
        const time = document.createElement('time')
        time.dateTime = e.at
        time.textContent = clock(new Date(e.at))
        const label = document.createElement('span')
        label.className = e.type
        label.textContent = e.type === 'woke' ? 'Woke' : 'Suspended'
        li.append(time, label)
        return li
      }),
    )
    if (!todays.length) {
      const li = document.createElement('li')
      li.className = 'empty'
      li.textContent = known ? 'Nothing yet today' : '—'
      events.replaceChildren(li)
    }

    for (const input of switches) {
      if (status && document.activeElement !== input) {
        input.checked = status[input.name]
        input.disabled = false
      }
    }
    refresh.disabled = pollInFlight
    refresh.setAttribute('aria-busy', String(pollInFlight))
  }

  const schedule = () => {
    clearTimeout(timerId)
    timerId = setTimeout(poll, Date.now() < fastUntil ? FAST_POLL_MS : intervalMs)
  }

  const poll = async () => {
    pollInFlight = true
    render()
    try {
      const response = await fetchWithTimeout(`${gatewayBaseUrl}/status`)
      status = response.ok ? await response.json() : null
    } catch {
      status = null
    }
    pollInFlight = false
    // A pending action is done once the state it asked for shows up.
    if ((pending === 'wake' && status?.awake) || (pending === 'suspend' && status?.awake === false) || Date.now() > fastUntil) {
      pending = null
    }
    render()
    schedule()
  }

  const act = async (kind, request) => {
    clearError()
    pending = kind
    fastUntil = Date.now() + FAST_POLL_FOR_MS
    render()
    const label = kind === 'wake' ? 'Wake' : 'Suspend'
    try {
      const response = await request()
      if (!response.ok) {
        pending = null
        showError(`${label} failed (HTTP ${response.status}).`)
      }
    } catch {
      pending = null
      showError(`${label} failed: could not reach the server.`)
    }
    void poll()
  }

  primary.addEventListener('click', () => {
    if (status?.awake) {
      // Suspend lives on hostd, Wake on gateway (blackberry can't wake itself).
      void act('suspend', () => fetchWithTimeout(`${hostdBaseUrl}/suspend`, { method: 'POST' }))
    } else {
      void act('wake', () => fetchWithTimeout(`${gatewayBaseUrl}/wake`, { method: 'POST' }))
    }
  })

  // gateway's switches: POST sets one, form-encoded so the browser sends no
  // extra CORS preflight headers.
  for (const input of switches) {
    input.addEventListener('change', async () => {
      clearError()
      input.disabled = true
      try {
        const response = await fetchWithTimeout(`${gatewayBaseUrl}/settings`, {
          method: 'POST',
          body: new URLSearchParams({ [input.name]: input.checked }),
        })
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        const settings = await response.json()
        if (status) Object.assign(status, settings)
      } catch (error) {
        input.checked = !input.checked
        showError(`Changing the switch failed: ${error.message}.`)
      }
      input.disabled = false
      render()
    })
  }

  refresh.addEventListener('click', () => void poll())

  void poll()
  // The countdown and "ago" labels tick between polls.
  const tickId = setInterval(render, 30_000)

  return () => {
    clearTimeout(timerId)
    clearInterval(tickId)
  }
}
