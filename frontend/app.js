const DEFAULT_API_BASE_URL = window.location.origin

const state = {
  apiBaseUrl: localStorage.getItem('location-service-api-base-url') || DEFAULT_API_BASE_URL,
  activeTab: 'browse',
  shortCodes: false,
  statsLoaded: false,
  statsAnimated: false,
  lastResponse: {},
}

const els = {
  apiBaseUrl: document.getElementById('apiBaseUrl'),
  resetApiUrl: document.getElementById('resetApiUrl'),
  healthDot: document.getElementById('healthDot'),
  healthText: document.getElementById('healthText'),
  refreshHealth: document.getElementById('refreshHealth'),
  tabs: Array.from(document.querySelectorAll('.nav-tab')),
  views: {
    browse: document.getElementById('browseView'),
    search: document.getElementById('searchView'),
  },
  viewTitle: document.getElementById('viewTitle'),
  viewSubtitle: document.getElementById('viewSubtitle'),
  shortCodeToggle: document.getElementById('shortCodeToggle'),
  reloadData: document.getElementById('reloadData'),
  resetData: document.getElementById('resetData'),
  quickSearch: document.getElementById('quickSearch'),
  provinceCount: document.getElementById('provinceCount'),
  regencyCount: document.getElementById('regencyCount'),
  districtCount: document.getElementById('districtCount'),
  villageCount: document.getElementById('villageCount'),
  selectionSummary: document.getElementById('selectionSummary'),
  selectionTitle: document.getElementById('selectionTitle'),
  selectionSubtitle: document.getElementById('selectionSubtitle'),
  selectionMetrics: document.getElementById('selectionMetrics'),
  treeRoot: document.getElementById('treeRoot'),
  treeFilter: document.getElementById('treeFilter'),
  treeRowCount: document.getElementById('treeRowCount'),
  breadcrumb: document.getElementById('breadcrumb'),
  searchInput: document.getElementById('searchInput'),
  searchLimit: document.getElementById('searchLimit'),
  runSearch: document.getElementById('runSearch'),
  searchRows: document.getElementById('searchRows'),
  searchMeta: document.getElementById('searchMeta'),
  responseOutput: document.getElementById('responseOutput'),
  responseMethod: document.getElementById('responseMethod'),
  copyEndpoint: document.getElementById('copyEndpoint'),
  copyResponse: document.getElementById('copyResponse'),
  responseDrawer: document.getElementById('responseDrawer'),
  responseDrawerToggle: document.getElementById('responseDrawerToggle'),
  toast: document.getElementById('toast'),
  openSidebar: document.getElementById('openSidebar'),
  closeSidebar: document.getElementById('closeSidebar'),
  sidebarOverlay: document.getElementById('sidebarOverlay'),
  sidebar: document.getElementById('sidebar'),
}

// ── Utilities ──

function apiBaseUrl() {
  return state.apiBaseUrl.replace(/\/+$/, '')
}

function setLastResponse(requestLine, payload) {
  state.lastResponse = payload
  const url = requestLine.replace(/^GET\s+/, '')
  els.responseMethod.textContent = requestLine
  els.responseMethod.href = url
  els.responseOutput.textContent = JSON.stringify(payload, null, 2)
  els.responseDrawer.classList.add('has-data')
}

function showToast(msg) {
  els.toast.textContent = msg
  els.toast.classList.add('show')
  clearTimeout(showToast.t)
  showToast.t = setTimeout(() => els.toast.classList.remove('show'), 2800)
}

async function request(path, params = {}, silent = false) {
  const url = new URL(apiBaseUrl() + path)
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && String(v).trim() !== '') url.searchParams.set(k, String(v).trim())
  })
  const res = await fetch(url.toString(), { headers: { Accept: 'application/json' } })
  const payload = await res.json().catch(() => ({}))
  if (!silent) setLastResponse(`GET ${url.toString()}`, payload)
  if (!res.ok || payload.status === false) {
    throw new Error(payload?.error?.message || payload?.message || `Request failed (${res.status})`)
  }
  return Array.isArray(payload.data) ? payload.data : payload.data || []
}

function codeFormatParams() {
  return state.shortCodes ? { code_format: 'short' } : {}
}

function formatCount(value) {
  return Number(value || 0).toLocaleString('en-US')
}

