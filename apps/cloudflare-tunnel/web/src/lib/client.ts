import { createMockApi } from './mock'
import { realApi, type ApiClient } from './api'

/** Development defaults to the mock adapter; production builds never do. */
export const USE_MOCK = String(import.meta.env.VITE_USE_MOCK ?? '0') === '1'

export const api: ApiClient = USE_MOCK ? createMockApi() : realApi
