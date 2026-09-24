import assert from 'node:assert/strict'
import test from 'node:test'

import {
  CLOCK_TOLERANCE_MS,
  OPERATIONS,
  TuCapi,
  PaymentsError,
  SignatureError,
  TransportError,
  parseEvent,
  sign,
  verifyEvent,
  type Transaction,
} from './index.js'

// ─── the shared vectors (same as Go and Python) ──────────────────────────────

const SECRET = 'whsec_test_0123456789'
const EVENT_BODY =
  '{"id":"0c9d1b2e-7f3a-4b8c-9d0e-1f2a3b4c5d6e","type":"payout.confirmed","occurred_at":"2026-09-23T15:04:05Z","data":{"id":"6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f","type":"payout","country":"VE","currency":"VES","method":"mobile_payment","amount":"1500.50","status":"confirmed","bank_reference":"000123456789","failure":null,"created_at":"2026-09-23T14:59:05Z","updated_at":"2026-09-23T15:04:05Z"}}'
const FIXED_TS = 1790000000
const FIXED_SIG = 'v1=f62b5544c9b42eb7d94c053ab1d6bda34febf84da6a205dc2df998a7d87d0382'

const TX_JSON =
  '{"id":"6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f","type":"payout","country":"VE","currency":"VES","method":"mobile_payment","amount":"1500.50","status":"pending","pending_reason":"processing","bank_reference":null,"failure":null,"created_at":"2026-09-23T14:59:05Z","updated_at":"2026-09-23T15:04:05Z"}'

// ─── a scripted fetch ────────────────────────────────────────────────────────

interface Seen {
  method: string
  url: string
  key: string | undefined
  body: string | undefined
}
type Answer = { status: number; body: string } | 'drop'

function fakeFetch(script: Record<string, Answer[]>) {
  const seen: Seen[] = []
  const fetch = async (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
    const url = String(input)
    const headers = Object.fromEntries(Object.entries(init?.headers as Record<string, string>).map(([k, v]) => [k.toLowerCase(), v]))
    seen.push({ method: init?.method ?? 'GET', url, key: headers['idempotency-key'], body: typeof init?.body === 'string' ? init.body : undefined })
    const path = new URL(url).pathname
    const answers = script[`${init?.method ?? 'GET'} ${path}`] ?? []
    const a = answers.length > 1 ? answers.shift()! : (answers[0] ?? { status: 200, body: '{}' })
    if (a === 'drop') throw new TypeError('fetch failed')
    return new Response(a.body, { status: a.status, headers: { 'content-type': 'application/json' } })
  }
  return { fetch: fetch as typeof globalThis.fetch, seen }
}

function client(f: ReturnType<typeof fakeFetch>, extra: ConstructorParameters<typeof TuCapi>[1] = {}) {
  return new TuCapi('tuc_test_x', { baseUrl: 'https://api.test', fetch: f.fetch, sleep: async () => {}, ...extra })
}

const payout = () => ({
  country: 'VE',
  currency: 'VES',
  method: 'mobile_payment' as const,
  amount: '1500.50',
  beneficiary: { name: 'Ana', document: { type: 'V', number: '12345678' }, bank_code: '0102', account_number: '04121234567' },
})

// ─── idempotency ─────────────────────────────────────────────────────────────

test('a create sends the key you give and returns it', async () => {
  const f = fakeFetch({ 'POST /v2/payouts': [{ status: 201, body: TX_JSON }] })
  const created = await client(f).payouts.create(payout(), { idempotencyKey: 'order-4821' })
  assert.equal(created.idempotencyKey, 'order-4821')
  assert.equal(created.status, 'pending')
  assert.equal(f.seen[0]?.key, 'order-4821')
  assert.match(f.seen[0]?.body ?? '', /"method":"mobile_payment"/)
})

test('a create without key generates one and reuses it on retry', async () => {
  const f = fakeFetch({ 'POST /v2/payins': ['drop', { status: 503, body: '{"error":{"code":"temporarily_unavailable","message":"x"}}' }, { status: 201, body: TX_JSON }] })
  const created = await client(f).payins.create({ country: 'VE', currency: 'VES', method: 'debit_otp', amount: '10.00', payer: { name: 'Ana' } })
  assert.equal(created.idempotencyKey.length, 32)
  assert.equal(f.seen.length, 3)
  for (const s of f.seen) assert.equal(s.key, created.idempotencyKey)
})

