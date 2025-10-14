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

  it('fetches profile via API and shows rested pool', async () => {
    const wrapper = mount(ScoreboardCard, {
      props: {
        scoreboard: [{ callsign: 'N0CALL', level: 3, experience_points: 10 }],
        levelConfig: {},
        renownXP: 36000,
        renownEnabled: false,
      }
    })

    const btn = wrapper.find('button.btn-link')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')

    // allow fetch + DOM updates
    await new Promise(r => setTimeout(r, 0))
    await wrapper.vm.$nextTick()

    expect(wrapper.html()).toContain('Rested:')
    // formatTime returns singular "hour" for exactly 1.0
    expect(wrapper.html()).toContain('1.0 hour')
  })
})
