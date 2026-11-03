import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createSearchClient, type SearchWorker } from './client'
import type { Query, Reply } from './engine'

/** A worker that does nothing by itself: the test sends its replies, in the order it likes. */
class FakeWorker implements SearchWorker {
  onmessage: ((event: MessageEvent<Reply>) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  sent: Query[] = []
  terminated = false

  postMessage(message: Query): void {
    this.sent.push(message)
  }

  terminate(): void {
    this.terminated = true
  }

  reply(reply: Reply): void {
    this.onmessage?.({ data: reply } as MessageEvent<Reply>)
  }

  fail(): void {
    this.onerror?.(new Event('error'))
  }
}

const hit = { file: 0, ptr: '/a', text: 'x', ranges: [[0, 1]] as [number, number][] }

describe('createSearchClient', () => {
  let worker: FakeWorker
  let make: ReturnType<typeof vi.fn<() => SearchWorker>>

  beforeEach(() => {
    worker = new FakeWorker()
    make = vi.fn(() => worker)
  })

  it('does not start the worker until the first search', () => {
    createSearchClient(make)
    expect(make).not.toHaveBeenCalled()
  })

  it('sends the query with an id and the base of the data, and resolves with the reply', async () => {
    const client = createSearchClient(make)
    const result = client.search('警告', 'zh', false)
    expect(worker.sent).toHaveLength(1)
    const sent = worker.sent[0] as Query
    expect(sent).toMatchObject({ q: '警告', lang: 'zh', re: false })
    expect(typeof sent.id).toBe('number')
    expect(new URL(sent.base).pathname).toMatch(/\/data\/$/)
    expect(sent.base).toBe(new URL('data/', document.baseURI).href)
    worker.reply({ id: sent.id, hits: [hit], total: 1 })
    expect(await result).toEqual({ hits: [hit], total: 1 })
  })

  it('resolves with the error of the worker', async () => {
    const client = createSearchClient(make)
    const result = client.search('(', 'zh', true)
    worker.reply({ id: (worker.sent[0] as Query).id, error: '正規表示式語法錯誤' })
    expect(await result).toEqual({ error: '正規表示式語法錯誤' })
  })

  it('starts one worker for all the searches', async () => {
    const client = createSearchClient(make)
    const first = client.search('a', 'zh', false)
    worker.reply({ id: (worker.sent[0] as Query).id, hits: [], total: 0 })
    await first
    void client.search('b', 'zh', false)
    expect(make).toHaveBeenCalledTimes(1)
    expect(worker.sent).toHaveLength(2)
  })

  it('newer query supersedes older', async () => {
    const client = createSearchClient(make)
    const first = client.search('a', 'zh', false)
    const second = client.search('ab', 'zh', false)
    const [one, two] = worker.sent as [Query, Query]
    expect(two.id).not.toBe(one.id)
    // the first is over as soon as the second is asked
    expect(await first).toEqual({ superseded: true })
    // the old reply comes late, after the new one, and does not take the place of it
    worker.reply({ id: two.id, hits: [hit], total: 1 })
    worker.reply({ id: one.id, hits: [], total: 0 })
    expect(await second).toEqual({ hits: [hit], total: 1 })
  })

  it('ignores the reply of the older query that comes before the reply of the newer', async () => {
    const client = createSearchClient(make)
    const first = client.search('a', 'zh', false)
    const second = client.search('ab', 'zh', false)
    const [one, two] = worker.sent as [Query, Query]
    worker.reply({ id: one.id, hits: [], total: 0 })
    let settled = false
    void second.then(() => (settled = true))
    await Promise.resolve()
    expect(settled).toBe(false)
    worker.reply({ id: two.id, hits: [hit], total: 1 })
    expect(await first).toEqual({ superseded: true })
    expect(await second).toEqual({ hits: [hit], total: 1 })
  })

  it('ignores a reply to no query, and a second reply to one', async () => {
    const client = createSearchClient(make)
    const result = client.search('a', 'zh', false)
    const { id } = worker.sent[0] as Query
    worker.reply({ id: id + 99, hits: [], total: 5 })
    worker.reply({ id, hits: [hit], total: 1 })
    worker.reply({ id, hits: [], total: 7 })
    expect(await result).toEqual({ hits: [hit], total: 1 })
  })

  it('knows which languages the worker holds: those it has answered a search for, even a late answer', async () => {
    const client = createSearchClient(make)
    expect(client.has('kr')).toBe(false)
    const first = client.search('a', 'kr', false)
    const second = client.search('b', 'jp', false)
    const [one, two] = worker.sent as [Query, Query]
    worker.reply({ id: two.id, hits: [], total: 0 })
    await second
    expect(client.has('jp')).toBe(true)
    expect(client.has('kr')).toBe(false)
    worker.reply({ id: one.id, hits: [], total: 0 })
    await first
    expect(client.has('kr')).toBe(true)
    expect(client.has('en')).toBe(false)
  })

  it('does not count an answer with an error as a language that came in', async () => {
    const client = createSearchClient(make)
    const result = client.search('(', 'kr', true)
    worker.reply({ id: (worker.sent[0] as Query).id, error: 'x' })
    await result
    expect(client.has('kr')).toBe(false)
  })

  it('resolves with an error when the worker fails, and starts a new worker for the next search', async () => {
    const client = createSearchClient(make)
    const result = client.search('a', 'zh', false)
    worker.fail()
    expect(await result).toEqual({ error: expect.any(String) })
    expect(worker.terminated).toBe(true)
    const next = new FakeWorker()
    make.mockReturnValue(next)
    void client.search('a', 'zh', false)
    expect(make).toHaveBeenCalledTimes(2)
    expect(next.sent).toHaveLength(1)
  })

  it('resolves with an error when no worker can be made', async () => {
    const client = createSearchClient(() => {
      throw new Error('no Worker here')
    })
    expect(await client.search('a', 'zh', false)).toEqual({ error: expect.any(String) })
  })
})
