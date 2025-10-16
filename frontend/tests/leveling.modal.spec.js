import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LevelingHelpModal from '../src/components/LevelingHelpModal.vue'

describe('LevelingHelpModal', () => {
  beforeEach(() => {
    // Reset fetch mocks before each test
    global.fetch = vi.fn()
  })

  it('renders XP values with human-readable times and weekly cap', () => {
    const levelConfig = { '1': 360, '2': 720 }
    const wrapper = mount(LevelingHelpModal, {
      props: {
        visible: true,
        levelConfig,
        renownXP: 36000,
        renownEnabled: true,
        weeklyCapSeconds: 3600 * 10, // 10 hours
        restedEnabled: true,
        restedAccumulationRate: 1.5,
        restedMaxHours: 336,
        restedMultiplier: 2.0,
      }
    })

    // Should show level 1 row and human readable time for 360s
    expect(wrapper.html()).toContain('360')
  expect(wrapper.html()).toContain('6 minutes')

  // Should show weekly cap in human-readable form
  expect(wrapper.html()).toContain('10.0 hours')

    // Should show renown configured value
    expect(wrapper.html()).toContain('36000')

    // Should show rested server values
    expect(wrapper.html()).toContain('Accumulation rate')
    expect(wrapper.html()).toContain('1.5')
    expect(wrapper.html()).toContain('Maximum cap')
    expect(wrapper.html()).toContain('336 hours')
    expect(wrapper.html()).toContain('Multiplier when rested')
    expect(wrapper.html()).toContain('2.0x')
  })

  it('fetches thresholds from API when modal opens and displays server data indicator', async () => {
    // Mock successful API response
    const mockThresholds = {
      levels: [
        { level: 1, xp: 360 },
        { level: 2, xp: 360 },
        { level: 10, xp: 360 },
        { level: 11, xp: 510 }
      ],
      calculation: 'level-based pow 1.8 scaled'
    }
    
    global.fetch = vi.fn(() => 
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockThresholds)
      })
    )

    // Mount with visible=false initially
    const wrapper = mount(LevelingHelpModal, {
      props: {
        visible: false,
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: true
      }
    })

    // Make it visible to trigger the watch
    await wrapper.setProps({ visible: true })
    
    // Wait for fetch to complete
    await flushPromises()

    // Should have called the API
    expect(global.fetch).toHaveBeenCalledWith('/api/leveling/thresholds?max_level=60')

    // Should show server data indicator
    expect(wrapper.html()).toContain('Values provided by server')
    
    // Should use server values (level 11 with 510 XP is a good indicator)
    expect(wrapper.html()).toContain('510')
  })

  it('falls back to local calculation when API fails and shows fallback indicator', async () => {
    // Mock failed API response
    global.fetch = vi.fn(() => 
      Promise.reject(new Error('Network error'))
    )

    // Mount with visible=false initially
    const wrapper = mount(LevelingHelpModal, {
      props: {
        visible: false,
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: true
      }
    })

    // Make it visible to trigger the watch
    await wrapper.setProps({ visible: true })
    
    // Wait for fetch to complete
    await flushPromises()

    // Should show fallback indicator
    expect(wrapper.html()).toContain('Using local calculation')
    
    // Should still show XP values using fallback
    expect(wrapper.html()).toContain('360') // level 1 fallback
  })

  it('uses levelConfig prop when server data unavailable', async () => {
    // Mock failed API response
    global.fetch = vi.fn(() => 
      Promise.reject(new Error('Network error'))
    )

    const levelConfig = { '1': 500, '2': 600 }
    
    // Mount with visible=false initially
    const wrapper = mount(LevelingHelpModal, {
      props: {
        visible: false,
        levelConfig,
        renownXP: 36000,
        renownEnabled: true
      }
    })

    // Make it visible to trigger the watch
    await wrapper.setProps({ visible: true })
    
    // Wait for fetch to complete
    await flushPromises()

    // Should use prop values as fallback
    expect(wrapper.html()).toContain('500') // level 1 from prop
    expect(wrapper.html()).toContain('600') // level 2 from prop
  })

  it('only fetches thresholds once when modal is opened', async () => {
    global.fetch = vi.fn(() => 
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ levels: [], calculation: 'test' })
      })
    )

    const wrapper = mount(LevelingHelpModal, {
      props: {
        visible: false,
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: true
      }
    })

    // Open and close modal multiple times
    await wrapper.setProps({ visible: true })
    await flushPromises()
    await wrapper.setProps({ visible: false })
    await wrapper.setProps({ visible: true })
    await flushPromises()

    // Should only fetch once
    expect(global.fetch).toHaveBeenCalledTimes(1)
  })
})
