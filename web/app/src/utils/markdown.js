import { marked } from 'marked'
import DOMPurify from 'dompurify'

const escapeHtml = (value) => {
  if (value === null || value === undefined) {
    return ''
  }
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

const renderer = new marked.Renderer()

renderer.link = (tokenOrHref, title, text) => {
  const tokenObject = typeof tokenOrHref === 'object' && tokenOrHref !== null
    ? tokenOrHref
    : null
  const href = tokenObject ? tokenObject.href : tokenOrHref
  const resolvedTitle = tokenObject ? tokenObject.title : title
  const resolvedText = tokenObject ? tokenObject.text : text
  const url = escapeHtml(href || '')
  const titleAttribute = resolvedTitle ? ` title="${escapeHtml(resolvedTitle)}"` : ''
  const linkText = resolvedText || ''
  return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="text-blue-700 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 underline font-medium"${titleAttribute}>${linkText}</a>`
}

marked.use({
  renderer,
  breaks: true,
  gfm: true,
  headerIds: false,
  mangle: false
})

export const formatAnnouncementMessage = (message) => {
  if (!message) {
    return ''
  }
  const markdown = String(message)
  const html = marked.parse(markdown)
  return DOMPurify.sanitize(html, { ADD_ATTR: ['target', 'rel'] })
}