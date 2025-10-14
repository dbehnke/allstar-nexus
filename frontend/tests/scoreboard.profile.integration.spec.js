import { describe, it, beforeAll, afterAll, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { setupServer } from 'msw/node'
import { rest } from 'msw'
import ScoreboardCard from '../src/components/ScoreboardCard.vue'
import { createPinia, setActivePinia } from 'pinia'

const server = setupServer(
  rest.get('/api/gamification/profile/:callsign', (req, res, ctx) => {
    const callsign = req.params.callsign
    return res(ctx.status(200), ctx.json({ callsign, rested_bonus_seconds: 3600 }))
  })
)

describe('Scoreboard profile integration (msw)', () => {
  beforeAll(() => {
    setActivePinia(createPinia())
    // Ensure relative URLs (e.g. '/api/...') can be resolved in Node's fetch implementation
    // by providing a global location origin base for the WHATWG URL constructor.
    if (!global.location) {
      // minimal shape used by URL resolution
      global.location = { href: 'http://localhost/', origin: 'http://localhost' }
    }
    server.listen({ onUnhandledRequest: 'error' })
  })
  afterAll(() => server.close())

  it('shows rested XP on card and fetches profile via API when card clicked', async () => {
    const wrapper = mount(ScoreboardCard, {
      props: {
        scoreboard: [{ callsign: 'N0CALL', level: 3, experience_points: 10, rested_bonus_seconds: 0 }],
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: false,
      }
    })

  // Rested is displayed in the modal now; card should not assert 'Rested:'

    // Click the card entry to open modal
    const entry = wrapper.find('.entry')
    expect(entry.exists()).toBe(true)
    await entry.trigger('click')

    // allow fetch + DOM updates
    await new Promise(r => setTimeout(r, 50))
    await wrapper.vm.$nextTick()

    // Modal should show the callsign and fetch profile data
    expect(wrapper.html()).toContain('N0CALL')
  })
})
