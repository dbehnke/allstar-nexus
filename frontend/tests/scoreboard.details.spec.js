import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ScoreboardCard from '../src/components/ScoreboardCard.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('ScoreboardCard - details fetch', () => {
  it('shows rested XP directly on card and opens modal when clicked', async () => {
    setActivePinia(createPinia())
    const mockProfile = {
      callsign: 'KX9Z',
      rested_bonus_seconds: 7200,
      level: 5,
      experience_points: 1234,
      next_level_xp: 2000,
      total_talk_time_seconds: 3600,
      daily_xp: 100,
      weekly_xp: 500
    }

    // stub global fetch for modal
    global.fetch = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve(mockProfile) }))

    const wrapper = mount(ScoreboardCard, {
      props: {
        scoreboard: [{ callsign: 'KX9Z', level: 5, experience_points: 1234, rested_bonus_seconds: 7200 }],
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: false,
      }
    })

    // Rested XP should be visible directly on the card
    expect(wrapper.html()).toContain('Rested:')
    expect(wrapper.html()).toContain('2.0 hours')

    // Click the card entry to open modal
    const entry = wrapper.find('.entry')
    expect(entry.exists()).toBe(true)
    await entry.trigger('click')

    // Wait for modal to appear
    await wrapper.vm.$nextTick()

    // Modal should be visible with callsign
    expect(wrapper.html()).toContain('KX9Z')
  })
})
