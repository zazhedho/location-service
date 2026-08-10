// Copy code buttons
const copyButtons = []
document.querySelectorAll('pre').forEach((block) => {
  const code = block.querySelector('code')?.textContent || ''
  const isLongBlock = code.split('\n').length > 3 || code.length > 120
  if (!isLongBlock) return

  const button = document.createElement('button')
  button.className = 'copy-code'
  button.type = 'button'
  button.textContent = 'Salin'
  button.setAttribute('aria-label', 'Salin contoh kode')

  button.addEventListener('click', async () => {
    await navigator.clipboard.writeText(code)
    button.textContent = docsLanguage === 'id' ? 'Tersalin' : 'Copied!'
    clearTimeout(button.t)
    button.t = setTimeout(() => { button.textContent = docsLanguage === 'id' ? 'Salin' : 'Copy' }, 1600)
  })

  block.appendChild(button)
  copyButtons.push(button)
})

const curlButtons = []
document.querySelectorAll('[data-curl-path]').forEach((button) => {
  const curl = `curl '${new URL(button.dataset.curlPath, window.location.origin).href}'`
  button.addEventListener('click', async () => {
    await navigator.clipboard.writeText(curl)
    button.textContent = docsLanguage === 'id' ? 'Tersalin' : 'Copied!'
    clearTimeout(button.t)
    button.t = setTimeout(() => { button.textContent = docsLanguage === 'id' ? 'Salin cURL' : 'Copy cURL' }, 1600)
  })
  curlButtons.push(button)
})

const languageTabs = Array.from(document.querySelectorAll('[data-example-tab]'))
const languagePanels = Array.from(document.querySelectorAll('[data-example-panel]'))

function activateExample(name) {
  languageTabs.forEach((tab) => {
    const active = tab.dataset.exampleTab === name
    tab.classList.toggle('active', active)
    tab.setAttribute('aria-selected', String(active))
  })

  languagePanels.forEach((panel) => {
    const active = panel.dataset.examplePanel === name
    panel.classList.toggle('active', active)
    panel.hidden = !active
  })
}

languageTabs.forEach((tab) => {
  tab.addEventListener('click', () => activateExample(tab.dataset.exampleTab))
  tab.addEventListener('keydown', (event) => {
    const currentIndex = languageTabs.indexOf(tab)
    let nextIndex = currentIndex

    if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % languageTabs.length
    if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + languageTabs.length) % languageTabs.length
    if (event.key === 'Home') nextIndex = 0
    if (event.key === 'End') nextIndex = languageTabs.length - 1
    if (nextIndex === currentIndex) return

    event.preventDefault()
    languageTabs[nextIndex].focus()
    activateExample(languageTabs[nextIndex].dataset.exampleTab)
  })
})

