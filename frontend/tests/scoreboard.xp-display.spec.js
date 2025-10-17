import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ScoreboardCard from '../src/components/ScoreboardCard.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('ScoreboardCard - XP display fix', () => {
  it('uses next_level_xp from backend response instead of calculating locally', async () => {
    setActivePinia(createPinia())
    
    // Mock a level 21 user with backend-provided next_level_xp
    // Backend correctly sends the XP required to reach level 22
    const mockScoreboard = [{
      callsign: 'KF8S',
      level: 21,
      experience_points: 221,
      next_level_xp: 1066, // Correct value from backend
      rested_bonus_seconds: 0,
      renown_level: 0
    }]

    const wrapper = mount(ScoreboardCard, {
      props: {
        scoreboard: mockScoreboard,
        levelConfig: {}, // Empty config to test fallback isn't used incorrectly
        renownXP: 36000,
        renownEnabled: false
      }
    })

    // The LevelProgressBar should receive the correct next_level_xp value (1066)
    // not the incorrectly calculated value from the old fallback formula
    const progressBar = wrapper.findComponent({ name: 'LevelProgressBar' })
    expect(progressBar.exists()).toBe(true)
    
    // Check that the required-xp prop is the correct value from backend
    expect(progressBar.props('requiredXp')).toBe(1066)
    expect(progressBar.props('currentXp')).toBe(221)
    expect(progressBar.props('level')).toBe(21)
  })

  it('falls back to requiredXP calculation when next_level_xp is missing', async () => {
    setActivePinia(createPinia())
    
    // Mock a user without next_level_xp (for backwards compatibility)
    const mockScoreboard = [{
      callsign: 'TEST',
      level: 5,
      experience_points: 100,
      // next_level_xp is missing - should fall back to requiredXP function
      rested_bonus_seconds: 0,
      renown_level: 0
    }]

    const wrapper = mount(ScoreboardCard, {
      props: {
        scoreboard: mockScoreboard,
        levelConfig: { 5: 360 }, // Provide level config for fallback
        renownXP: 36000,
        renownEnabled: false
      }
    })

    const progressBar = wrapper.findComponent({ name: 'LevelProgressBar' })
    expect(progressBar.exists()).toBe(true)
    
    // Should use the fallback calculation (level 5 = 360 XP)
    expect(progressBar.props('requiredXp')).toBe(360)
  })
})
