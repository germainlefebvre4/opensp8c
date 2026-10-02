import axios from 'axios'
import type { AgentPoolConfig } from '../components/AgentPoolModal'

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export const api = axios.create({ baseURL })

// ApiError keeps the HTTP status and the machine-readable `code` (and `target`)
// that some endpoints put in their JSON error body, so callers can branch on
// them instead of parsing the message.
export class ApiError extends Error {
  status?: number
  code?: string
  target?: string
  constructor(message: string, status?: number, code?: string, target?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.target = target
  }
}

api.interceptors.response.use(
  r => r,
  err => {
    if (axios.isAxiosError(err) && err.response) {
      const data = err.response.data
      let message: string
      if (typeof data === 'string' && data.trim()) {
        message = data.trim()
      } else if (data && typeof data === 'object') {
        message = (data as Record<string, unknown>).error as string
          ?? (data as Record<string, unknown>).message as string
          ?? err.message
      } else {
        message = err.message
      }
      const body = data && typeof data === 'object' ? (data as Record<string, unknown>) : {}
      return Promise.reject(new ApiError(
        message,
        err.response.status,
        typeof body.code === 'string' ? body.code : undefined,
        typeof body.target === 'string' ? body.target : undefined,
      ))
    }
    return Promise.reject(err)
  }
)

export const wsURL = (path: string) => {
  const base = baseURL.replace(/^http/, 'ws')
  return `${base}${path}`
}

export interface AgentStatus {
  id: string
  label: string
  installed: boolean
  version?: string
  docsUrl?: string
}

export interface SupportedLanguage {
  code: string
  nativeName: string
  englishName: string
}

export type AgentLanguageLevel = 'chat' | 'documentation' | 'code'

export type AgentLanguages = Record<AgentLanguageLevel, string>

// Partial update accepted by PATCH /api/preferences (one level at a time is fine).
export interface AgentLanguagesPatch {
  agentLanguages?: Partial<AgentLanguages>
  uiLocale?: string
}

export type Role = 'explorer' | 'ff' | 'implementer' | 'fixer' | 'documenter'
export const ROLES: Role[] = ['explorer', 'ff', 'implementer', 'fixer', 'documenter']

// One level of agent/model/effort; an empty or absent field means "inherit".
export interface RoleSetting {
  agent?: string
  model?: string
  effort?: string
}

export interface AgentSettings {
  global: RoleSetting
  roles: Record<Role, RoleSetting>
}

export interface ResolvedRole {
  agent: string
  model: string
  effort: string
}

export interface ResolvedSettings {
  global: ResolvedRole
  roles: Record<Role, ResolvedRole>
}

export type DelegationMode = 'full-autonomy' | 'hitl-review'

export interface PoolSettings {
  size: number
  delegationMode: DelegationMode
  maxAttempts: number
  // Shell command run after each agent turn; absent/empty = auto-detected.
  validationCommand?: string
}

// Partial per-workspace pool override; absent field = inherit.
export type PoolOverride = Partial<PoolSettings>

// null resets a field to inheritance.
export interface RoleSettingPatch {
  agent?: string | null
  model?: string | null
  effort?: string | null
}

export interface AgentSettingsPatch {
  global?: RoleSettingPatch
  roles?: Partial<Record<Role, RoleSettingPatch | null>>
}

export interface PoolPatch {
  size?: number | null
  delegationMode?: DelegationMode | null
  maxAttempts?: number | null
  validationCommand?: string | null
}

export interface GlobalSettingsPatch {
  agentSettings?: AgentSettingsPatch
  poolDefaults?: PoolPatch
}

export interface AgentModel {
  id: string
  label: string
  source: 'seed' | 'cli'
}

export interface AgentModelCatalogEntry {
  models: AgentModel[]
  effortLevels: string[]
  supportsModel: boolean
  supportsEffort: boolean
}

export type AgentModelCatalog = Record<string, AgentModelCatalogEntry>

export interface WorkspaceSettings {
  overrides: {
    agentSettings: AgentSettings
    pool: PoolOverride
    env: Record<string, string>
    agentEnv: Record<string, Record<string, string>>
  }
  inherited: {
    agentSettings: ResolvedSettings
    pool: PoolSettings
    env: Record<string, string>
    agentEnv: Record<string, Record<string, string>>
  }
  resolved: {
    agentSettings: ResolvedSettings
    pool: PoolSettings
  }
}

