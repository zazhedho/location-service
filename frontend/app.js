const DEFAULT_API_BASE_URL = window.location.protocol.startsWith('http')
  ? window.location.origin
  : 'https://location-service-y7si.onrender.com'
  // : 'https://localhost:8080'

const state = {
  apiBaseUrl: localStorage.getItem('location-service-api-base-url') || DEFAULT_API_BASE_URL,
  activeTab: 'browse',
  shortCodes: false,
  provinces: [],
  regencies: [],
  districts: [],
  villages: [],
  selectedProvince: '',
  selectedRegency: '',
  selectedDistrict: '',
  lastResponse: {},
}

const els = {
  apiBaseUrl: document.getElementById('apiBaseUrl'),
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
  provinceFilter: document.getElementById('provinceFilter'),
  childFilter: document.getElementById('childFilter'),
  provinceRows: document.getElementById('provinceRows'),
  childRows: document.getElementById('childRows'),
  childTableTitle: document.getElementById('childTableTitle'),
  childTableTitleText: document.getElementById('childTableTitleText'),
  provinceRowCount: document.getElementById('provinceRowCount'),
  childRowCount: document.getElementById('childRowCount'),
  breadcrumb: document.getElementById('breadcrumb'),
  searchInput: document.getElementById('searchInput'),
  searchLimit: document.getElementById('searchLimit'),
  runSearch: document.getElementById('runSearch'),
  searchRows: document.getElementById('searchRows'),
  searchMeta: document.getElementById('searchMeta'),
  responseOutput: document.getElementById('responseOutput'),
  copyResponse: document.getElementById('copyResponse'),
  responseDrawer: document.getElementById('responseDrawer'),
  responseDrawerToggle: document.getElementById('responseDrawerToggle'),
  toast: document.getElementById('toast'),
  openSidebar: document.getElementById('openSidebar'),
  closeSidebar: document.getElementById('closeSidebar'),
  sidebarOverlay: document.getElementById('sidebarOverlay'),
  sidebar: document.getElementById('sidebar'),
}

function apiBaseUrl() {
  return state.apiBaseUrl.replace(/\/+$/, '')
}

function setLastResponse(requestLine, payload) {
  state.lastResponse = payload
  els.responseOutput.textContent = `// ${requestLine}\n\n${JSON.stringify(payload, null, 2)}`
}

function showToast(message) {
  els.toast.textContent = message
  els.toast.classList.add('show')
  window.clearTimeout(showToast.timer)
  showToast.timer = window.setTimeout(() => els.toast.classList.remove('show'), 2800)
}

async function request(path, params = {}, silent = false) {
  const url = new URL(apiBaseUrl() + path)
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      url.searchParams.set(key, String(value).trim())
    }
  })

  const res = await fetch(url.toString(), { headers: { Accept: 'application/json' } })
  const payload = await res.json().catch(() => ({}))
  if (!silent) setLastResponse(`GET ${url.toString()}`, payload)
  if (!res.ok || payload.status === false) {
    const message = payload?.error?.message || payload?.message || `Request failed with ${res.status}`
    throw new Error(message)
  }
  return Array.isArray(payload.data) ? payload.data : payload.data || []
}

async function checkHealth() {
  els.healthDot.className = 'status-dot'
  els.healthText.textContent = 'Checking service'
  try {
    await request('/healthz', {}, true)
    els.healthDot.className = 'status-dot ok'
    els.healthText.textContent = 'Service online'
  } catch (error) {
    els.healthDot.className = 'status-dot fail'
    els.healthText.textContent = 'Service unavailable'
  }
}

function filterRows(items, filterValue) {
  const q = String(filterValue || '').trim().toLowerCase()
  if (!q) return items
  return items.filter((item) => {
    return [item.code, item.full_code, item.name, item.level, item.parent_code]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(q))
  })
}