const translations = {
  id: {
    brandAria: 'Kembali ke aplikasi Location Service', navAria: 'Navigasi utama', languageAria: 'Bahasa',
    brandSubtitle: 'API Wilayah Indonesia', navTry: 'Coba API', navEndpoints: 'Endpoint', navExamples: 'Contoh',
    heroEyebrow: 'Dokumentasi', heroTitle: 'API wilayah Indonesia untuk formulir alamat dan pencarian lokasi.',
    heroLead: 'Gunakan API ini untuk menampilkan provinsi, kabupaten/kota, kecamatan, dan kelurahan/desa secara berjenjang (cascading).',
    baseUrl: 'Base URL', hierarchyTitle: 'Struktur Data Wilayah',
    hierarchyCopy: 'Data wilayah disusun secara bertingkat. Anda memerlukan <strong>kode area sebelumnya</strong> untuk mencari area di bawahnya.',
    province: 'Provinsi', prefix: 'Awalan', regencyCity: 'Kab/Kota', needsProvinceCode: 'Butuh kode Provinsi',
    district: 'Kecamatan', needsRegencyCode: 'Butuh kode Kab/Kota', village: 'Desa/Kelurahan', needsDistrictCode: 'Butuh kode Kecamatan',
    demoTitle: 'Coba Langsung', demoCopy: 'Simulasikan form alamat dengan API ini. Pilih tiap tingkat wilayah untuk melihat request berjenjang secara otomatis.',
    demoProvinceLabel: '1. Pilih Provinsi', demoLoadingProvinces: 'Memuat data provinsi...', demoRegencyLabel: '2. Pilih Kabupaten / Kota',
    demoDistrictLabel: '3. Pilih Kecamatan', demoVillageLabel: '4. Pilih Desa / Kelurahan',
    demoSelectProvinceFirst: 'Pilih provinsi terlebih dahulu', demoSelectRegencyFirst: 'Pilih kabupaten/kota terlebih dahulu', demoSelectDistrictFirst: 'Pilih kecamatan terlebih dahulu',
    demoWaiting: 'Menunggu aksi...', demoResponsePlaceholder: '// Hasil respons API (JSON) akan tampil di sini',
    quickStartTitle: 'Mulai di Sini', stepLoadProvincesTitle: 'Muat provinsi', stepLoadProvincesCopy: 'Gunakan saat halaman dimuat untuk mengisi dropdown pertama.',
    stepLoadChildrenTitle: 'Muat wilayah turunan', stepLoadChildrenCopy: 'Saat pengguna memilih provinsi, gunakan kodenya untuk memuat kabupaten/kota, lalu ulangi untuk kecamatan dan desa.',
    stepInspectTitle: 'Periksa lokasi', stepInspectCopy: 'Gunakan endpoint detail untuk koordinat, lalu minta boundary saat ingin menggambarkannya di peta.',
    stepSaveTitle: 'Simpan nilai stabil', stepSaveCopy: 'Simpan <code>full_code</code> dan <code>name</code> di sistem Anda untuk referensi alamat yang konsisten.',
    endpointsTitle: 'Endpoint', endpointsCopy: 'Semua endpoint mengembalikan JSON. Tambahkan <code>code_format=short</code> jika hanya memerlukan kode level turunan.',
    endpointsNote: 'Ganti placeholder seperti <code>{province_code}</code> dengan nilai dari endpoint sebelumnya.', endpointHealth: 'Pemeriksaan kesehatan',
    endpointStats: 'Total dan hitungan berdasarkan wilayah', endpointStatsNote: 'Opsional: <code>province_code={province_code}</code>, <code>regency_code={regency_code}</code>, <code>district_code={district_code}</code>',
    endpointProvinces: 'Provinsi', endpointRegencies: 'Kabupaten / kota', endpointDistricts: 'Kecamatan', endpointVillages: 'Desa / kelurahan', endpointSearch: 'Cari berdasarkan nama',
    endpointDetail: 'Detail lokasi', endpointDetailNote: 'Mengembalikan hierarki, koordinat jika tersedia, dan <code>has_boundary</code>.',
    endpointBoundary: 'Batas wilayah lokasi', endpointBoundaryNote: 'Mengembalikan <code>leaflet_path</code> dalam urutan Leaflet <code>[latitude, longitude]</code>. Batas yang tidak tersedia mengembalikan <code>404</code>.',
    endpointIslands: 'Daftar pulau', endpointIslandsNote: '<code>province_code</code> bersifat opsional. Paginasi default halaman <code>1</code>, limit <code>50</code>, maksimum <code>500</code>.',
    endpointIslandDetail: 'Detail pulau', endpointIslandDetailNote: 'Gunakan kode pulau seperti <code>11.01.40001</code>.',
    endpointPopulation: 'Penduduk', endpointPopulationNote: 'Mengembalikan <code>male</code>, <code>female</code>, <code>total</code>, sumber, tanggal referensi, dan waktu impor.',
    endpointArea: 'Luas wilayah', endpointAreaNote: 'Mengembalikan <code>area_km2</code>, sumber, tanggal referensi, dan waktu impor.',
    responseTitle: 'Format Respons', responseCopy: 'Respons berhasil menggunakan envelope yang sama. Data dapat berupa object atau array, tergantung endpoint.',
    examplesTitle: 'Contoh Implementasi', examplesAria: 'Contoh bahasa pemrograman', laravelNote: 'Memerlukan HTTP Client Laravel.', pythonNote: 'Menggunakan library standar Python.',
    codeFormatTitle: 'Format Kode', codeFormatCopy: 'Gunakan full code saat menyimpan data. Short code hanya untuk tampilan UI atau form lama.',
    tableLevel: 'Level', tableFullCode: 'Full code', tableShortCode: 'Short code', tableProvince: 'Provinsi', tableRegency: 'Kabupaten/kota', tableDistrict: 'Kecamatan', tableVillage: 'Desa/kelurahan'
  },
  en: {
    brandAria: 'Back to the Location Service app', navAria: 'Main navigation', languageAria: 'Language',
    brandSubtitle: 'Indonesian Regional API', navTry: 'Try API', navEndpoints: 'Endpoints', navExamples: 'Examples',
    heroEyebrow: 'Documentation', heroTitle: 'Indonesian location API for address forms and location search.',
    heroLead: 'Use this API to load provinces, regencies/cities, districts, and villages in a cascading address form.',
    baseUrl: 'Base URL', hierarchyTitle: 'Location Data Hierarchy',
    hierarchyCopy: 'Locations are organized hierarchically. You need the <strong>previous area code</strong> to find its child areas.',
    province: 'Province', prefix: 'Starting level', regencyCity: 'Regency/City', needsProvinceCode: 'Needs province code',
    district: 'District', needsRegencyCode: 'Needs regency code', village: 'Village', needsDistrictCode: 'Needs district code',
    demoTitle: 'Interactive Demo', demoCopy: 'Simulate an address form with this API. Select each location level to see the cascading requests.',
    demoProvinceLabel: '1. Select Province', demoLoadingProvinces: 'Loading provinces...', demoRegencyLabel: '2. Select Regency / City',
    demoDistrictLabel: '3. Select District', demoVillageLabel: '4. Select Village',
    demoSelectProvinceFirst: 'Select a province first', demoSelectRegencyFirst: 'Select a regency/city first', demoSelectDistrictFirst: 'Select a district first',
    demoWaiting: 'Waiting for an action...', demoResponsePlaceholder: '// API response (JSON) appears here',
    quickStartTitle: 'Start Here', stepLoadProvincesTitle: 'Load provinces', stepLoadProvincesCopy: 'Use this on page load for your first dropdown.',
    stepLoadChildrenTitle: 'Load child areas', stepLoadChildrenCopy: 'When a user selects a province, use its code to load regencies/cities, then repeat for districts and villages.',
    stepInspectTitle: 'Inspect a location', stepInspectCopy: 'Use the detail endpoint for coordinates, then request its boundary when you need to draw it on a map.',
    stepSaveTitle: 'Save stable values', stepSaveCopy: 'Store <code>full_code</code> and <code>name</code> in your system for reliable address references.',
    endpointsTitle: 'Endpoints', endpointsCopy: 'All endpoints return JSON. Add <code>code_format=short</code> when you only need child-level codes.',
    endpointsNote: 'Replace placeholders like <code>{province_code}</code> with values returned from the previous endpoint.', endpointHealth: 'Health check',
    endpointStats: 'Total and scoped counts', endpointStatsNote: 'Optional: <code>province_code={province_code}</code>, <code>regency_code={regency_code}</code>, <code>district_code={district_code}</code>',
    endpointProvinces: 'Provinces', endpointRegencies: 'Regencies / cities', endpointDistricts: 'Districts', endpointVillages: 'Villages', endpointSearch: 'Search by name',
    endpointDetail: 'Location detail', endpointDetailNote: 'Returns hierarchy data, coordinates when available, and <code>has_boundary</code>.',
    endpointBoundary: 'Location boundary', endpointBoundaryNote: 'Returns <code>leaflet_path</code> in Leaflet <code>[latitude, longitude]</code> order. A missing boundary returns <code>404</code>.',
    endpointIslands: 'Islands list', endpointIslandsNote: '<code>province_code</code> is optional. Pagination defaults to page <code>1</code>, limit <code>50</code>, and caps at <code>500</code>.',
    endpointIslandDetail: 'Island detail', endpointIslandDetailNote: 'Use an island code such as <code>11.01.40001</code>.',
    endpointPopulation: 'Population', endpointPopulationNote: 'Returns <code>male</code>, <code>female</code>, <code>total</code>, source, reference date, and import timestamp.',
    endpointArea: 'Area', endpointAreaNote: 'Returns <code>area_km2</code>, source, reference date, and import timestamp.',
    responseTitle: 'Response Format', responseCopy: 'Successful responses use the same envelope. Data can be an object or an array depending on the endpoint.',
    examplesTitle: 'Implementation Examples', examplesAria: 'Programming language examples', laravelNote: 'Requires Laravel HTTP Client.', pythonNote: 'Uses the Python standard library.',
    codeFormatTitle: 'Code Format', codeFormatCopy: 'Use full codes when saving data. Short codes are useful only for UI display or legacy forms.',
    tableLevel: 'Level', tableFullCode: 'Full code', tableShortCode: 'Short code', tableProvince: 'Province', tableRegency: 'Regency/city', tableDistrict: 'District', tableVillage: 'Village'
  }
}

