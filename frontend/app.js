const DEFAULT_API_BASE_URL = window.location.origin

const state = {
  apiBaseUrl: localStorage.getItem('location-service-api-base-url') || DEFAULT_API_BASE_URL,
  activeTab: 'browse',
  shortCodes: false,
  statsLoaded: false,
  statsAnimated: false,
  lastResponse: {},
  map: null,
  mapTileLayer: null,
  mapLayers: null,
  mapSelectionId: 0,
  currentTheme: 'light',
}

const els = {
  appShell: document.querySelector('.app-shell'),
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
  selectionIcon: document.getElementById('selectionIcon'),
  selectionTitle: document.getElementById('selectionTitle'),
  selectionLevel: document.getElementById('selectionLevel'),
  selectionSubtitle: document.getElementById('selectionSubtitle'),
  selectionCode: document.getElementById('selectionCode'),
  selectionPostalToken: document.getElementById('selectionPostalToken'),
  selectionPostal: document.getElementById('selectionPostal'),
  selectionStatus: document.getElementById('selectionStatus'),
  selectionMetrics: document.getElementById('selectionMetrics'),
  mapStatus: document.getElementById('mapStatus'),
  mapMessage: document.getElementById('mapMessage'),
  locationMap: document.getElementById('locationMap'),
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
  responseDrawerBody: document.getElementById('responseDrawerBody'),
  drawerStatusPill: document.getElementById('drawerStatusPill'),
  toast: document.getElementById('toast'),
  openSidebar: document.getElementById('openSidebar'),
  collapseSidebar: document.getElementById('collapseSidebar'),
  sidebarOverlay: document.getElementById('sidebarOverlay'),
  sidebar: document.getElementById('sidebar'),
  themeToggle: document.getElementById('themeToggle'),
}

// ── Theme Management ──

function applyTheme(theme) {
  state.currentTheme = theme
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem('location-service-theme', theme)

  if (els.themeToggle) {
    const label = els.themeToggle.querySelector('.theme-label')
    if (label) {
      label.textContent = theme === 'dark' ? 'Tema Gelap' : 'Tema Terang'
    }
  }

  // Update Leaflet tile layer if map is initialized
  if (state.map && state.mapTileLayer) {
    state.map.removeLayer(state.mapTileLayer)
    const tileUrl = theme === 'dark'
      ? 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png'
      : 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png'

    state.mapTileLayer = window.L.tileLayer(tileUrl, {
      attribution: '&copy; <a href="https://carto.com/">CARTO</a> &copy; OpenStreetMap contributors',
      maxZoom: 19,
      subdomains: 'abcd',
    }).addTo(state.map)
  }
}

function toggleTheme() {
  const nextTheme = state.currentTheme === 'dark' ? 'light' : 'dark'
  applyTheme(nextTheme)
  showToast(`Beralih ke ${nextTheme === 'dark' ? 'Tema Gelap' : 'Tema Terang'}`)
}

// ── Utilities ──

function apiBaseUrl() {
  return state.apiBaseUrl.replace(/\/+$/, '')
}

function setLastResponse(requestLine, payload, statusCode = 200) {
  state.lastResponse = payload
  const url = requestLine.replace(/^GET\s+/, '')
  els.responseMethod.textContent = requestLine
  els.responseMethod.href = url
  els.responseMethod.setAttribute('aria-label', `Open ${requestLine}`)
  els.responseOutput.textContent = JSON.stringify(payload, null, 2)

  if (els.drawerStatusPill) {
    els.drawerStatusPill.textContent = `${statusCode} ${statusCode === 200 ? 'OK' : 'Error'}`
    els.drawerStatusPill.style.color = statusCode === 200 ? 'var(--success)' : 'var(--danger)'
  }

  els.responseDrawer.classList.add('has-data')
  if (!els.responseDrawer.classList.contains('open')) {
    els.responseDrawer.classList.add('has-unread-response')
  }
}

function setResponseDrawerState(open) {
  const wasOpen = els.responseDrawer.classList.contains('open')
  els.responseDrawer.classList.toggle('open', open)
  els.responseDrawerToggle.setAttribute('aria-expanded', String(open))
  els.responseDrawerToggle.setAttribute('aria-label', open ? 'Tutup panel respon API terakhir' : 'Buka panel respon API terakhir')
  els.responseDrawerToggle.title = open ? 'Tutup panel respon API terakhir' : 'Buka panel respon API terakhir'
  els.responseDrawerBody.setAttribute('aria-hidden', String(!open))

  if (open) {
    els.responseDrawer.classList.remove('has-unread-response')
  } else if (wasOpen && els.responseDrawerBody.contains(document.activeElement)) {
    els.responseDrawerToggle.focus()
  }
}