test('a create that never gets an answer returns the key in the error', async () => {
  const f = fakeFetch({ 'POST /v2/payouts': ['drop', 'drop', 'drop'] })
  await assert.rejects(client(f).payouts.create(payout(), { idempotencyKey: 'order-1' }), (err: unknown) => {
    assert.ok(err instanceof TransportError)
    assert.equal(err.attempts, 3)
    assert.equal(err.idempotencyKey, 'order-1')
    return true
  })
  assert.equal(f.seen.length, 3)
})

// ─── retries: only when nothing can move twice ───────────────────────────────

test('a GET is retried on 5xx and 429', async () => {
  const f = fakeFetch({ 'GET /v2/transactions/t1': [{ status: 502, body: 'bad gateway' }, { status: 429, body: '{"error":{"code":"x","message":"slow"}}' }, { status: 200, body: TX_JSON }] })
  const tx = await client(f).transactions.get('t1')
  assert.equal(tx.id, '6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f')
  assert.equal(f.seen.length, 3)
})

test('a GET gives up after maxAttempts', async () => {
  const f = fakeFetch({ 'GET /v2/balances': [{ status: 500, body: '{"error":{"code":"internal","message":"x"}}' }, { status: 500, body: '{"error":{"code":"internal","message":"x"}}' }] })
  await assert.rejects(client(f, { maxAttempts: 2 }).balances.list(), (err: unknown) => err instanceof PaymentsError && err.status === 500)
  assert.equal(f.seen.length, 2)
})

test('confirm, resend, cancel and webhook endpoints are never retried', async () => {
  const cases: Array<[string, (c: TuCapi) => Promise<unknown>]> = [
    ['POST /v2/payins/p1/confirm', (c) => c.payins.confirm('p1', '123456')],
    ['POST /v2/payins/p1/resend-code', (c) => c.payins.resendCode('p1')],
    ['POST /v2/payouts/p1/cancel', (c) => c.payouts.cancel('p1')],
    ['POST /v2/webhook-endpoints', (c) => c.webhookEndpoints.create('https://x')],
  ]
  for (const [route, call] of cases) {
    const f = fakeFetch({ [route]: [{ status: 503, body: '{"error":{"code":"temporarily_unavailable","message":"x"}}' }, { status: 200, body: TX_JSON }] })
    await assert.rejects(call(client(f)), (err: unknown) => err instanceof PaymentsError && err.code === 'temporarily_unavailable')
    assert.equal(f.seen.length, 1, `${route} was retried`)
    assert.equal(f.seen[0]?.key, undefined, `${route} sent a key it does not have`)
  }
})

// ─── errors and shapes ───────────────────────────────────────────────────────

test('an API error is typed with its details', async () => {
  const f = fakeFetch({ 'POST /v2/payouts': [{ status: 400, body: '{"error":{"code":"field_invalid","message":"Hay campos con un valor inválido.","details":[{"field":"beneficiary.bank_code","code":"invalid"}]}}' }] })
  await assert.rejects(client(f).payouts.create(payout(), { idempotencyKey: 'k' }), (err: unknown) => {
    assert.ok(err instanceof PaymentsError)
    assert.equal(err.code, 'field_invalid')
    assert.equal(err.status, 400)
    assert.deepEqual(err.details, [{ field: 'beneficiary.bank_code', code: 'invalid' }])
    assert.match(err.message, /beneficiary\.bank_code: invalid/)
    return true
  })
})

test('an unreadable error still has the status', async () => {
  const f = fakeFetch({ 'GET /v2/payins/p1': [{ status: 404, body: '<html>' }] })
  await assert.rejects(client(f).payins.get('p1'), (err: unknown) => err instanceof PaymentsError && err.status === 404 && err.code === 'unreadable_response')
})

