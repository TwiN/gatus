const test = require('node:test')
const assert = require('node:assert/strict')

const dashboardQuery = import('../src/utils/dashboard-query.mjs')

test('getFirstQueryValue handles Vue Router query values', async () => {
  const { getFirstQueryValue } = await dashboardQuery
  assert.equal(getFirstQueryValue('api'), 'api')
  assert.equal(getFirstQueryValue(['api', 'web']), 'api')
  assert.equal(getFirstQueryValue([null, 'web']), 'web')
  assert.equal(getFirstQueryValue([1, 'web']), 'web')
  assert.equal(getFirstQueryValue([undefined, 'web']), 'web')
  assert.equal(getFirstQueryValue([1, undefined, null]), undefined)
  assert.equal(getFirstQueryValue(null), undefined)
  assert.equal(getFirstQueryValue(undefined), undefined)
})

test('normalizeSearchQuery returns a safe string', async () => {
  const { normalizeSearchQuery } = await dashboardQuery
  assert.equal(normalizeSearchQuery('database'), 'database')
  assert.equal(normalizeSearchQuery(['database', 'cache']), 'database')
  assert.equal(normalizeSearchQuery('  database cluster  '), 'database cluster')
  assert.equal(normalizeSearchQuery('database cluster'), 'database cluster')
  assert.equal(normalizeSearchQuery(['  database  ', 'cache']), 'database')
  assert.equal(normalizeSearchQuery(null), '')
})

test('normalizeDashboardOption accepts known values and falls back for invalid values', async () => {
  const {
    DASHBOARD_FILTER_VALUES,
    DASHBOARD_SORT_VALUES,
    normalizeDashboardOption,
  } = await dashboardQuery

  assert.equal(normalizeDashboardOption('failing', DASHBOARD_FILTER_VALUES, 'none'), 'failing')
  assert.equal(normalizeDashboardOption('invalid', DASHBOARD_FILTER_VALUES, 'none'), 'none')
  assert.equal(normalizeDashboardOption(['group', 'health'], DASHBOARD_SORT_VALUES, 'name'), 'group')
  assert.equal(normalizeDashboardOption(null, DASHBOARD_SORT_VALUES, 'health'), 'health')
})

test('withQueryValue preserves unrelated parameters and sets or removes the target', async () => {
  const { withQueryValue } = await dashboardQuery
  assert.deepEqual(withQueryValue({ error: 'access_denied' }, 'search', 'api'), {
    error: 'access_denied',
    search: 'api',
  })
  assert.deepEqual(withQueryValue({ search: 'api', sort: 'group' }, 'search', ''), {
    sort: 'group',
  })
  assert.deepEqual(withQueryValue({}, 'page', 0), { page: 0 })
  assert.deepEqual(withQueryValue({}, 'enabled', false), { enabled: false })
  assert.deepEqual(withQueryValue({ search: 'api' }, 'search', null), {})
  assert.deepEqual(withQueryValue({ search: 'api' }, 'search', undefined), {})
})