const demoCopy = {
  id: {
    endpointProvinces: 'GET /api/locations/provinces', endpointRegencies: (code) => `GET /api/locations/regencies?province_code=${code}`,
    endpointDistricts: (code) => `GET /api/locations/districts?regency_code=${code}`, endpointVillages: (code) => `GET /api/locations/villages?district_code=${code}`,
    loadingProvinces: 'Memuat data provinsi...', loadingRegencies: 'Memuat data kabupaten/kota...', loadingDistricts: 'Memuat data kecamatan...', loadingVillages: 'Memuat data desa/kelurahan...',
    chooseProvince: '-- Pilih Provinsi --', chooseRegency: '-- Pilih Kabupaten / Kota --', chooseDistrict: '-- Pilih Kecamatan --', chooseVillage: '-- Pilih Desa / Kelurahan --',
    selectProvinceFirst: 'Pilih provinsi terlebih dahulu', selectRegencyFirst: 'Pilih kabupaten/kota terlebih dahulu', selectDistrictFirst: 'Pilih kecamatan terlebih dahulu', waiting: 'Menunggu aksi...',
    responsePlaceholder: '// Hasil respons API (JSON) akan tampil di sini', error: 'Error: '
  },
  en: {
    endpointProvinces: 'GET /api/locations/provinces', endpointRegencies: (code) => `GET /api/locations/regencies?province_code=${code}`,
    endpointDistricts: (code) => `GET /api/locations/districts?regency_code=${code}`, endpointVillages: (code) => `GET /api/locations/villages?district_code=${code}`,
    loadingProvinces: 'Loading provinces...', loadingRegencies: 'Loading regencies/cities...', loadingDistricts: 'Loading districts...', loadingVillages: 'Loading villages...',
    chooseProvince: '-- Select Province --', chooseRegency: '-- Select Regency / City --', chooseDistrict: '-- Select District --', chooseVillage: '-- Select Village --',
    selectProvinceFirst: 'Select a province first', selectRegencyFirst: 'Select a regency/city first', selectDistrictFirst: 'Select a district first', waiting: 'Waiting for an action...',
    responsePlaceholder: '// API response (JSON) appears here', error: 'Error: '
  }
}

