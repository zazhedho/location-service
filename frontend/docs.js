// Copy code buttons
document.querySelectorAll('pre').forEach((block) => {
  const code = block.querySelector('code')?.textContent || ''
  const isLongBlock = code.split('\n').length > 3 || code.length > 120
  if (!isLongBlock) return

  const button = document.createElement('button')
  button.className = 'copy-code'
  button.type = 'button'
  button.textContent = 'Copy'
  button.setAttribute('aria-label', 'Copy code example')

  button.addEventListener('click', async () => {
    await navigator.clipboard.writeText(code)
    button.textContent = 'Copied!'
    clearTimeout(button.t)
    button.t = setTimeout(() => { button.textContent = 'Copy' }, 1600)
  })

  block.appendChild(button)
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
