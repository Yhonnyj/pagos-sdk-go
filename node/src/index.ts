/**
 * TuCapi — Node SDK for the payments API, version 2.
 *
 * ═══════════════════════════════════════════════════════════════════════════
 * WHAT IT DOES FOR YOU, AND WHY IT LIVES HERE AND NOT IN YOUR CODE.
 *
 *   1. Authentication with your API key.
 *   2. The contract, typed (contract.gen.ts is generated from the OpenAPI).
 *   3. Idempotency and retries that are SAFE: every create carries an
 *      Idempotency-Key (yours, or one the SDK generates and hands back), and
 *      the SDK only retries a request that cannot move money twice.
 *   4. WEBHOOK SIGNATURE VERIFICATION: constant time, with the clock window,
 *      over the RAW body (`JSON.parse` + `JSON.stringify` changes the bytes).
 *
 * ═══════════════════════════════════════════════════════════════════════════
 * EXAMPLE
 *
 *   const c = new TuCapi('ck_live_...')
 *   const created = await c.payouts.create({ country: 'VE', currency: 'VES', method: 'mobile_payment',
 *     amount: '1500.50', beneficiary: { name: 'Ana', document: { type: 'V', number: '12345678' },
 *     bank_code: '0102', account_number: '04121234567' } }, { idempotencyKey: 'order-4821' })
 *   // created.idempotencyKey is the key that was sent: keep it with your order.
 *
 *   const tx = await c.transactions.waitUntilFinal(created.id)
 *
 * And in your webhook receiver (Express: `express.raw({ type: 'application/json' })`):
 *
 *   const ev = parseEvent(req.body, req.headers, mySecret)
 */

import { createHmac, randomBytes, timingSafeEqual } from 'node:crypto'

export * from './contract.gen.js'
import {
  CLOCK_TOLERANCE_MS,
  DEFAULT_BASE_URL,
  HEADER_IDEMPOTENCY_KEY,
  HEADER_SIGNATURE,
  HEADER_TIMESTAMP,
  OPERATIONS,
  ROUTE_BALANCES_LIST,
  ROUTE_BANKS_LIST,
  ROUTE_CAPABILITIES_GET,
  ROUTE_EVENTS_LIST,
  ROUTE_METHODS_LIST,
  ROUTE_PAYINS_CONFIRM,
  ROUTE_PAYINS_CREATE,
  ROUTE_PAYINS_GET,
  ROUTE_PAYINS_RESEND_CODE,
  ROUTE_PAYOUTS_CANCEL,
  ROUTE_PAYOUTS_CREATE,
  ROUTE_PAYOUTS_GET,
  ROUTE_TRANSACTIONS_GET,
  ROUTE_WEBHOOK_ENDPOINTS_CREATE,
  SIGNATURE_VERSION,
} from './contract.gen.js'
import type {
  BalanceList,
  BankList,
  Capabilities,
  Direction,
  ErrorCode,
  ErrorEnvelope,
  Event,
  EventList,
  FieldError,
  MethodCode,
  MethodList,
  NewPayin,
  NewPayout,
  Transaction,
  WebhookEndpoint,
} from './contract.gen.js'

/** The version of this SDK, sent in User-Agent. */
export const VERSION = '0.1.0'

/** How long a single request waits. Creating a payment talks to a bank. */
export const DEFAULT_TIMEOUT_MS = 30_000

/** How many times a retryable request is tried in total. */
export const DEFAULT_MAX_ATTEMPTS = 3

/** An error answered by the API, with its stable code. Compare `code`, never `message`. */
export class PaymentsError extends Error {
  readonly status: number
  readonly code: ErrorCode | 'unreadable_response'
  readonly details: FieldError[]

  constructor(status: number, code: ErrorCode | 'unreadable_response', message: string, details: FieldError[] = []) {
    super(details.length ? `${message} (${code}): ${details.map((d) => `${d.field}: ${d.code}`).join('; ')}` : `${message} (${code})`)
    this.name = 'PaymentsError'
    this.status = status
    this.code = code
    this.details = details
  }
}

/**
 * No HTTP response was obtained at all, after every attempt.
 *
 * ⚠️ IF THIS COMES FROM A CREATE, THE OPERATION MAY EXIST ANYWAY. Do not send
 * it again with a new key: repeat the SAME request with `idempotencyKey`.
 */
export class TransportError extends Error {
  readonly attempts: number
  readonly idempotencyKey: string | undefined
  constructor(cause: unknown, attempts: number, idempotencyKey?: string) {
    super(`payments: no response after ${attempts} attempt(s): ${cause instanceof Error ? cause.message : String(cause)}`)
    this.name = 'TransportError'
    this.attempts = attempts
    this.idempotencyKey = idempotencyKey
  }
}