test('ids are escaped and queries are built from the arguments', async () => {
  const f = fakeFetch({})
  const c = client(f)
  await c.payouts.get('p1/cancel')
  await c.banks.list('VE', 'debit_otp')
  await c.methods.list({ direction: 'payin' })
  await c.events.list('c-9', 25)
  assert.deepEqual(
    f.seen.map((s) => s.url),
    ['https://api.test/v2/payouts/p1%2Fcancel', 'https://api.test/v2/banks?country=VE&method=debit_otp', 'https://api.test/v2/methods?direction=payin', 'https://api.test/v2/events?cursor=c-9&limit=25'],
  )
})

test('waitUntilFinal polls until the outcome', async () => {
  const final = TX_JSON.replace('"status":"pending","pending_reason":"processing","bank_reference":null', '"status":"confirmed","bank_reference":"000123"')
  const f = fakeFetch({ 'GET /v2/transactions/t1': [{ status: 200, body: TX_JSON }, { status: 200, body: TX_JSON }, { status: 200, body: final }] })
  const tx: Transaction = await client(f).transactions.waitUntilFinal('t1', 1)
  assert.equal(tx.status, 'confirmed')
  assert.equal(tx.bank_reference, '000123')
  assert.equal(f.seen.length, 3)
})

test('the generated table says which operations carry a key', () => {
  assert.ok(OPERATIONS['payins.create']?.idempotent && OPERATIONS['payouts.create']?.idempotent)
  for (const op of ['payins.confirm', 'payouts.cancel', 'payins.resendCode', 'webhookEndpoints.create']) assert.equal(OPERATIONS[op]?.idempotent, false)
})

// ─── webhooks ────────────────────────────────────────────────────────────────

test('the shared vector verifies', () => {
  const ev = verifyEvent(Buffer.from(EVENT_BODY), FIXED_SIG, String(FIXED_TS), SECRET, FIXED_TS * 1000 + 60_000)
  assert.equal(ev.type, 'payout.confirmed')
  assert.equal(ev.data.status, 'confirmed')
  assert.equal(sign(Buffer.from(EVENT_BODY), FIXED_TS, SECRET), FIXED_SIG)
})

test('parseEvent takes the headers of a request', () => {
  const now = Math.floor(Date.now() / 1000)
  const body = Buffer.from(EVENT_BODY)
  const ev = parseEvent(body, { 'x-firma': sign(body, now, SECRET), 'x-timestamp': String(now) }, SECRET)
  assert.equal(ev.id, '0c9d1b2e-7f3a-4b8c-9d0e-1f2a3b4c5d6e')
  const h = new Headers({ 'X-Firma': sign(body, now, SECRET), 'X-Timestamp': String(now) })
  assert.equal(parseEvent(body, h, SECRET).id, ev.id)
})

test('a tampered body, another secret, an old event and broken headers are rejected', () => {
  const now = Math.floor(Date.now() / 1000)
  const body = Buffer.from(EVENT_BODY)
  const good = sign(body, now, SECRET)
  assert.throws(() => verifyEvent(Buffer.from(EVENT_BODY.replace('1500.50', '9500.50')), good, String(now), SECRET), SignatureError)
  assert.throws(() => verifyEvent(body, sign(body, now, 'other'), String(now), SECRET), SignatureError)
  const old = now - CLOCK_TOLERANCE_MS / 1000 - 60
  assert.throws(() => verifyEvent(body, sign(body, old, SECRET), String(old), SECRET), SignatureError)
  assert.throws(() => verifyEvent(body, undefined, String(now), SECRET), SignatureError)
  assert.throws(() => verifyEvent(body, good, undefined, SECRET), SignatureError)
  assert.throws(() => verifyEvent(body, good, 'ayer', SECRET), SignatureError)
  assert.throws(() => verifyEvent(body, 'v1=zz', String(now), SECRET), SignatureError)
})

test('during a rotation either signature is enough', () => {
  const now = Math.floor(Date.now() / 1000)
  const body = Buffer.from(EVENT_BODY)
  const header = `${sign(body, now, 'new-secret')} ${sign(body, now, SECRET)}`
  assert.equal(verifyEvent(body, header, String(now), SECRET).type, 'payout.confirmed')
  assert.equal(verifyEvent(body, header, String(now), 'new-secret').type, 'payout.confirmed')
  assert.throws(() => verifyEvent(body, header, String(now), 'neither'), SignatureError)
})
