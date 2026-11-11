// node --test ui/app.test.mjs: the "On today" sum.
import test from 'node:test'
import assert from 'node:assert/strict'
import { awakeToday } from './app.js'

const at = (h, m = 0) => {
  const d = new Date()
  d.setHours(h, m, 0, 0)
  return d
}
const H = 3_600_000

test('awake intervals today add up', () => {
  const events = [
    { at: at(10).toISOString(), type: 'woke' },
    { at: at(11).toISOString(), type: 'suspended' },
    { at: at(14).toISOString(), type: 'woke' },
  ]
  assert.equal(awakeToday(events, true, at(14).toISOString(), at(15)), 2 * H)
})

test('awake since before midnight counts from midnight', () => {
  const events = [{ at: at(2).toISOString(), type: 'suspended' }]
  assert.equal(awakeToday(events, false, at(2).toISOString(), at(9)), 2 * H)
})

test('no events today: from since, or nothing if asleep', () => {
  assert.equal(awakeToday([], true, at(8).toISOString(), at(9)), H)
  assert.equal(awakeToday([], false, at(8).toISOString(), at(9)), 0)
})