function animateCount(el, target, shouldAnimate) {
  const end = Number(target || 0)
  if (!shouldAnimate || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    el.textContent = formatCount(end)
    return
  }

  const duration = 900
  const startTime = performance.now()

  function tick(now) {
    const progress = Math.min((now - startTime) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    el.textContent = formatCount(Math.round(end * eased))
    if (progress < 1) requestAnimationFrame(tick)
  }

  requestAnimationFrame(tick)
}

function setStats(stats) {
  const shouldAnimate = !state.statsAnimated
  animateCount(els.provinceCount, stats.provinces, shouldAnimate)
  animateCount(els.regencyCount, stats.regencies, shouldAnimate)
  animateCount(els.districtCount, stats.districts, shouldAnimate)
  animateCount(els.villageCount, stats.villages, shouldAnimate)
  state.statsLoaded = true
  state.statsAnimated = true
}

async function loadStats() {
  try {
    const stats = await request('/api/locations/stats', {}, true)
    setStats(stats)
  } catch (err) {
    state.statsLoaded = false
    showToast(`Stats unavailable: ${err.message}`)
  }
}

async function refreshData() {
  await loadStats()
  resetSelectionSummary()
  await loadTree()
}

function scopedStatsParams(item) {
  const code = item.full_code || item.code
  switch (item.level) {
    case 'province': return { province_code: code }
    case 'regency': return { regency_code: code }
    case 'district': return { district_code: code }
    default: return {}
  }
}

function scopedMetric(label, value, tone) {
  return `
    <div class="selection-metric selection-metric-${tone}">
      <span>${label}</span>
      <strong>${formatCount(value)}</strong>
    </div>
  `
}

function scopedInlineText(item, stats) {
  if (item.level === 'province') {
    return `${formatCount(stats.regencies)} reg · ${formatCount(stats.districts)} dist · ${formatCount(stats.villages)} vil`
  }
  if (item.level === 'regency') {
    return `${formatCount(stats.districts)} dist · ${formatCount(stats.villages)} vil`
  }
  if (item.level === 'district') {
    return `${formatCount(stats.villages)} vil`
  }
  return ''
}

function resetSelectionSummary() {
  els.selectionSummary.classList.add('is-hidden')
  els.selectionTitle.textContent = 'Indonesia'
  els.selectionSubtitle.textContent = 'Global overview is shown above. Select a location to see scoped counts.'
  els.selectionMetrics.innerHTML = ''
}

function setInlineStats(node, text) {
  if (!node) return
  const inlineStats = node.querySelector(':scope > .tree-row .tree-inline-stats')
  if (!inlineStats) return
  inlineStats.textContent = text
  node.classList.toggle('has-inline-stats', Boolean(text))
}

function renderScopedStats(item, stats, node) {
  const levelLabel = item.level.charAt(0).toUpperCase() + item.level.slice(1)
  setInlineStats(node, scopedInlineText(item, stats))
  els.selectionSummary.classList.remove('is-hidden')
  els.selectionTitle.textContent = item.name
  els.selectionSubtitle.textContent = `${levelLabel} code ${item.full_code || item.code}`

  if (item.level === 'province') {
    els.selectionMetrics.innerHTML = [
      scopedMetric('Regencies', stats.regencies, 'regency'),
      scopedMetric('Districts', stats.districts, 'district'),
      scopedMetric('Villages', stats.villages, 'village'),
    ].join('')
    return
  }

  if (item.level === 'regency') {
    els.selectionMetrics.innerHTML = [
      scopedMetric('Districts', stats.districts, 'district'),
      scopedMetric('Villages', stats.villages, 'village'),
    ].join('')
    return
  }

  if (item.level === 'district') {
    els.selectionMetrics.innerHTML = scopedMetric('Villages', stats.villages, 'village')
    return
  }

  els.selectionMetrics.innerHTML = ''
}

async function loadScopedStats(item, node) {
  setInlineStats(node, 'Counting…')
  try {
    const stats = await request('/api/locations/stats', scopedStatsParams(item), true)
    renderScopedStats(item, stats, node)
  } catch (err) {
    setInlineStats(node, '')
    els.selectionSummary.classList.remove('is-hidden')
    els.selectionTitle.textContent = item.name
    els.selectionSubtitle.textContent = `Scoped counts unavailable: ${err.message}`
    els.selectionMetrics.innerHTML = ''
  }
}

// ── Health ──

async function checkHealth() {
  els.healthDot.className = 'status-dot'
  els.healthText.textContent = 'Checking…'
  try {
    await request('/healthz', {}, true)
    els.healthDot.className = 'status-dot ok'
    els.healthText.textContent = 'Service online'
  } catch {
    els.healthDot.className = 'status-dot fail'
    els.healthText.textContent = 'Service unavailable'
  }
}

// ── Tree View ──

const CHEVRON_SVG = '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>'

function fetchChildren(item) {
  const p = codeFormatParams()
  const code = item.full_code || item.code
  switch (item.level) {
    case 'province': return request('/api/locations/regencies', { province_code: code, ...p })
    case 'regency': return request('/api/locations/districts', { regency_code: code, ...p })
    case 'district': return request('/api/locations/villages', { district_code: code, ...p })
    default: return Promise.resolve([])
  }
}

function createTreeNode(item) {
  const isLeaf = item.level === 'village'
  const node = document.createElement('div')
  node.className = 'tree-node' + (isLeaf ? ' leaf' : '')
  node.dataset.code = item.full_code || item.code
  node.dataset.level = item.level
  node.dataset.name = item.name.toLowerCase()

  const row = document.createElement('div')
  row.className = 'tree-row'
  row.setAttribute('role', 'treeitem')
  row.setAttribute('aria-expanded', 'false')

  const chevron = document.createElement('span')
  chevron.className = 'tree-chevron'
  chevron.innerHTML = CHEVRON_SVG

  const code = document.createElement('span')
  code.className = 'tree-code'
  code.textContent = item.code
  code.title = 'Click to copy'
  code.addEventListener('click', (e) => {
    e.stopPropagation()
    navigator.clipboard.writeText(item.full_code || item.code).then(() => showToast(`Copied: ${item.full_code || item.code}`))
  })

  const name = document.createElement('span')
  name.className = 'tree-name'
  name.textContent = item.name

  const inlineStats = document.createElement('span')
  inlineStats.className = 'tree-inline-stats'

  const badge = document.createElement('span')
  badge.className = `tree-badge tree-badge-${item.level}`
  badge.textContent = item.level

  row.append(chevron, code, name, inlineStats, badge)
  node.appendChild(row)

  if (!isLeaf) {
    const children = document.createElement('div')
    children.className = 'tree-children'
    children.setAttribute('role', 'group')
    node.appendChild(children)

    row.addEventListener('click', () => toggleNode(node, item))
  }

  return node
}

async function toggleNode(node, item) {
  const children = node.querySelector(':scope > .tree-children')
  const row = node.querySelector(':scope > .tree-row')

  if (node.classList.contains('expanded')) {
    node.classList.remove('expanded')
    row.setAttribute('aria-expanded', 'false')
    return
  }

  node.classList.add('expanded')
  row.setAttribute('aria-expanded', 'true')
  loadScopedStats(item, node)

  // already loaded
  if (children.dataset.loaded) return

  // show loading
  const loading = document.createElement('div')
  loading.className = 'tree-loading'
  loading.textContent = 'Loading…'
  children.appendChild(loading)
  children.dataset.loaded = '1'

  try {
    const items = await fetchChildren(item)
    children.removeChild(loading)
    if (!items.length) {
      const empty = document.createElement('div')
      empty.className = 'tree-empty'
      empty.textContent = 'No data'
      children.appendChild(empty)
    } else {
      items.forEach(child => children.appendChild(createTreeNode(child)))
    }
  } catch (err) {
    children.removeChild(loading)
    const errEl = document.createElement('div')
    errEl.className = 'tree-empty'
    errEl.style.color = 'var(--danger)'
    errEl.textContent = err.message
    children.appendChild(errEl)
    children.dataset.loaded = ''
  }
}

async function loadTree() {
  els.treeRoot.innerHTML = ''
  const loading = document.createElement('div')
  loading.className = 'tree-loading'
  loading.textContent = 'Loading provinces…'
  els.treeRoot.appendChild(loading)

  try {
    const provinces = await request('/api/locations/provinces')
    els.treeRoot.removeChild(loading)
    if (!state.statsLoaded) els.provinceCount.textContent = provinces.length
    els.treeRowCount.textContent = provinces.length
    provinces.forEach(p => els.treeRoot.appendChild(createTreeNode(p)))
  } catch (err) {
    els.treeRoot.removeChild(loading)
    const errEl = document.createElement('div')
    errEl.className = 'tree-empty'
    errEl.style.color = 'var(--danger)'
    errEl.textContent = err.message
    els.treeRoot.appendChild(errEl)
    showToast(err.message)
  }
}

function filterTree() {
  const q = els.treeFilter.value.trim().toLowerCase()
  const nodes = els.treeRoot.querySelectorAll('.tree-node[data-level="province"]')
  let visible = 0
  nodes.forEach(node => {
    const match = !q || node.dataset.name.includes(q) || node.dataset.code.includes(q)
    node.style.display = match ? '' : 'none'
    if (match) visible++
  })
  els.treeRowCount.textContent = visible
}

// ── Breadcrumb (simplified — shows nothing for tree, user navigates via tree) ──

function renderBreadcrumb() {
  els.breadcrumb.innerHTML = ''
}

// ── Search ──

function renderSearchRows(tbody, items) {
  tbody.innerHTML = ''
  if (!items.length) {
    const row = document.createElement('tr')
    row.className = 'empty-row'
    const cell = document.createElement('td')
    cell.colSpan = 6
    cell.textContent = 'No results'
    row.appendChild(cell)
    tbody.appendChild(row)
    return
  }
  items.forEach((item) => {
    const row = document.createElement('tr')
    row.className = 'search-row'
    ;['code', 'full_code', 'name', 'level', 'parent_code'].forEach((col, idx) => {
      const cell = document.createElement('td')
      const value = item[col] || '-'
      if (idx === 0 && value !== '-') {
        cell.className = 'code-cell'
        cell.title = 'Click to copy'
        cell.addEventListener('click', (e) => {
          e.stopPropagation()
          navigator.clipboard.writeText(value).then(() => showToast(`Copied: ${value}`))
        })
        cell.textContent = value
      } else if (col === 'level' && value !== '-') {
        const badge = document.createElement('span')
        badge.className = `level-badge level-${value}`
        badge.textContent = value
        cell.appendChild(badge)
      } else {
        cell.textContent = value
      }
      row.appendChild(cell)
    })
    const action = document.createElement('td')
    action.className = 'action-cell'
    action.innerHTML = `<button class="browse-btn" title="Browse this location"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg> Browse</button>`
    action.querySelector('.browse-btn').addEventListener('click', (e) => {
      e.stopPropagation()
      navigateToBrowse(item)
    })
    row.appendChild(action)
    row.addEventListener('click', () => navigateToBrowse(item))
    tbody.appendChild(row)
  })
}

async function navigateToBrowse(item) {
  switchTab('browse')
  // expand tree to the item
  const fc = item.full_code || item.code
  const parts = fc.split('.')
  let parentNode = els.treeRoot

  for (let i = 0; i < parts.length - 1; i++) {
    const code = parts.slice(0, i + 1).join('.')
    let node = parentNode.querySelector(`:scope > .tree-node[data-code="${code}"]`)
    if (!node) break
    if (!node.classList.contains('expanded')) {
      const row = node.querySelector(':scope > .tree-row')
      row.click()
      // wait for load
      await new Promise(r => setTimeout(r, 600))
    }
    parentNode = node.querySelector(':scope > .tree-children')
    if (!parentNode) break
  }

  // scroll to target
  const target = document.querySelector(`.tree-node[data-code="${fc}"]`)
  if (target) {
    target.querySelector('.tree-row').style.background = 'var(--accent-light)'
    target.scrollIntoView({ behavior: 'smooth', block: 'center' })
    setTimeout(() => target.querySelector('.tree-row').style.background = '', 2000)
  }
}

function setTableLoading(tbody, columns) {
  tbody.innerHTML = ''
  for (let i = 0; i < 5; i++) {
    const row = document.createElement('tr')
    row.className = 'skeleton-row'
    columns.forEach((_, idx) => {
      const cell = document.createElement('td')
      const bar = document.createElement('div')
      bar.className = 'skeleton-cell'
      bar.style.width = idx === 0 ? '60px' : `${60 + Math.random() * 40}%`
      cell.appendChild(bar)
      row.appendChild(cell)
    })
    tbody.appendChild(row)
  }
}

function setTableError(tbody, columns, message) {
  tbody.innerHTML = ''
  const row = document.createElement('tr')
  row.className = 'error-row'
  const cell = document.createElement('td')
  cell.colSpan = columns.length
  cell.innerHTML = `<svg class="error-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg> ${message}`
  row.appendChild(cell)
  tbody.appendChild(row)
}

async function runSearch() {
  const q = els.searchInput.value.trim()
  if (!q) { showToast('Search query is required'); return }
  setTableLoading(els.searchRows, ['', '', '', '', '', ''])
  els.searchMeta.textContent = 'Searching…'
  try {
    const rows = await request('/api/locations/search', { q, limit: els.searchLimit.value || 25 })
    renderSearchRows(els.searchRows, rows)
    const isMobile = window.innerWidth <= 768
    els.searchMeta.textContent = `${rows.length} result${rows.length === 1 ? '' : 's'}${isMobile ? ' — tap a row to browse' : ''}`
  } catch (err) {
    els.searchMeta.textContent = 'Search failed'
    setTableError(els.searchRows, ['', '', '', '', '', ''], err.message)
    showToast(err.message)
  }
}

// ── Tabs ──

function switchTab(tab) {
  state.activeTab = tab
  els.tabs.forEach(b => b.classList.toggle('active', b.dataset.tab === tab))
  Object.entries(els.views).forEach(([k, v]) => v.classList.toggle('active', k === tab))
  if (tab === 'browse') {
    els.viewTitle.textContent = 'Browse Locations'
    els.viewSubtitle.textContent = 'Explore provinces, regencies, districts, and villages.'
  }
  if (tab === 'search') {
    els.viewTitle.textContent = 'Search Locations'
    els.viewSubtitle.textContent = 'Search across all administrative levels.'
  }
}

// ── Sidebar ──

function openSidebar() { els.sidebar.classList.add('open'); els.sidebarOverlay.classList.add('show') }
function closeSidebar() { els.sidebar.classList.remove('open'); els.sidebarOverlay.classList.remove('show') }

// ── Events ──

function bindEvents() {
  els.apiBaseUrl.value = state.apiBaseUrl
  els.apiBaseUrl.placeholder = DEFAULT_API_BASE_URL
  els.resetApiUrl.addEventListener('click', () => {
    state.apiBaseUrl = DEFAULT_API_BASE_URL
    els.apiBaseUrl.value = DEFAULT_API_BASE_URL
    localStorage.removeItem('location-service-api-base-url')
    checkHealth(); refreshData()
  })
  els.openSidebar.addEventListener('click', openSidebar)
  els.closeSidebar.addEventListener('click', closeSidebar)
  els.sidebarOverlay.addEventListener('click', closeSidebar)
  els.apiBaseUrl.addEventListener('change', () => {
    state.apiBaseUrl = els.apiBaseUrl.value.trim() || DEFAULT_API_BASE_URL
    localStorage.setItem('location-service-api-base-url', state.apiBaseUrl)
    checkHealth(); refreshData()
  })
  els.refreshHealth.addEventListener('click', checkHealth)
  els.tabs.forEach(b => b.addEventListener('click', () => { switchTab(b.dataset.tab); closeSidebar() }))
  els.shortCodeToggle.addEventListener('change', () => {
    state.shortCodes = els.shortCodeToggle.checked
    loadTree()
  })
  els.responseDrawerToggle.addEventListener('click', () => {
    const open = els.responseDrawer.classList.toggle('open')
    els.responseDrawerToggle.setAttribute('aria-expanded', String(open))
  })
  els.reloadData.addEventListener('click', refreshData)
  els.resetData.addEventListener('click', () => {
    state.shortCodes = false
    els.shortCodeToggle.checked = false
    els.treeFilter.value = ''
    els.quickSearch.value = ''
    els.provinceCount.textContent = '0'
    els.regencyCount.textContent = '0'
    els.districtCount.textContent = '0'
    els.villageCount.textContent = '0'
    state.statsAnimated = false
    resetSelectionSummary()
    history.replaceState(null, '', location.pathname)
    refreshData()
    showToast('Reset')
  })

  let qst
  els.quickSearch.addEventListener('input', () => {
    clearTimeout(qst)
    const q = els.quickSearch.value.trim()
    if (!q) return
    qst = setTimeout(() => { els.searchInput.value = q; switchTab('search'); runSearch() }, 350)
  })
  els.quickSearch.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { clearTimeout(qst); const q = els.quickSearch.value.trim(); if (!q) return; els.searchInput.value = q; switchTab('search'); runSearch() }
  })

  els.treeFilter.addEventListener('input', filterTree)
  els.runSearch.addEventListener('click', runSearch)
  els.searchInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') runSearch() })
  els.searchLimit.addEventListener('keydown', (e) => { if (e.key === 'Enter') runSearch() })
  els.copyEndpoint.addEventListener('click', async () => {
    await navigator.clipboard.writeText(els.responseMethod.href)
    showToast('URL copied')
  })
  els.copyResponse.addEventListener('click', async () => {
    await navigator.clipboard.writeText(JSON.stringify(state.lastResponse, null, 2))
    showToast('Response copied')
  })
}

// ── Init ──

async function init() {
  bindEvents()
  switchTab('browse')
  resetSelectionSummary()
  await checkHealth()
  await refreshData()
}

init()
