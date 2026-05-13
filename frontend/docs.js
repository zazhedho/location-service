document.querySelectorAll('pre').forEach((block) => {
  const button = document.createElement('button')
  button.className = 'copy-code'
  button.type = 'button'
  button.textContent = 'Copy'
  button.setAttribute('aria-label', 'Copy code example')

  button.addEventListener('click', async () => {
    const code = block.querySelector('code')?.textContent || ''
    await navigator.clipboard.writeText(code)
    button.textContent = 'Copied'
    clearTimeout(button.resetTimer)
    button.resetTimer = setTimeout(() => {
      button.textContent = 'Copy'
    }, 1600)
  })

  block.appendChild(button)
})
