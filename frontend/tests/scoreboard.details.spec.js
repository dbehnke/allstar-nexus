import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ScoreboardCard from '../src/components/ScoreboardCard.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('ScoreboardCard - details fetch', () => {
  it('fetches profile and displays rested pool when Details clicked', async () => {
    setActivePinia(createPinia())
    const mockProfile = { callsign: 'KX9Z', rested_bonus_seconds: 7200 }

    // stub global fetch
    global.fetch = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve(mockProfile) }))

    const wrapper = mount(ScoreboardCard, {
      props: {
        scoreboard: [{ callsign: 'KX9Z', level: 5, experience_points: 1234 }],
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: false,
      }
    })

    // Click the Details button
    const btn = wrapper.find('button.btn-link')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')

    // Wait for fetch to resolve and DOM to update
    await new Promise(r => setTimeout(r, 0))
    await wrapper.vm.$nextTick()

    expect(global.fetch).toHaveBeenCalled()
    expect(wrapper.html()).toContain('Rested:')
    expect(wrapper.html()).toContain('2.0 hours')
  })
})