const languageButtons = Array.from(document.querySelectorAll('[data-doc-language]'))
const demoProvince = document.getElementById('demo-province')
const demoRegency = document.getElementById('demo-regency')
const demoDistrict = document.getElementById('demo-district')
const demoVillage = document.getElementById('demo-village')
const demoResponse = document.getElementById('demo-response')
const demoEndpointUrl = document.getElementById('demo-endpoint-url')
const demoBaseUrl = window.location.origin
let docsLanguage = 'id'
let demoRequestId = 0

try {
  docsLanguage = localStorage.getItem('location-service-docs-language') === 'en' ? 'en' : 'id'
} catch (_) {}

function copyFor(key) {
  return translations[docsLanguage][key] || key
}

function refreshDemoLanguage() {
  if (!demoProvince || !demoRegency || !demoDistrict || !demoVillage) return
  if (demoProvince.options[0]) {
    demoProvince.options[0].textContent = demoProvince.options.length > 1
      ? demoCopy[docsLanguage].chooseProvince
      : demoCopy[docsLanguage].loadingProvinces
  }
  if (demoRegency.options[0]) {
    demoRegency.options[0].textContent = demoRegency.options.length > 1
      ? demoCopy[docsLanguage].chooseRegency
      : demoCopy[docsLanguage].selectProvinceFirst
  }
  if (demoDistrict.options[0]) {
    demoDistrict.options[0].textContent = demoDistrict.options.length > 1
      ? demoCopy[docsLanguage].chooseDistrict
      : demoCopy[docsLanguage].selectRegencyFirst
  }
  if (demoVillage.options[0]) {
    demoVillage.options[0].textContent = demoVillage.options.length > 1
      ? demoCopy[docsLanguage].chooseVillage
      : demoCopy[docsLanguage].selectDistrictFirst
  }
  const responseText = demoResponse?.textContent.trim() || ''
  if (!demoProvince.value && demoResponse && !responseText.startsWith('{') && !responseText.startsWith('[') && !responseText.startsWith('Error:')) {
    demoResponse.textContent = demoCopy[docsLanguage].responsePlaceholder
  }
}