function showToast(msg) {
  els.toast.textContent = msg
  els.toast.classList.add('show')
  clearTimeout(showToast.t)
  showToast.t = setTimeout(() => els.toast.classList.remove('show'), 2600)
}

async function request(path, params = {}, silent = false) {
  const url = new URL(apiBaseUrl() + path)
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && String(v).trim() !== '') url.searchParams.set(k, String(v).trim())
  })

  try {
    const res = await fetch(url.toString(), { headers: { Accept: 'application/json' } })
    const payload = await res.json().catch(() => ({}))
    if (!silent) setLastResponse(`GET ${url.toString()}`, payload, res.status)
    if (!res.ok || payload.status === false) {
      const error = new Error(payload?.error?.message || payload?.message || `Request gagal (${res.status})`)
      error.status = res.status
      throw error
    }
    return Array.isArray(payload.data) ? payload.data : payload.data || []
  } catch (err) {
    if (!silent) setLastResponse(`GET ${url.toString()}`, { error: err.message }, err.status || 500)
    throw err
  }
}

function setMapStatus(message, kind = '') {
  els.mapStatus.textContent = message
  els.mapStatus.dataset.state = kind
}

function setMapMessage(message, kind = '') {
  if (message) {
    els.mapMessage.innerHTML = `
      <div class="map-empty-icon">${kind === 'error' ? '⚠️' : '📍'}</div>
      <strong>${escapeHTML(message)}</strong>
    `
    els.mapMessage.hidden = false
  } else {
    els.mapMessage.hidden = true
  }
}

function clearMapLayers() {
  state.mapLayers?.clearLayers()
}

function ensureMap() {
  if (state.map) return state.map
  if (!window.L) {
    setMapStatus('Library peta tidak tersedia.', 'error')
    setMapMessage('Library Leaflet tidak dapat dimuat.', 'error')
    return null
  }

  try {
    const map = window.L.map(els.locationMap, {
      zoomControl: true,
      fadeAnimation: true,
    }).setView([-2.5489, 118.0149], 5)

    const tileUrl = state.currentTheme === 'dark'
      ? 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png'
      : 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png'

    state.mapTileLayer = window.L.tileLayer(tileUrl, {
      attribution: '&copy; <a href="https://carto.com/">CARTO</a> &copy; OpenStreetMap contributors',
      maxZoom: 19,
      subdomains: 'abcd',
    }).addTo(map)

    state.mapLayers = window.L.layerGroup().addTo(map)
    state.map = map
    return map
  } catch (error) {
    setMapStatus('Peta tidak tersedia.', 'error')
    setMapMessage(error.message || 'Peta tidak dapat diinisialisasi.', 'error')
    return null
  }
}

function coordinatesFor(value) {
  const coordinates = value?.coordinates || value?.centroid || value
  const latitude = Number(coordinates?.latitude ?? coordinates?.lat)
  const longitude = Number(coordinates?.longitude ?? coordinates?.lng)
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return null
  if (latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180) return null
  return [latitude, longitude]
}

function normalizeLeafletPath(path) {
  if (!Array.isArray(path)) return []
  if (path.length >= 2 && !Array.isArray(path[0]) && !Array.isArray(path[1])) {
    const latitude = Number(path[0])
    const longitude = Number(path[1])
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return null
    if (latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180) return null
    return [latitude, longitude]
  }
  return path.map(normalizeLeafletPath).filter((item) => Array.isArray(item) && item.length)
}

function leafletPointCount(path) {
  if (!Array.isArray(path)) return 0
  if (path.length === 2 && path.every(Number.isFinite)) return 1
  return path.reduce((total, item) => total + leafletPointCount(item), 0)
}

