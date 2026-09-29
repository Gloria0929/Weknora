import assert from 'node:assert/strict'
import test from 'node:test'
import { readStoredPtyId, sandboxPtyStorageKey, writeStoredPtyId } from './sandboxPtyId'

const memory = new Map<string, string>()

function installFakeStorage() {
  const fake = {
    getItem(key: string) {
      return memory.has(key) ? memory.get(key)! : null
    },
    setItem(key: string, value: string) {
      memory.set(key, value)
    },
    removeItem(key: string) {
      memory.delete(key)
    },
  }
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: fake,
  })
}

test('round-trips a pid keyed by session', () => {
  memory.clear()
  installFakeStorage()
  writeStoredPtyId('sess-a', 4321)
  assert.equal(readStoredPtyId('sess-a'), 4321)
  assert.equal(readStoredPtyId('sess-b'), null)
  assert.equal(sandboxPtyStorageKey('sess-a'), 'weknora_sandbox_pty:sess-a')
})

test('forgets the pid when cleared or invalid', () => {
  memory.clear()
  installFakeStorage()
  writeStoredPtyId('sess-a', 4321)
  writeStoredPtyId('sess-a', null)
  assert.equal(readStoredPtyId('sess-a'), null)

  writeStoredPtyId('sess-a', 7)
  memory.set(sandboxPtyStorageKey('sess-a'), 'nope')
  assert.equal(readStoredPtyId('sess-a'), null)
})

test('keeps independent pids for terminal slots in the same session', () => {
  memory.clear()
  installFakeStorage()
  writeStoredPtyId('sess-a', 101, 'programming-1')
  writeStoredPtyId('sess-a', 202, 'programming-2')

  assert.equal(readStoredPtyId('sess-a', 'programming-1'), 101)
  assert.equal(readStoredPtyId('sess-a', 'programming-2'), 202)
  assert.equal(readStoredPtyId('sess-a'), null)
  assert.equal(
    sandboxPtyStorageKey('sess-a', 'programming-2'),
    'weknora_sandbox_pty:sess-a:programming-2',
  )
})