function applyLanguage(language) {
  docsLanguage = translations[language] ? language : 'id'
  document.documentElement.lang = docsLanguage
  document.querySelectorAll('[data-i18n]').forEach((element) => {
    element.textContent = copyFor(element.dataset.i18n)
  })
  document.querySelectorAll('[data-i18n-html]').forEach((element) => {
    element.innerHTML = copyFor(element.dataset.i18nHtml)
  })
  document.querySelectorAll('[data-i18n-aria]').forEach((element) => {
    element.setAttribute('aria-label', copyFor(element.dataset.i18nAria))
  })
  languageButtons.forEach((button) => {
    const active = button.dataset.docLanguage === docsLanguage
    button.classList.toggle('active', active)
    button.setAttribute('aria-pressed', String(active))
  })
  copyButtons.forEach((button) => {
    button.textContent = docsLanguage === 'id' ? 'Salin' : 'Copy'
    button.setAttribute('aria-label', docsLanguage === 'id' ? 'Salin contoh kode' : 'Copy code example')
  })
  curlButtons.forEach((button) => {
    button.textContent = docsLanguage === 'id' ? 'Salin cURL' : 'Copy cURL'
    button.setAttribute('aria-label', docsLanguage === 'id' ? 'Salin perintah cURL' : 'Copy cURL command')
  })
  try { localStorage.setItem('location-service-docs-language', docsLanguage) } catch (_) {}
  refreshDemoLanguage()
}

languageButtons.forEach((button) => {
  button.addEventListener('click', () => applyLanguage(button.dataset.docLanguage))
})

async function requestDemo(path) {
  const response = await fetch(`${demoBaseUrl}${path}`)
  const data = await response.json()
  if (!response.ok || !data.status) throw new Error(data.message || `HTTP ${response.status}`)
  return data
}

function fillSelect(select, placeholder, items) {
  select.replaceChildren(new Option(placeholder, ''))
  items.forEach((item) => select.appendChild(new Option(item.name, item.full_code)))
}

async function loadProvinces() {
  demoEndpointUrl.textContent = demoCopy[docsLanguage].endpointProvinces
  demoResponse.textContent = demoCopy[docsLanguage].loadingProvinces
  try {
    const data = await requestDemo('/api/locations/provinces')
    demoResponse.textContent = JSON.stringify(data, null, 2)
    fillSelect(demoProvince, demoCopy[docsLanguage].chooseProvince, data.data)
  } catch (error) {
    demoResponse.textContent = demoCopy[docsLanguage].error + error.message
  }
}

function resetDemoSelect(select, placeholder) {
  select.replaceChildren(new Option(placeholder, ''))
  select.disabled = true
}