export interface WorkspaceSettingsPatch extends GlobalSettingsPatch {
  pool?: PoolPatch
  env?: Record<string, string>
  agentEnv?: Record<string, Record<string, string>>
}

export interface Preferences {
  defaultAgent: string
  agentSettings?: AgentSettings
  resolvedAgentSettings?: ResolvedSettings
  poolDefaults?: PoolSettings
  env: Record<string, string>
  agentEnv?: Record<string, Record<string, string>>
  systemEnv?: Record<string, string>
  nativeQuestionMode?: boolean
  customAgentSpecializations?: string[]
  agentLanguages?: AgentLanguages
  uiLocale?: string
  supportedLanguages?: SupportedLanguage[]
}

export interface AgentSpecializations {
  base: string[]
  custom: string[]
}

export const getAgents = () =>
  api.get<AgentStatus[]>('/api/agents').then(r => r.data)

export const getPreferences = () =>
  api.get<Preferences>('/api/preferences').then(r => r.data)

export const patchPreferences = (
  data: Omit<Partial<Preferences>, 'agentLanguages' | 'agentSettings' | 'poolDefaults'> & AgentLanguagesPatch & GlobalSettingsPatch,
) => api.patch('/api/preferences', data)

export const getAgentModels = () =>
  api.get<AgentModelCatalog>('/api/agents/models').then(r => r.data)

export const getWorkspaceSettings = (workspaceId: string) =>
  api.get<WorkspaceSettings>(`/api/workspaces/${workspaceId}/settings`).then(r => r.data)

export const patchWorkspaceSettings = (workspaceId: string, patch: WorkspaceSettingsPatch) =>
  api.patch<WorkspaceSettings>(`/api/workspaces/${workspaceId}/settings`, patch).then(r => r.data)

export const getAgentSpecializations = () =>
  api.get<AgentSpecializations>('/api/agent-specializations').then(r => r.data)

export const patchTask = (workspaceId: string, changeName: string, taskIndex: number) =>
  api.patch(`/api/workspaces/${workspaceId}/changes/${changeName}/tasks/${taskIndex}`)

export const triggerFF = (workspaceId: string, changeName: string) =>
  api.post(`/api/workspaces/${workspaceId}/changes/${changeName}/ff`)

export const resetTasks = (workspaceId: string, changeName: string) =>
  api.patch(`/api/workspaces/${workspaceId}/changes/${changeName}/tasks/reset`)

export const launchChange = (workspaceId: string, changeName: string) =>
  api.patch(`/api/workspaces/${workspaceId}/changes/${changeName}/launch`)

export const unlaunchChange = (workspaceId: string, changeName: string, force?: boolean) =>
  api.patch(`/api/workspaces/${workspaceId}/changes/${changeName}/unlaunch${force ? '?force=true' : ''}`)

// Translation key (kanban namespace) of the toast shown when a forced
// demotion fails, from the error code the backend reports.
export const unlaunchErrorKey = (err: unknown): string => {
  const code = err instanceof ApiError ? err.code : undefined
  if (code === 'change_already_merged') return 'errors.unlaunchAlreadyMerged'
  if (code === 'worker_still_running') return 'errors.unlaunchWorkerStillRunning'
  return 'errors.unlaunchFailed'
}

export const reorderReady = (workspaceId: string, order: string[]) =>
  api.put(`/api/workspaces/${workspaceId}/ready-order`, { order })

export const stopExploreSession = (workspaceId: string, changeName: string) =>
  api.delete(`/api/workspaces/${workspaceId}/changes/${changeName}/explore`)

export interface ConversationRunMeta {
  ts: string
  messageCount: number
}

export interface ConversationRun {
  ts: string
  messages: unknown[]
}

export const getConversationRuns = (workspaceId: string, changeName: string, kind: string) =>
  api.get<ConversationRunMeta[]>(`/api/workspaces/${workspaceId}/changes/${changeName}/conversations/${kind}`).then(r => r.data)

export const getConversationRun = (workspaceId: string, changeName: string, kind: string, ts: string) =>
  api.get<ConversationRun>(`/api/workspaces/${workspaceId}/changes/${changeName}/conversations/${kind}/${ts}`).then(r => r.data)

export const retagChange = (workspaceId: string, changeName: string) =>
  api.post(`/api/workspaces/${workspaceId}/changes/${changeName}/retag`)

