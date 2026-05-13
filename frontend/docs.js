// Copy code buttons
document.querySelectorAll('pre').forEach((block) => {
  const button = document.createElement('button')
  button.className = 'copy-code'
  button.type = 'button'
  button.textContent = 'Copy'
  button.setAttribute('aria-label', 'Copy code example')

  button.addEventListener('click', async () => {
    const code = block.querySelector('code')?.textContent || ''
    await navigator.clipboard.writeText(code)
    button.textContent = 'Copied!'
    clearTimeout(button.t)
    button.t = setTimeout(() => { button.textContent = 'Copy' }, 1600)
  })

  block.appendChild(button)
})

// TOC active state on scroll
const tocLinks = document.querySelectorAll('.toc a[href^="#"]')
const sections = Array.from(tocLinks).map(a => document.getElementById(a.getAttribute('href').slice(1))).filter(Boolean)

function updateToc() {
  let current = ''
  for (const section of sections) {
    if (section.getBoundingClientRect().top <= 120) current = section.id
  }
  tocLinks.forEach(a => {
    a.classList.toggle('active', a.getAttribute('href') === `#${current}`)
  })
}

window.addEventListener('scroll', updateToc, { passive: true })
updateToc()
