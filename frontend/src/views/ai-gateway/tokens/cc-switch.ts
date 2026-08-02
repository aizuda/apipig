export const CC_SWITCH_APPS = [
  { value: 'codex', label: 'Codex' },
  { value: 'claude', label: 'Claude Code' },
  { value: 'gemini', label: 'Gemini CLI' },
  { value: 'grokbuild', label: 'Grok Build' },
  { value: 'opencode', label: 'OpenCode' },
  { value: 'openclaw', label: 'OpenClaw' },
  { value: 'hermes', label: 'Hermes' },
] as const

export type CCSwitchApp = (typeof CC_SWITCH_APPS)[number]['value']

export interface CCSwitchImportSelection {
  app: CCSwitchApp
  name: string
  model?: string
}

export interface CCSwitchProviderImport {
  app: CCSwitchApp
  name: string
  endpoint: string
  apiKey: string
  model?: string
  notes?: string
}

export function createCCSwitchImportDefaults(
  name: string,
  models: string[],
): CCSwitchImportSelection {
  return {
    app: 'codex',
    name,
    model: models[0] || undefined,
  }
}

export function isUsableAccessToken(token?: string) {
  const value = token?.trim()
  return Boolean(value && !value.includes('*') && !value.startsWith('sha256:'))
}

export function resolveGatewayEndpoint(apiBase: string, currentHref: string) {
  return new URL(apiBase, currentHref).toString().replace(/\/+$/, '')
}

export function buildCCSwitchImportUrl(provider: CCSwitchProviderImport) {
  const params = new URLSearchParams({
    resource: 'provider',
    app: provider.app,
    name: provider.name,
    endpoint: provider.endpoint,
    apiKey: provider.apiKey,
  })

  if (provider.model) params.set('model', provider.model)
  if (provider.notes) params.set('notes', provider.notes)

  return `ccswitch://v1/import?${params.toString()}`
}
