import { describe, expect, it } from 'vitest'
import { getActivityColor, ACTIVITY_COLORS } from './activityColors'

describe('getActivityColor', () => {
  it('resolves known non-tool categories correctly', () => {
    expect(getActivityColor('kanban')).toEqual(ACTIVITY_COLORS.kanban)
    expect(getActivityColor('pool')).toEqual(ACTIVITY_COLORS.pool)
    expect(getActivityColor('git')).toEqual(ACTIVITY_COLORS.git)
    expect(getActivityColor('agent')).toEqual(ACTIVITY_COLORS.agent)
  })

  it('resolves specific tools correctly', () => {
    expect(getActivityColor('tool', 'bash')).toEqual(ACTIVITY_COLORS['tool:bash'])
    expect(getActivityColor('tool', 'execute_command')).toEqual(ACTIVITY_COLORS['tool:bash'])
    expect(getActivityColor('tool', 'run_command')).toEqual(ACTIVITY_COLORS['tool:bash'])

    expect(getActivityColor('tool', 'read_file')).toEqual(ACTIVITY_COLORS['tool:read'])
    expect(getActivityColor('tool', 'view_file')).toEqual(ACTIVITY_COLORS['tool:read'])

    expect(getActivityColor('tool', 'edit_file')).toEqual(ACTIVITY_COLORS['tool:edit'])
    expect(getActivityColor('tool', 'replace_file_content')).toEqual(ACTIVITY_COLORS['tool:edit'])

    expect(getActivityColor('tool', 'write_to_file')).toEqual(ACTIVITY_COLORS['tool:write'])
    expect(getActivityColor('tool', 'create_file')).toEqual(ACTIVITY_COLORS['tool:write'])
  })

  it('falls back to generic tool for unrecognized tool name', () => {
    expect(getActivityColor('tool', 'custom_weather_lookup')).toEqual(ACTIVITY_COLORS['tool:other'])
    expect(getActivityColor('tool', 'unknown_rpc')).toEqual(ACTIVITY_COLORS['tool:other'])
  })

  it('falls back to default fallback for completely unknown category', () => {
    expect(getActivityColor('unrecognized_cat')).toEqual(ACTIVITY_COLORS.fallback)
    expect(getActivityColor('')).toEqual(ACTIVITY_COLORS.fallback)
  })
})