/** The webhook signature did not verify. Discard the event: do not process it. */
export class SignatureError extends Error {
  constructor(reason: string) {
    super(`payments: the webhook signature is not valid: ${reason}`)
    this.name = 'SignatureError'
  }
}

export interface ClientOptions {
  /** Another server: tests, or your own environment. */
  baseUrl?: string
  /** Per-request timeout, in milliseconds. */
  timeoutMs?: number
  /** How many times a retryable request is tried in total (1 = never retry). */
  maxAttempts?: number
  /** Replaces fetch. For tests. */
  fetch?: typeof globalThis.fetch
  /** Replaces the wait between attempts. For tests. */
  sleep?: (ms: number) => Promise<void>
  /** Replaces how a key is generated when you do not pass one. */
  newIdempotencyKey?: () => string
}

export interface RequestOptions {
  /** Derive it from YOUR order id, so a retry from a fresh process sends the same key. */
  idempotencyKey?: string
}

/** A transaction just created, with the Idempotency-Key that was sent. */
export type Created = Transaction & { idempotencyKey: string }

/** The default key generator: 32 hex characters. */
export function newIdempotencyKey(): string {
  return randomBytes(16).toString('hex')
}

/** 250 ms, 1 s, 4 s… with ±25 % jitter, capped at 10 s. */
function defaultBackoff(attempt: number): number {
  let ms = 250 * 4 ** (attempt - 1)
  if (ms > 10_000) ms = 10_000
  return ms + (Math.random() - 0.5) * ms * 0.5
}

const sleepFor = (ms: number): Promise<void> => new Promise((r) => setTimeout(r, ms))

export class TuCapi {
  readonly #apiKey: string
  readonly #baseUrl: string
  readonly #timeoutMs: number
  readonly #maxAttempts: number
  readonly #fetch: typeof globalThis.fetch
  readonly #sleep: (ms: number) => Promise<void>
  readonly #newKey: () => string

  readonly capabilities: CapabilitiesService
  readonly methods: MethodsService
  readonly banks: BanksService
  readonly balances: BalancesService
  readonly payins: PayinsService
  readonly payouts: PayoutsService
  readonly transactions: TransactionsService
  readonly events: EventsService
  readonly webhookEndpoints: WebhookEndpointsService

  constructor(apiKey: string, options: ClientOptions = {}) {
    this.#apiKey = apiKey.trim()
    this.#baseUrl = (options.baseUrl ?? DEFAULT_BASE_URL).replace(/\/+$/, '')
    this.#timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS
    this.#maxAttempts = Math.max(1, options.maxAttempts ?? DEFAULT_MAX_ATTEMPTS)
    this.#fetch = options.fetch ?? globalThis.fetch
    this.#sleep = options.sleep ?? sleepFor
    this.#newKey = options.newIdempotencyKey ?? newIdempotencyKey
    this.capabilities = new CapabilitiesService(this)
    this.methods = new MethodsService(this)
    this.banks = new BanksService(this)
    this.balances = new BalancesService(this)
    this.payins = new PayinsService(this)
    this.payouts = new PayoutsService(this)
    this.transactions = new TransactionsService(this)
    this.events = new EventsService(this)
    this.webhookEndpoints = new WebhookEndpointsService(this)
  }

  /** @internal */
  async create<T>(op: string, route: string, body: unknown, options: RequestOptions): Promise<T & { idempotencyKey: string }> {
    const key = options.idempotencyKey?.trim() || this.#newKey()
    const out = await this.call<T>(op, route, undefined, body, key)
    return { ...out, idempotencyKey: key }
  }

  /** @internal */
  async sleepBetween(ms: number): Promise<void> {
    await this.#sleep(ms)
  }