export const promoteGhost = (workspaceId: string, ghostId: string, context: string) =>
  api.post(`/api/workspaces/${workspaceId}/explorations/${ghostId}/promote`, { context })

export const deleteGhost = (workspaceId: string, ghostId: string) =>
  api.delete(`/api/workspaces/${workspaceId}/explorations/${ghostId}`)

export const deleteChange = (workspaceId: string, changeName: string) =>
  api.delete(`/api/workspaces/${workspaceId}/changes/${changeName}`)

export const startPool = (workspaceId: string, config: AgentPoolConfig) =>
  api.post(`/api/workspaces/${workspaceId}/pool/start`, config)

export const stopPool = (workspaceId: string) =>
  api.post(`/api/workspaces/${workspaceId}/pool/stop`)

export const resumeWorker = (workspaceId: string, workerId: number) =>
  api.post(`/api/workspaces/${workspaceId}/pool/workers/${workerId}/resume`)

export interface DraftTask {
  id: string
  text: string
  done: boolean
}

export interface ExplorationDraft {
  ghostId: string
  workspaceId: string
  name: string
  description: string
  tasks: DraftTask[]
  lastSavedAt?: string
}

export const getGhostDraft = (workspaceId: string, ghostId: string) =>
  api.get<ExplorationDraft>(`/api/workspaces/${workspaceId}/explorations/${ghostId}/draft`).then(r => r.data)

export const updateGhostDraft = (workspaceId: string, ghostId: string, draft: ExplorationDraft) =>
  api.put<ExplorationDraft>(`/api/workspaces/${workspaceId}/explorations/${ghostId}/draft`, draft).then(r => r.data)

export const deleteGhostDraft = (workspaceId: string, ghostId: string) =>
  api.delete(`/api/workspaces/${workspaceId}/explorations/${ghostId}/draft`)

export const triggerDocsGenerate = (workspaceId: string) =>
  api.post(`/api/workspaces/${workspaceId}/docs/generate`)

export interface ActivityEntry {
  ts: string
  type: string
  category: string
  summary: string
  durationMs?: number
  meta?: Record<string, unknown>
}

export type PoolRunOutcome = 'running' | 'completed' | 'awaiting-review' | 'paused' | 'stopped' | 'interrupted'

export interface PoolRun {
  change: string
  ts: string
  worker_id: number
  outcome: PoolRunOutcome
  reason?: string
  started_at: string
  ended_at?: string
  line_count: number
}

export interface PoolRunDetail extends PoolRun {
  entries: ActivityEntry[]
}

export const getPoolRuns = (workspaceId: string) =>
  api.get<PoolRun[]>(`/api/workspaces/${workspaceId}/pool/runs`).then(r => r.data)

export const getPoolRun = (workspaceId: string, change: string, ts: string) =>
  api.get<PoolRunDetail>(`/api/workspaces/${workspaceId}/pool/runs/${encodeURIComponent(change)}/${encodeURIComponent(ts)}`).then(r => r.data)

export type ReviewFileStatus = 'added' | 'modified' | 'deleted'

export interface ReviewFile {
  path: string
  status: ReviewFileStatus
  additions: number
  deletions: number
  binary: boolean
}

export interface ChangeReview {
  branch: string
  base: string
  target_ahead: boolean
  files: ReviewFile[]
}

export interface ReviewDiff {
  path: string
  patch: string
  binary: boolean
  truncated: boolean
}

export interface ReviewFileContent {
  path: string
  content: string
  binary: boolean
  truncated: boolean
}

const reviewURL = (workspaceId: string, changeName: string) =>
  `/api/workspaces/${workspaceId}/changes/${encodeURIComponent(changeName)}/review`

export const getChangeReview = (workspaceId: string, changeName: string) =>
  api.get<ChangeReview>(reviewURL(workspaceId, changeName)).then(r => r.data)

export const getReviewDiff = (workspaceId: string, changeName: string, path: string) =>
  api.get<ReviewDiff>(`${reviewURL(workspaceId, changeName)}/diff`, { params: { path } }).then(r => r.data)

export const getReviewFile = (workspaceId: string, changeName: string, path: string) =>
  api.get<ReviewFileContent>(`${reviewURL(workspaceId, changeName)}/file`, { params: { path } }).then(r => r.data)

export const getActivity = (workspaceId: string, changeName: string) =>
  api.get<ActivityEntry[]>(`/api/workspaces/${workspaceId}/changes/${changeName}/activity`).then(r => r.data)