function setTableError(tbody, columns, message) {
  tbody.innerHTML = ''
  const row = document.createElement('tr')
  row.className = 'error-row'
  const cell = document.createElement('td')
  cell.colSpan = columns.length
  cell.innerHTML = `<span class="error-icon">⚠</span> ${message}`
  row.appendChild(cell)
  tbody.appendChild(row)
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

function renderRows(tbody, items, columns, selectedCode, onClick) {
  tbody.innerHTML = ''
  if (!items.length) {
    const row = document.createElement('tr')
    row.className = 'empty-row'
    const cell = document.createElement('td')
    cell.colSpan = columns.length
    cell.textContent = 'No data'
    row.appendChild(cell)
    tbody.appendChild(row)
    return
  }

  items.forEach((item) => {
    const row = document.createElement('tr')
    if ((item.full_code || item.code) === selectedCode) row.classList.add('selected')
    row.addEventListener('click', () => onClick?.(item))
    columns.forEach((column, idx) => {
      const cell = document.createElement('td')
      const value = item[column] || '-'
      if (idx === 0 && value !== '-') {
        cell.className = 'code-cell'
        cell.title = 'Click to copy'
        cell.addEventListener('click', (e) => {
          e.stopPropagation()
          navigator.clipboard.writeText(value).then(() => showToast(`Copied: ${value}`))
        })
        cell.textContent = value
      } else if (column === 'level' && value !== '-') {
        const badge = document.createElement('span')
        badge.className = `level-badge level-${value}`
        badge.textContent = value
        cell.appendChild(badge)
      } else {
        cell.textContent = value
      }
      row.appendChild(cell)
    })
    tbody.appendChild(row)
  })
}

function renderBreadcrumb() {
  const crumbs = []
  if (state.selectedProvince) {
    const p = state.provinces.find(x => (x.full_code || x.code) === state.selectedProvince)
    crumbs.push({ label: p ? p.name : state.selectedProvince, onClick: () => { state.selectedProvince = ''; state.selectedRegency = ''; state.selectedDistrict = ''; syncToUrl(); onProvinceChange() } })
  }
  if (state.selectedRegency) {
    const r = state.regencies.find(x => (x.full_code || x.code) === state.selectedRegency)
    crumbs.push({ label: r ? r.name : state.selectedRegency, onClick: () => { state.selectedRegency = ''; state.selectedDistrict = ''; syncToUrl(); onRegencyChange() } })
  }
  if (state.selectedDistrict) {
    const d = state.districts.find(x => (x.full_code || x.code) === state.selectedDistrict)
    crumbs.push({ label: d ? d.name : state.selectedDistrict, onClick: null })
  }

  els.breadcrumb.innerHTML = ''
  if (!crumbs.length) {
    els.breadcrumb.innerHTML = ''
    return
  }
  crumbs.forEach((crumb, i) => {
    const item = document.createElement('span')
    item.className = 'breadcrumb-item' + (i === crumbs.length - 1 ? ' current' : '')
    const btn = document.createElement('button')
    btn.textContent = crumb.label
    if (crumb.onClick) btn.addEventListener('click', crumb.onClick)
    item.appendChild(btn)
    if (i < crumbs.length - 1) {
      const sep = document.createElement('span')
      sep.className = 'breadcrumb-sep'
      sep.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>'
      item.appendChild(sep)
    }
    els.breadcrumb.appendChild(item)
  })
}

function animateCount(el, target) {
  const start = parseInt(el.textContent) || 0
  if (start === target) return
  const duration = 400
  const startTime = performance.now()
  const tick = (now) => {
    const p = Math.min((now - startTime) / duration, 1)
    el.textContent = Math.round(start + (target - start) * (1 - Math.pow(1 - p, 3)))
    if (p < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

function renderBrowse() {
  renderBreadcrumb()
  animateCount(els.provinceCount, state.provinces.length)
  animateCount(els.regencyCount, state.regencies.length)
  animateCount(els.districtCount, state.districts.length)
  animateCount(els.villageCount, state.villages.length)

  const filteredProvinces = filterRows(state.provinces, els.provinceFilter.value)
  renderRows(
    els.provinceRows,
    filteredProvinces,
    ['code', 'name'],
    state.selectedProvince,
    (item) => {
      state.selectedProvince = item.full_code || item.code
      onProvinceChange()
    },
  )
  els.provinceRowCount.textContent = filteredProvinces.length || ''

  const childRows = state.villages.length ? state.villages : state.districts.length ? state.districts : state.regencies
  const childTitle = state.villages.length ? 'Villages' : state.districts.length ? 'Districts' : 'Regencies / Cities'
  els.childTableTitleText.textContent = childTitle
  const childSelectedCode = state.villages.length ? '' : state.districts.length ? state.selectedDistrict : state.selectedRegency
  const filteredChild = filterRows(childRows, els.childFilter.value)
  renderRows(els.childRows, filteredChild, ['code', 'full_code', 'name', 'level'], childSelectedCode, (item) => {
    if (item.level === 'regency') {
      state.selectedRegency = item.full_code || item.code
      onRegencyChange()
    }
    if (item.level === 'district') {
      state.selectedDistrict = item.full_code || item.code
      onDistrictChange()
    }
  })
  els.childRowCount.textContent = filteredChild.length || ''
}

function codeFormatParams() {
  return state.shortCodes ? { code_format: 'short' } : {}
}

async function loadProvinces() {
  setTableLoading(els.provinceRows, ['code', 'name'])
  state.provinces = await request('/api/locations/provinces')
  renderBrowse()
}

async function loadRegencies() {
  state.regencies = []
  state.districts = []
  state.villages = []
  if (!state.selectedProvince) {
    renderBrowse()
    return
  }
  setTableLoading(els.childRows, ['code', 'full_code', 'name', 'level'])
  state.regencies = await request('/api/locations/regencies', {
    province_code: state.selectedProvince,
    ...codeFormatParams(),
  })
  renderBrowse()
}

async function loadDistricts() {
  state.districts = []
  state.villages = []
  if (!state.selectedRegency) {
    renderBrowse()
    return
  }
  setTableLoading(els.childRows, ['code', 'full_code', 'name', 'level'])
  state.districts = await request('/api/locations/districts', {
    regency_code: state.selectedRegency,
    ...codeFormatParams(),
  })
  renderBrowse()
}

async function loadVillages() {
  state.villages = []
  if (!state.selectedDistrict) {
    renderBrowse()
    return
  }
  setTableLoading(els.childRows, ['code', 'full_code', 'name', 'level'])
  state.villages = await request('/api/locations/villages', {
    district_code: state.selectedDistrict,
    ...codeFormatParams(),
  })
  renderBrowse()
}

async function onProvinceChange() {
  state.selectedRegency = ''
  state.selectedDistrict = ''
  syncToUrl()
  try {
    await loadRegencies()
  } catch (error) {
    setTableError(els.childRows, ['code', 'full_code', 'name', 'level'], error.message)
    showToast(error.message)
    renderBrowse()
  }
}

async function onRegencyChange() {
  state.selectedDistrict = ''
  syncToUrl()
  try {
    await loadDistricts()
  } catch (error) {
    setTableError(els.childRows, ['code', 'full_code', 'name', 'level'], error.message)
    showToast(error.message)
    renderBrowse()
  }
}

async function onDistrictChange() {
  syncToUrl()
  try {
    await loadVillages()
  } catch (error) {
    setTableError(els.childRows, ['code', 'full_code', 'name', 'level'], error.message)
    showToast(error.message)
    renderBrowse()
  }
}

async function reloadAll() {
  try {
    await loadProvinces()
    if (state.selectedProvince) await loadRegencies()
    if (state.selectedRegency) await loadDistricts()
    if (state.selectedDistrict) await loadVillages()
    showToast('Data reloaded')
  } catch (error) {
    showToast(error.message)
  }
}

function switchTab(tab) {
  state.activeTab = tab
  els.tabs.forEach((button) => button.classList.toggle('active', button.dataset.tab === tab))
  Object.entries(els.views).forEach(([key, view]) => view.classList.toggle('active', key === tab))
  if (tab === 'browse') {
    els.viewTitle.textContent = 'Browse Locations'
    els.viewSubtitle.textContent = 'Select a province and drill down to village data.'
  }
  if (tab === 'search') {
    els.viewTitle.textContent = 'Search Locations'
    els.viewSubtitle.textContent = 'Search across provinces, regencies, districts, and villages.'
  }
}

async function runSearch() {
  const q = els.searchInput.value.trim()
  if (!q) {
    showToast('Search query is required')
    return
  }
  setTableLoading(els.searchRows, ['code', 'full_code', 'name', 'level', 'parent_code'])
  els.searchMeta.textContent = 'Searching…'
  try {
    const rows = await request('/api/locations/search', {
      q,
      limit: els.searchLimit.value || 25,
    })
    renderRows(els.searchRows, rows, ['code', 'full_code', 'name', 'level', 'parent_code'])
    els.searchMeta.textContent = `${rows.length} result${rows.length === 1 ? '' : 's'}`
  } catch (error) {
    els.searchMeta.textContent = 'Search failed'
    setTableError(els.searchRows, ['code', 'full_code', 'name', 'level', 'parent_code'], error.message)
    showToast(error.message)
  }
}

function syncToUrl() {
  const params = new URLSearchParams()
  if (state.selectedProvince) params.set('province', state.selectedProvince)
  if (state.selectedRegency) params.set('regency', state.selectedRegency)
  if (state.selectedDistrict) params.set('district', state.selectedDistrict)
  const query = params.toString()
  history.replaceState(null, '', query ? `?${query}` : location.pathname)
}

function readFromUrl() {
  const params = new URLSearchParams(location.search)
  state.selectedProvince = params.get('province') || ''
  state.selectedRegency = params.get('regency') || ''
  state.selectedDistrict = params.get('district') || ''
}

function openSidebar() {
  els.sidebar.classList.add('open')
  els.sidebarOverlay.classList.add('show')
}

function closeSidebar() {
  els.sidebar.classList.remove('open')
  els.sidebarOverlay.classList.remove('show')
}

function bindEvents() {
  els.apiBaseUrl.value = state.apiBaseUrl
  els.openSidebar.addEventListener('click', openSidebar)
  els.closeSidebar.addEventListener('click', closeSidebar)
  els.sidebarOverlay.addEventListener('click', closeSidebar)
  els.apiBaseUrl.addEventListener('change', () => {
    state.apiBaseUrl = els.apiBaseUrl.value.trim() || DEFAULT_API_BASE_URL
    localStorage.setItem('location-service-api-base-url', state.apiBaseUrl)
    checkHealth()
    reloadAll()
  })
  els.refreshHealth.addEventListener('click', checkHealth)
  els.tabs.forEach((button) => button.addEventListener('click', () => { switchTab(button.dataset.tab); closeSidebar() }))
  els.shortCodeToggle.addEventListener('change', () => {
    state.shortCodes = els.shortCodeToggle.checked
    reloadAll()
  })
  els.responseDrawerToggle.addEventListener('click', () => {
    const open = els.responseDrawer.classList.toggle('open')
    els.responseDrawerToggle.setAttribute('aria-expanded', String(open))
  })
  els.reloadData.addEventListener('click', reloadAll)
  els.resetData.addEventListener('click', () => {
    state.selectedProvince = ''
    state.selectedRegency = ''
    state.selectedDistrict = ''
    state.regencies = []
    state.districts = []
    state.villages = []
    state.shortCodes = false
    els.shortCodeToggle.checked = false
    els.provinceFilter.value = ''
    els.childFilter.value = ''
    els.quickSearch.value = ''
    syncToUrl()
    renderBrowse()
    showToast('Reset')
  })

  let quickSearchTimer
  els.quickSearch.addEventListener('input', () => {
    clearTimeout(quickSearchTimer)
    const q = els.quickSearch.value.trim()
    if (!q) return
    quickSearchTimer = setTimeout(() => {
      els.searchInput.value = q
      switchTab('search')
      runSearch()
    }, 350)
  })
  els.quickSearch.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      clearTimeout(quickSearchTimer)
      const q = els.quickSearch.value.trim()
      if (!q) return
      els.searchInput.value = q
      switchTab('search')
      runSearch()
    }
  })
  els.provinceFilter.addEventListener('input', renderBrowse)
  els.childFilter.addEventListener('input', renderBrowse)
  els.runSearch.addEventListener('click', runSearch)
  els.searchInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') runSearch()
  })
  els.copyResponse.addEventListener('click', async () => {
    await navigator.clipboard.writeText(JSON.stringify(state.lastResponse, null, 2))
    showToast('Response copied')
  })
}

async function init() {
  readFromUrl()
  bindEvents()
  switchTab('browse')
  renderBrowse()
  await checkHealth()
  try {
    await loadProvinces()
    if (state.selectedProvince) await loadRegencies()
    if (state.selectedRegency) await loadDistricts()
    if (state.selectedDistrict) await loadVillages()
  } catch (error) {
    setTableError(els.provinceRows, ['code', 'name'], error.message)
    showToast(error.message)
  }
}

init()