  /**
   * Performs one operation with the retry policy of the contract: a GET is
   * retried on a network error, a 429 or a 5xx; a POST with Idempotency-Key
   * is retried the same way WITH THE SAME KEY; any other POST (confirm,
   * resend-code, cancel, webhook-endpoints) is never retried automatically.
   * @internal
   */
  async call<T>(op: string, route: string, query?: Record<string, string | undefined>, body?: unknown, idempotencyKey?: string): Promise<T> {
    const spec = OPERATIONS[op]
    if (!spec) throw new Error(`payments: unknown operation ${op}`)
    if (spec.idempotent && !idempotencyKey) throw new Error('payments: this operation needs an Idempotency-Key')
    const retryable = spec.method === 'GET' || (spec.idempotent && !!idempotencyKey)
    const attempts = retryable ? this.#maxAttempts : 1

    let url = this.#baseUrl + route
    if (query) {
      const q = new URLSearchParams()
      for (const [k, v] of Object.entries(query)) if (v !== undefined && v !== '') q.set(k, v)
      const s = q.toString()
      if (s) url += '?' + s
    }
    const headers: Record<string, string> = {
      Authorization: `Bearer ${this.#apiKey}`,
      Accept: 'application/json',
      'User-Agent': `payments-sdk-node/${VERSION}`,
    }
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    if (idempotencyKey) headers[HEADER_IDEMPOTENCY_KEY] = idempotencyKey

    let lastError: unknown
    for (let attempt = 1; attempt <= attempts; attempt++) {
      if (attempt > 1) await this.#sleep(defaultBackoff(attempt - 1))
      const init: RequestInit = { method: spec.method, headers, signal: AbortSignal.timeout(this.#timeoutMs) }
      if (body !== undefined) init.body = JSON.stringify(body)
      let res: Response
      try {
        res = await this.#fetch(url, init)
      } catch (err) {
        lastError = err
        continue
      }
      const text = await res.text()
      if (res.status === 429 || res.status >= 500) {
        const apiError = decodeError(res.status, text)
        if (retryable && attempt < attempts) {
          lastError = apiError
          continue
        }
        throw apiError
      }
      if (!res.ok) throw decodeError(res.status, text)
      return (text ? JSON.parse(text) : {}) as T
    }
    if (lastError instanceof PaymentsError) throw lastError
    throw new TransportError(lastError, attempts, idempotencyKey)
  }
}

function decodeError(status: number, text: string): PaymentsError {
  try {
    const env = JSON.parse(text) as Partial<ErrorEnvelope>
    if (env.error?.code) return new PaymentsError(status, env.error.code, env.error.message ?? `HTTP ${status}`, env.error.details ?? [])
  } catch {
    // falls through
  }
  return new PaymentsError(status, 'unreadable_response', `HTTP ${status}`)
}

const withId = (route: string, id: string): string => route.replace('{id}', encodeURIComponent(id))

/** What your key can do today. */
export class CapabilitiesService {
  constructor(private readonly c: TuCapi) {}
  get(): Promise<Capabilities> {
    return this.c.call('capabilities.get', ROUTE_CAPABILITIES_GET)
  }
}

/** The catalogue of methods. */
export class MethodsService {
  constructor(private readonly c: TuCapi) {}
  list(filter: { country?: string; direction?: Direction } = {}): Promise<MethodList> {
    return this.c.call('methods.list', ROUTE_METHODS_LIST, { country: filter.country, direction: filter.direction })
  }
}

/** The banks of a country. */
export class BanksService {
  constructor(private readonly c: TuCapi) {}
  list(country: string, method?: MethodCode): Promise<BankList> {
    return this.c.call('banks.list', ROUTE_BANKS_LIST, { country, method })
  }
}

/** Your balances by currency. */
export class BalancesService {
  constructor(private readonly c: TuCapi) {}
  list(): Promise<BalanceList> {
    return this.c.call('balances.list', ROUTE_BALANCES_LIST)
  }
}

/** Charging a person. */
export class PayinsService {
  constructor(private readonly c: TuCapi) {}
  /** Creates a payin. Safe to retry: the same key returns the SAME operation and asks for no new code. */
  create(payin: NewPayin, options: RequestOptions = {}): Promise<Created> {
    return this.c.create<Transaction>('payins.create', ROUTE_PAYINS_CREATE, payin, options)
  }
  get(id: string): Promise<Transaction> {
    return this.c.call('payins.get', withId(ROUTE_PAYINS_GET, id))
  }
  /**
   * Executes the debit with the code the payer typed. ⚠️ THE CODE HAS ONE TRY;
   * a wrong one fails the operation. Never retried automatically: a lost
   * answer means `get`, not `confirm` again.
   */
  confirm(id: string, code: string): Promise<Transaction> {
    return this.c.call('payins.confirm', withId(ROUTE_PAYINS_CONFIRM, id), undefined, { code })
  }
  /** Asks the payer's bank for a NEW code; the previous one stops working. */
  resendCode(id: string): Promise<Transaction> {
    return this.c.call('payins.resendCode', withId(ROUTE_PAYINS_RESEND_CODE, id), undefined, null)
  }
}

/** Paying a person. */
export class PayoutsService {
  constructor(private readonly c: TuCapi) {}
  /** Creates and sends a payout. Safe to retry with the same key. */
  create(payout: NewPayout, options: RequestOptions = {}): Promise<Created> {
    return this.c.create<Transaction>('payouts.create', ROUTE_PAYOUTS_CREATE, payout, options)
  }
  get(id: string): Promise<Transaction> {
    return this.c.call('payouts.get', withId(ROUTE_PAYOUTS_GET, id))
  }
  /**
   * Cancels a payout still pending. `not_cancellable` if it no longer can be;
   * `temporarily_unavailable` if the provider did not answer clearly — the
   * payout is still pending then, and this is NOT retried: check with `get`.
   */
  cancel(id: string): Promise<Transaction> {
    return this.c.call('payouts.cancel', withId(ROUTE_PAYOUTS_CANCEL, id), undefined, null)
  }
}

/** Any operation by id, payin or payout. */
export class TransactionsService {
  constructor(private readonly c: TuCapi) {}
  get(id: string): Promise<Transaction> {
    return this.c.call('transactions.get', withId(ROUTE_TRANSACTIONS_GET, id))
  }
  /** Polls `get` until confirmed or failed. Webhooks are the right way; this is for scripts. */
  async waitUntilFinal(id: string, everyMs = 2_000, maxPolls = 150): Promise<Transaction> {
    let tx = await this.get(id)
    for (let i = 1; tx.status === 'pending' && i < maxPolls; i++) {
      await this.c.sleepBetween(everyMs)
      tx = await this.get(id)
    }
    return tx
  }
}

/** What happened to your operations, by query. */
export class EventsService {
  constructor(private readonly c: TuCapi) {}
  /** Events in order. `cursor` = `next_cursor` of the previous page. */
  list(cursor?: string, limit?: number): Promise<EventList> {
    return this.c.call('events.list', ROUTE_EVENTS_LIST, { cursor, limit: limit ? String(limit) : undefined })
  }
}

/** Where events are delivered. */
export class WebhookEndpointsService {
  constructor(private readonly c: TuCapi) {}
  /** Registers the https url. KEEP THE SECRET: it is shown once. */
  create(url: string): Promise<WebhookEndpoint> {
    return this.c.call('webhookEndpoints.create', ROUTE_WEBHOOK_ENDPOINTS_CREATE, undefined, { url })
  }
}

// ─── webhooks: the delicate part ─────────────────────────────────────────────

/**
 * Verifies and reads an event.
 *
 * ⚠️ `body` MUST BE THE RAW BODY, not the parsed object. In Express use
 * `express.raw({ type: 'application/json' })`, not `express.json()`.
 *
 * The header may carry SEVERAL signatures separated by spaces during a secret
 * rotation: one match is enough.
 *
 * @throws SignatureError if it does not verify. Discard the event.
 */
export function verifyEvent(body: Buffer | Uint8Array, signature: string | undefined, timestamp: string | undefined, secret: string, nowMs = Date.now()): Event {
  if (!signature) throw new SignatureError('missing signature header')
  if (!timestamp) throw new SignatureError('missing timestamp header')
  const seconds = Number.parseInt(timestamp.trim(), 10)
  if (!Number.isFinite(seconds)) throw new SignatureError('invalid timestamp')
  if (Math.abs(nowMs - seconds * 1000) > CLOCK_TOLERANCE_MS) throw new SignatureError('outside the clock window')

  const expected = Buffer.from(sign(body, seconds, secret), 'utf8')
  const matches = signature
    .trim()
    .split(/\s+/)
    .some((candidate) => {
      const b = Buffer.from(candidate, 'utf8')
      // timingSafeEqual throws on different lengths; the length is public, so comparing it leaks nothing.
      return expected.length === b.length && timingSafeEqual(expected, b)
    })
  if (!matches) throw new SignatureError('no match')
  return JSON.parse(Buffer.from(body).toString('utf8')) as Event
}

/** Takes the headers of a request (Express `req.headers` or fetch `Headers`) and verifies the event. */
export function parseEvent(body: Buffer | Uint8Array, headers: Headers | Record<string, string | string[] | undefined>, secret: string): Event {
  const take = (name: string): string | undefined => {
    if (typeof (headers as Headers).get === 'function') return (headers as Headers).get(name) ?? undefined
    const v = (headers as Record<string, string | string[] | undefined>)[name]
    return Array.isArray(v) ? v[0] : v
  }
  return verifyEvent(body, take(HEADER_SIGNATURE), take(HEADER_TIMESTAMP), secret)
}

/** Computes "v1=<hex>" over `timestamp + "." + body`. Exported for tests and for signing your own fixtures. */
export function sign(body: Buffer | Uint8Array, seconds: number, secret: string): string {
  const mac = createHmac('sha256', secret)
  mac.update(String(seconds))
  mac.update('.')
  mac.update(body)
  return `${SIGNATURE_VERSION}=${mac.digest('hex')}`
}