function escapeHTML(value) {
  return String(value ?? '').replace(/[&<>"']/g, (character) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  })[character])
}

function locationPopup(location) {
  const name = escapeHTML(location.name || 'Wilayah Terpilih')
  const level = escapeHTML(location.level ? location.level.toUpperCase() : '')
  const code = escapeHTML(location.full_code || location.code)
  const postalCode = escapeHTML(location.postal_code)
  return `
    <div style="padding: 4px; font-family: var(--font-sans);">
      <div style="font-size: 11px; font-weight: 700; color: var(--accent); text-transform: uppercase;">${level}</div>
      <strong style="font-size: 15px; display: block; margin: 2px 0 6px; color: var(--text-primary);">${name}</strong>
      ${code ? `<div style="font-size: 12px; margin-bottom: 2px;">Kode: <code style="padding: 2px 5px; background: rgba(0,0,0,0.1); border-radius: 4px; font-weight: 600;">${code}</code></div>` : ''}
      ${postalCode ? `<div style="font-size: 12px; color: var(--color-dist);">Kode Pos: <strong>${postalCode}</strong></div>` : ''}
    </div>
  `
}

function renderMapLocation(item, detail, boundary, boundaryError) {
  const location = { ...item, ...detail }
  const path = normalizeLeafletPath(boundary?.leaflet_path)
  const centroid = coordinatesFor(detail) || coordinatesFor(boundary)
  const popup = locationPopup(location)
  clearMapLayers()

  if (leafletPointCount(path) >= 3) {
    const polygon = window.L.polygon(path, {
      color: '#f59e0b',
      fillColor: '#f59e0b',
      fillOpacity: 0.25,
      weight: 2.5,
    }).bindPopup(popup)
    state.mapLayers.addLayer(polygon)
    if (centroid) state.mapLayers.addLayer(window.L.marker(centroid).bindPopup(popup))
    state.map.fitBounds(polygon.getBounds(), { padding: [32, 32], maxZoom: 14 })
    setMapStatus(`Batas wilayah dimuat untuk ${location.name || item.name || location.code}.`, 'success')
    setMapMessage('')
    return
  }

  if (centroid) {
    state.mapLayers.addLayer(window.L.marker(centroid).bindPopup(popup))
    state.map.setView(centroid, 11)
    const boundaryMissing = !boundaryError || boundaryError.status === 404
    setMapStatus(boundaryMissing ? 'Poligon batas belum tersedia; menampilkan titik koordinat.' : 'Permintaan poligon gagal; menampilkan titik koordinat.', 'warning')
    setMapMessage('')
    return
  }

  const message = boundaryError && boundaryError.status !== 404
    ? `Batas wilayah tidak tersedia: ${boundaryError.message}`
    : 'Koordinat atau batas poligon belum tersedia untuk wilayah ini.'
  setMapStatus('Wilayah tidak dapat digambar pada peta.', 'error')
  setMapMessage(message, 'error')
}

function resetMap() {
  state.mapSelectionId += 1
  clearMapLayers()
  setMapStatus('Pilih salah satu wilayah untuk menampilkan batas peta.')
  setMapMessage('Pilih salah satu wilayah untuk menampilkan batas peta.')
}

async function selectLocation(item, node = null) {
  const code = item.full_code || item.code
  if (!code) return

  renderBreadcrumb(item, node)
  const selectionId = ++state.mapSelectionId
  setSelectionSummary(item, node)
  if (item.level !== 'village') loadScopedStats(item, node, selectionId)
  const name = item.name || code
  const map = ensureMap()
  if (!map) return

  clearMapLayers()
  setMapStatus(`Memuat ${name}…`, 'loading')
  setMapMessage('Memuat detail wilayah…', 'loading')

  try {
    const detail = await request(`/api/locations/${encodeURIComponent(code)}`, {}, true)
    if (selectionId !== state.mapSelectionId) return
    if (!detail || Array.isArray(detail) || typeof detail !== 'object') throw new Error('Detail wilayah tidak tersedia')
    if (detail.postal_code && !item.postal_code) {
      item.postal_code = detail.postal_code
      updateTreePostalCode(node, detail.postal_code)
      setSelectionSummary(item, node)
    }

    let boundary = null
    let boundaryError = null
    if (detail.has_boundary) {
      setMapStatus(`Memuat batas poligon ${detail.name || name}…`, 'loading')
      try {
        boundary = await request(`/api/locations/${encodeURIComponent(code)}/boundary`, {}, true)
      } catch (error) {
        boundaryError = error
      }
      if (selectionId !== state.mapSelectionId) return
    }

    renderMapLocation(item, detail, boundary, boundaryError)
  } catch (error) {
    if (selectionId !== state.mapSelectionId) return
    clearMapLayers()
    setMapStatus(`Gagal memuat ${name}.`, 'error')
    setMapMessage(error.message || 'Detail wilayah tidak tersedia.', 'error')
  }
}

function codeFormatParams() {
  return state.shortCodes ? { code_format: 'short' } : {}
}

function isActivationKey(event) {
  return event.key === 'Enter' || event.key === ' '
}

function formatCount(value) {
  return Number(value || 0).toLocaleString('id-ID')
}

function animateCount(el, target, shouldAnimate) {
  const end = Number(target || 0)
  if (!shouldAnimate || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    el.textContent = formatCount(end)
    return
  }

  const duration = 800
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
    showToast(`Gagal memuat statistik: ${err.message}`)
  }
}

async function refreshData() {
  resetMap()
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
  const icon = tone === 'regency' ? '🏙️' : tone === 'district' ? '🏘️' : '🏡'
  return `
    <div class="selection-metric-chip selection-metric-${tone}">
      <span class="metric-icon-mini">${icon}</span>
      <span class="metric-title">${label}:</span>
      <strong class="metric-number">${formatCount(value)}</strong>
    </div>
  `
}

function scopedInlineText(item, stats) {
  if (item.level === 'province') {
    return `${formatCount(stats.regencies)} kab/kota · ${formatCount(stats.districts)} kec · ${formatCount(stats.villages)} desa`
  }
  if (item.level === 'regency') {
    return `${formatCount(stats.districts)} kec · ${formatCount(stats.villages)} desa`
  }
  if (item.level === 'district') {
    return `${formatCount(stats.villages)} desa`
  }
  return ''
}

function resetSelectionSummary() {
  els.selectionSummary.classList.add('is-hidden')
  els.selectionTitle.textContent = 'Indonesia'
  els.selectionLevel.textContent = ''
  els.selectionCode.textContent = '—'
  els.selectionPostal.textContent = '—'
  els.selectionPostalToken.hidden = true
  els.selectionStatus.textContent = ''
  els.selectionStatus.hidden = true
  els.selectionMetrics.innerHTML = ''
}

function setSelectionSummary(item, node) {
  const level = item.level || 'province'
  const levelLabel = level.charAt(0).toUpperCase() + level.slice(1)
  const code = item.full_code || item.code || '—'
  const postal = item.postal_code || ''
  
  const iconMap = {
    province: '🗺️',
    regency: '🏙️',
    district: '🏘️',
    village: '🏡',
  }
  const icon = iconMap[level] || '📍'

  setInlineStats(node, '')
  els.selectionSummary.classList.remove('is-hidden')
  els.selectionSummary.dataset.level = level
  els.selectionTitle.textContent = item.name || code
  els.selectionLevel.textContent = levelLabel
  if (els.selectionIcon) els.selectionIcon.textContent = icon
  els.selectionCode.textContent = code
  els.selectionPostal.textContent = postal || '—'
  els.selectionPostalToken.hidden = !postal
  els.selectionStatus.textContent = ''
  els.selectionStatus.hidden = true

  if (level === 'village') {
    els.selectionMetrics.innerHTML = `
      <div class="selection-empty-metrics">
        <span>🏡</span>
        <span>Tingkat Terendah (Desa/Kelurahan)</span>
      </div>
    `
  }
}

function setInlineStats(node, text) {
  if (!node) return
  const inlineStats = node.querySelector(':scope > .tree-row .tree-inline-stats')
  if (!inlineStats) return
  inlineStats.textContent = text
  node.classList.toggle('has-inline-stats', Boolean(text))
}

function renderScopedStats(item, stats, node) {
  setSelectionSummary(item, node)
  setInlineStats(node, scopedInlineText(item, stats))

  if (item.level === 'province') {
    els.selectionMetrics.innerHTML = [
      scopedMetric('Kab/Kota', stats.regencies, 'regency'),
      scopedMetric('Kecamatan', stats.districts, 'district'),
      scopedMetric('Desa/Kel', stats.villages, 'village'),
    ].join('')
    return
  }

  if (item.level === 'regency') {
    els.selectionMetrics.innerHTML = [
      scopedMetric('Kecamatan', stats.districts, 'district'),
      scopedMetric('Desa/Kel', stats.villages, 'village'),
    ].join('')
    return
  }

  if (item.level === 'district') {
    els.selectionMetrics.innerHTML = scopedMetric('Desa/Kel', stats.villages, 'village')
    return
  }

  if (item.level === 'village') {
    els.selectionMetrics.innerHTML = `
      <div class="selection-empty-metrics">
        <span>🏡</span>
        <span>Tingkat Terendah (Desa/Kelurahan)</span>
      </div>
    `
    return
  }

  els.selectionMetrics.innerHTML = ''
}

async function loadScopedStats(item, node, selectionId = null) {
  setInlineStats(node, 'Menghitung…')
  try {
    const stats = await request('/api/locations/stats', scopedStatsParams(item), true)
    if (selectionId !== null && selectionId !== state.mapSelectionId) return
    renderScopedStats(item, stats, node)
  } catch (err) {
    if (selectionId !== null && selectionId !== state.mapSelectionId) return
    setInlineStats(node, '')
    setSelectionSummary(item, node)
    els.selectionStatus.textContent = `Hitungan wilayah tidak tersedia: ${err.message}`
    els.selectionStatus.hidden = false
  }
}

// ── Health Check ──

async function checkHealth() {
  els.healthDot.className = 'status-dot'
  els.healthText.textContent = 'Memeriksa API…'
  try {
    await request('/healthz', {}, true)
    els.healthDot.className = 'status-dot ok'
    els.healthText.textContent = 'Service online'
  } catch {
    els.healthDot.className = 'status-dot fail'
    els.healthText.textContent = 'Service offline'
  }
}

// ── Tree View ──

const CHEVRON_SVG = '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>'

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
  node.locationItem = item
  node.dataset.code = item.full_code || item.code
  node.dataset.level = item.level
  node.dataset.name = item.name.toLowerCase()

  const row = document.createElement('div')
  row.className = 'tree-row'
  row.setAttribute('role', 'treeitem')
  row.tabIndex = 0

  const chevron = document.createElement('span')
  chevron.className = 'tree-chevron'
  chevron.innerHTML = CHEVRON_SVG

  const code = document.createElement('span')
  code.className = 'tree-code'
  code.textContent = item.code
  code.title = 'Klik untuk salin kode'
  code.setAttribute('role', 'button')
  code.setAttribute('aria-label', `Salin kode wilayah ${item.full_code || item.code}`)
  code.tabIndex = 0
  const copyCode = (e) => {
    e.stopPropagation()
    navigator.clipboard.writeText(item.full_code || item.code).then(() => showToast(`Kode disalin: ${item.full_code || item.code}`))
  }
  code.addEventListener('click', copyCode)
  code.addEventListener('keydown', (e) => {
    if (!isActivationKey(e)) return
    e.preventDefault()
    copyCode(e)
  })

  const name = document.createElement('span')
  name.className = 'tree-name'
  name.textContent = item.name

  const inlineStats = document.createElement('span')
  inlineStats.className = 'tree-inline-stats'

  const postalCode = document.createElement('span')
  postalCode.className = 'tree-postal-code'
  postalCode.textContent = item.postal_code || ''
  postalCode.title = item.postal_code ? `Kode pos ${item.postal_code}` : ''

  const badge = document.createElement('span')
  badge.className = `tree-badge tree-badge-${item.level}`
  badge.textContent = item.level

  row.append(chevron, code, name, inlineStats, postalCode, badge)
  node.appendChild(row)

  const selectRow = () => {
    selectLocation(item, node)
    if (!isLeaf) toggleNode(node, item)
  }
  row.addEventListener('click', selectRow)
  row.addEventListener('keydown', (e) => {
    if (!isActivationKey(e)) return
    e.preventDefault()
    selectRow()
  })

  if (!isLeaf) {
    row.setAttribute('aria-expanded', 'false')
    const children = document.createElement('div')
    children.className = 'tree-children'
    children.setAttribute('role', 'group')
    node.appendChild(children)
  }

  return node
}