async function loadRegencies(provinceCode) {
  const requestId = ++demoRequestId
  resetDemoSelect(demoDistrict, demoCopy[docsLanguage].selectRegencyFirst)
  resetDemoSelect(demoVillage, demoCopy[docsLanguage].selectDistrictFirst)
  if (!provinceCode) {
    resetDemoSelect(demoRegency, demoCopy[docsLanguage].selectProvinceFirst)
    demoEndpointUrl.textContent = demoCopy[docsLanguage].waiting
    demoResponse.textContent = demoCopy[docsLanguage].responsePlaceholder
    return
  }

  demoEndpointUrl.textContent = demoCopy[docsLanguage].endpointRegencies(encodeURIComponent(provinceCode))
  demoResponse.textContent = demoCopy[docsLanguage].loadingRegencies
  demoRegency.disabled = true
  try {
    const data = await requestDemo(`/api/locations/regencies?province_code=${encodeURIComponent(provinceCode)}`)
    if (requestId !== demoRequestId) return
    demoResponse.textContent = JSON.stringify(data, null, 2)
    fillSelect(demoRegency, demoCopy[docsLanguage].chooseRegency, data.data)
    demoRegency.disabled = false
  } catch (error) {
    if (requestId !== demoRequestId) return
    demoResponse.textContent = demoCopy[docsLanguage].error + error.message
  }
}

async function loadDistricts(regencyCode) {
  const requestId = ++demoRequestId
  resetDemoSelect(demoVillage, demoCopy[docsLanguage].selectDistrictFirst)
  if (!regencyCode) {
    resetDemoSelect(demoDistrict, demoCopy[docsLanguage].selectRegencyFirst)
    demoEndpointUrl.textContent = demoCopy[docsLanguage].waiting
    demoResponse.textContent = demoCopy[docsLanguage].responsePlaceholder
    return
  }

  demoEndpointUrl.textContent = demoCopy[docsLanguage].endpointDistricts(encodeURIComponent(regencyCode))
  demoResponse.textContent = demoCopy[docsLanguage].loadingDistricts
  try {
    const data = await requestDemo(`/api/locations/districts?regency_code=${encodeURIComponent(regencyCode)}`)
    if (requestId !== demoRequestId) return
    demoResponse.textContent = JSON.stringify(data, null, 2)
    fillSelect(demoDistrict, demoCopy[docsLanguage].chooseDistrict, data.data)
    demoDistrict.disabled = false
  } catch (error) {
    if (requestId !== demoRequestId) return
    demoResponse.textContent = demoCopy[docsLanguage].error + error.message
  }
}

async function loadVillages(districtCode) {
  const requestId = ++demoRequestId
  if (!districtCode) {
    resetDemoSelect(demoVillage, demoCopy[docsLanguage].selectDistrictFirst)
    demoEndpointUrl.textContent = demoCopy[docsLanguage].waiting
    demoResponse.textContent = demoCopy[docsLanguage].responsePlaceholder
    return
  }

  demoEndpointUrl.textContent = demoCopy[docsLanguage].endpointVillages(encodeURIComponent(districtCode))
  demoResponse.textContent = demoCopy[docsLanguage].loadingVillages
  try {
    const data = await requestDemo(`/api/locations/villages?district_code=${encodeURIComponent(districtCode)}`)
    if (requestId !== demoRequestId) return
    demoResponse.textContent = JSON.stringify(data, null, 2)
    fillSelect(demoVillage, demoCopy[docsLanguage].chooseVillage, data.data)
    demoVillage.disabled = false
  } catch (error) {
    if (requestId !== demoRequestId) return
    demoResponse.textContent = demoCopy[docsLanguage].error + error.message
  }
}

applyLanguage(docsLanguage)

if (demoProvince) {
  demoProvince.addEventListener('change', (event) => loadRegencies(event.target.value))
  demoRegency.addEventListener('change', (event) => loadDistricts(event.target.value))
  demoDistrict.addEventListener('change', (event) => loadVillages(event.target.value))
  loadProvinces()
}
