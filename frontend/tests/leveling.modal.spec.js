import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LevelingHelpModal from '../src/components/LevelingHelpModal.vue'

describe('LevelingHelpModal', () => {
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
})