function updateTreePostalCode(node, value) {
  if (!node) return
  const postalCode = node.querySelector(':scope > .tree-row .tree-postal-code')
  if (!postalCode) return
  postalCode.textContent = value || ''
  postalCode.title = value ? `Kode pos ${value}` : ''
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

  if (children.dataset.loaded) return

  const loading = document.createElement('div')
  loading.className = 'tree-loading'
  loading.textContent = 'Memuat wilayah turunan…'
  children.appendChild(loading)
  children.dataset.loaded = '1'

  try {
    const items = await fetchChildren(item)
    children.removeChild(loading)
    if (!items.length) {
      const empty = document.createElement('div')
      empty.className = 'tree-empty'
      empty.textContent = 'Tidak ada data'
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
  loading.textContent = 'Memuat daftar provinsi…'
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

// ── Breadcrumb ──

function breadcrumbPath(item, node) {
  const path = []
  let current = node
  while (current?.classList.contains('tree-node')) {
    path.unshift({ item: current.locationItem, node: current })
    current = current.parentElement?.closest('.tree-node')
  }

  const itemCode = item.full_code || item.code
  const last = path[path.length - 1]
  const lastCode = last?.item && (last.item.full_code || last.item.code)
  if (!last || lastCode !== itemCode) {
    path.push({ item, node })
  } else {
    last.item = item
  }
  return path
}

function renderBreadcrumb(item = null, node = null) {
  els.breadcrumb.innerHTML = ''
  if (!item) return

  breadcrumbPath(item, node).forEach((entry, index, path) => {
    if (index > 0) {
      const separator = document.createElement('span')
      separator.className = 'breadcrumb-sep'
      separator.textContent = '›'
      separator.setAttribute('aria-hidden', 'true')
      els.breadcrumb.appendChild(separator)
    }

    const wrapper = document.createElement('span')
    wrapper.className = 'breadcrumb-item'
    const button = document.createElement('button')
    button.type = 'button'
    button.textContent = entry.item.name || entry.item.code

    if (index === path.length - 1) {
      wrapper.classList.add('current')
      button.setAttribute('aria-current', 'location')
    } else {
      button.addEventListener('click', () => {
        selectLocation(entry.item, entry.node)
        scrollTreeNodeIntoView(entry.node)
      })
    }

    wrapper.appendChild(button)
    els.breadcrumb.appendChild(wrapper)
  })
}

function scrollTreeNodeIntoView(node) {
  const row = node?.querySelector(':scope > .tree-row')
  row?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
}

// ── Search View ──

function renderSearchRows(tbody, items) {
  tbody.innerHTML = ''
  if (!items.length) {
    const row = document.createElement('tr')
    row.className = 'empty-row'
    const cell = document.createElement('td')
    cell.colSpan = 7
    cell.textContent = 'Tidak ada wilayah yang sesuai dengan pencarian.'
    row.appendChild(cell)
    tbody.appendChild(row)
    return
  }
  items.forEach((item) => {
    const row = document.createElement('tr')
    row.className = 'search-row'
    ;['code', 'full_code', 'name', 'level', 'parent_code', 'postal_code'].forEach((col, idx) => {
      const cell = document.createElement('td')
      const value = item[col] || '-'
      if (idx === 0 && value !== '-') {
        cell.className = 'code-cell'
        cell.title = 'Klik untuk salin kode'
        cell.addEventListener('click', (e) => {
          e.stopPropagation()
          navigator.clipboard.writeText(value).then(() => showToast(`Kode disalin: ${value}`))
        })
        cell.textContent = value
      } else if (col === 'level' && value !== '-') {
        const badge = document.createElement('span')
        badge.className = `level-badge level-${value}`
        badge.textContent = value
        cell.appendChild(badge)
      } else if (col === 'postal_code' && value !== '-') {
        cell.className = 'postal-code-cell'
        cell.textContent = value
      } else {
        cell.textContent = value
      }
      row.appendChild(cell)
    })
    const action = document.createElement('td')
    action.className = 'action-cell'
    action.innerHTML = `<button class="browse-btn" title="Buka dan petakan wilayah ini"><svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="3 6 9 3 15 6 21 3 21 18 15 21 9 18 3 21"/><line x1="9" x2="9" y1="3" y2="18"/><line x1="15" x2="15" y1="6" y2="21"/></svg> Buka</button>`
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
  selectLocation(item)
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
      await new Promise(r => setTimeout(r, 600))
    }
    parentNode = node.querySelector(':scope > .tree-children')
    if (!parentNode) break
  }

  const target = document.querySelector(`.tree-node[data-code="${fc}"]`)
  if (target) {
    target.querySelector('.tree-row').style.background = 'var(--accent-bg)'
    scrollTreeNodeIntoView(target)
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
      bar.style.width = idx === 0 ? '60px' : `${50 + Math.random() * 40}%`
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
  cell.innerHTML = `⚠️ ${escapeHTML(message)}`
  row.appendChild(cell)
  tbody.appendChild(row)
}

async function runSearch() {
  const q = els.searchInput.value.trim()
  if (!q) { showToast('Ketik kata kunci pencarian'); return }
  setTableLoading(els.searchRows, ['', '', '', '', '', '', ''])
  els.searchMeta.textContent = 'Mencari wilayah…'
  try {
    const rows = await request('/api/locations/search', { q, limit: els.searchLimit.value || 25 })
    renderSearchRows(els.searchRows, rows)
    const isMobile = window.innerWidth <= 768
    els.searchMeta.textContent = `${rows.length} hasil ditemukan${isMobile ? ' — sentuh baris untuk membuka peta' : ''}`
  } catch (err) {
    els.searchMeta.textContent = 'Pencarian gagal'
    setTableError(els.searchRows, ['', '', '', '', '', '', ''], err.message)
    showToast(err.message)
  }
}

// ── Tab Navigation ──

function switchTab(tab) {
  state.activeTab = tab
  els.tabs.forEach((button) => {
    const active = button.dataset.tab === tab
    button.classList.toggle('active', active)
    button.setAttribute('aria-selected', String(active))
  })
  Object.entries(els.views).forEach(([k, v]) => {
    const active = k === tab
    v.classList.toggle('active', active)
    v.hidden = !active
  })
  if (tab === 'browse') {
    els.viewTitle.textContent = 'Eksplorasi Wilayah'
    els.viewSubtitle.textContent = 'Pilih provinsi dan telusuri hingga tingkat desa/kelurahan serta koordinat peta.'
  }
  if (tab === 'search') {
    els.viewTitle.textContent = 'Pencarian Global'
    els.viewSubtitle.textContent = 'Cari nama wilayah di seluruh tingkatan administratif Indonesia.'
  }
  if (tab === 'browse' && state.map) {
    requestAnimationFrame(() => state.map.invalidateSize())
  }
}

// ── Sidebar Controls ──

function isMobileSidebar() {
  return window.matchMedia('(max-width: 768px)').matches
}

function setSidebarState(open) {
  if (isMobileSidebar()) {
    els.appShell.classList.remove('sidebar-collapsed')
    els.sidebar.classList.toggle('open', open)
    els.sidebarOverlay.classList.toggle('show', open)
  } else {
    els.sidebar.classList.remove('open')
    els.appShell.classList.toggle('sidebar-collapsed', !open)
    els.sidebarOverlay.classList.remove('show')
  }
  els.openSidebar.setAttribute('aria-expanded', String(open))
  els.openSidebar.setAttribute('aria-label', open ? 'Tutup sidebar' : 'Buka sidebar')
  els.collapseSidebar.setAttribute('aria-expanded', String(open))
  const collapseLabel = isMobileSidebar()
    ? (open ? 'Tutup Sidebar' : 'Buka Sidebar')
    : (open ? 'Ciutkan Sidebar' : 'Perluas Sidebar')
  els.collapseSidebar.setAttribute('aria-label', collapseLabel)
  const collapseText = els.collapseSidebar.querySelector('span')
  if (collapseText) collapseText.textContent = collapseLabel
}

function openSidebar() { setSidebarState(true) }
function closeSidebar() { setSidebarState(false) }
function toggleSidebar() {
  const open = isMobileSidebar()
    ? els.sidebar.classList.contains('open')
    : !els.appShell.classList.contains('sidebar-collapsed')
  setSidebarState(!open)
}

// ── Event Bindings ──

function bindEvents() {
  // Theme Toggle
  if (els.themeToggle) {
    els.themeToggle.addEventListener('click', toggleTheme)
  }

  els.apiBaseUrl.value = state.apiBaseUrl
  els.apiBaseUrl.placeholder = DEFAULT_API_BASE_URL
  els.resetApiUrl.addEventListener('click', () => {
    state.apiBaseUrl = DEFAULT_API_BASE_URL
    els.apiBaseUrl.value = DEFAULT_API_BASE_URL
    localStorage.removeItem('location-service-api-base-url')
    checkHealth(); refreshData()
  })

  els.openSidebar.addEventListener('click', toggleSidebar)
  els.collapseSidebar.addEventListener('click', toggleSidebar)
  els.sidebarOverlay.addEventListener('click', closeSidebar)

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeSidebar()
      if (els.responseDrawer.classList.contains('open')) {
        setResponseDrawerState(false)
      }
    }
    // Shortcut '/' for quick search focus
    if (e.key === '/' && document.activeElement !== els.quickSearch && document.activeElement !== els.searchInput) {
      e.preventDefault()
      if (state.activeTab === 'browse') {
        els.quickSearch.focus()
      } else {
        els.searchInput.focus()
      }
    }
  })

  els.apiBaseUrl.addEventListener('change', () => {
    state.apiBaseUrl = els.apiBaseUrl.value.trim() || DEFAULT_API_BASE_URL
    localStorage.setItem('location-service-api-base-url', state.apiBaseUrl)
    checkHealth(); refreshData()
  })

  els.refreshHealth.addEventListener('click', checkHealth)
  els.tabs.forEach(b => b.addEventListener('click', () => {
    switchTab(b.dataset.tab)
    if (isMobileSidebar()) closeSidebar()
  }))

  document.querySelectorAll('[data-mobile-panel]').forEach((button) => {
    button.addEventListener('click', () => {
      const panel = button.dataset.mobilePanel
      document.querySelectorAll('[data-mobile-panel]').forEach((tab) => {
        const active = tab === button
        tab.classList.toggle('active', active)
        tab.setAttribute('aria-selected', String(active))
      })
      const contentGrid = document.querySelector('.content-grid')
      contentGrid?.classList.toggle('mobile-show-map', panel === 'map')
      if (panel === 'map' && state.map) {
        requestAnimationFrame(() => state.map.invalidateSize())
      }
    })
  })

  els.shortCodeToggle.addEventListener('change', () => {
    state.shortCodes = els.shortCodeToggle.checked
    loadTree()
  })

  els.responseDrawerToggle.addEventListener('click', () => {
    setResponseDrawerState(!els.responseDrawer.classList.contains('open'))
  })

  els.reloadData.addEventListener('click', () => {
    els.reloadData.classList.add('loading')
    refreshData().finally(() => els.reloadData.classList.remove('loading'))
  })

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
    showToast('Tampilan berhasil direset')
  })

  let qst
  els.quickSearch.addEventListener('input', () => {
    clearTimeout(qst)
    const q = els.quickSearch.value.trim()
    if (!q) return
    qst = setTimeout(() => { els.searchInput.value = q; switchTab('search'); runSearch() }, 350)
  })
  els.quickSearch.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      clearTimeout(qst)
      const q = els.quickSearch.value.trim()
      if (!q) return
      els.searchInput.value = q
      switchTab('search')
      runSearch()
    }
  })

  els.treeFilter.addEventListener('input', filterTree)
  els.runSearch.addEventListener('click', runSearch)
  els.searchInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') runSearch() })
  els.searchLimit.addEventListener('keydown', (e) => { if (e.key === 'Enter') runSearch() })

  els.copyEndpoint.addEventListener('click', async () => {
    await navigator.clipboard.writeText(els.responseMethod.href)
    showToast('URL Endpoint berhasil disalin')
  })

  els.copyResponse.addEventListener('click', async () => {
    await navigator.clipboard.writeText(JSON.stringify(state.lastResponse, null, 2))
    showToast('JSON output berhasil disalin')
  })

  // Selection Card Actions
  const copySelectionBtn = document.getElementById('copySelectionCodeBtn')
  if (copySelectionBtn) {
    copySelectionBtn.addEventListener('click', () => {
      const code = els.selectionCode.textContent.trim()
      if (code && code !== '—') {
        navigator.clipboard.writeText(code).then(() => showToast(`Kode wilayah disalin: ${code}`))
      }
    })
  }

  const closeSelectionBtn = document.getElementById('closeSelectionBtn')
  if (closeSelectionBtn) {
    closeSelectionBtn.addEventListener('click', () => {
      resetSelectionSummary()
      resetMap()
      showToast('Seleksi wilayah ditutup')
    })
  }

  setSidebarState(!isMobileSidebar())
}

// ── App Initialization ──

async function init() {
  applyTheme(state.currentTheme)
  bindEvents()
  switchTab('browse')
  resetSelectionSummary()
  await checkHealth()
  await refreshData()
}

init()
