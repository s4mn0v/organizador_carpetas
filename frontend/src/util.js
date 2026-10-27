const fmtFecha = new Intl.DateTimeFormat('es-CO', {day: '2-digit', month: 'short', year: 'numeric'})
const fmtHora = new Intl.DateTimeFormat('es-CO', {hour: '2-digit', minute: '2-digit'})

export function fecha(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  return fmtFecha.format(d).replace('.', '')
}
export function fechaHora(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  return `${fecha(iso)}, ${fmtHora.format(d)}`
}
export function hace(iso) {
  if (!iso) return ''
  const dias = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000)
  if (dias < 1) return 'hoy'
  if (dias === 1) return 'ayer'
  if (dias < 31) return `hace ${dias} días`
  const meses = Math.floor(dias / 30.4)
  if (meses < 12) return `hace ${meses} ${meses === 1 ? 'mes' : 'meses'}`
  const anios = Math.floor(meses / 12)
  return `hace ${anios} ${anios === 1 ? 'año' : 'años'}`
}
export function tamano(b) {
  if (!b && b !== 0) return '—'
  if (b < 1024 * 1024) return `${Math.max(1, Math.round(b / 1024))} KB`
  return `${(b / 1024 / 1024).toLocaleString('es-CO', {maximumFractionDigits: 1})} MB`
}
export function sinTildes(s) {
  return (s || '').normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()
}
export function mensajeError(e) {
  if (!e) return 'Error desconocido'
  return typeof e === 'string' ? e : (e.message || String(e))
}
export function tamanoLargo(b) {
  if (!b) return '0 KB'
  if (b >= 1024 ** 3) return `${(b / 1024 ** 3).toLocaleString('es-CO', {maximumFractionDigits: 2})} GB`
  return tamano(b)
}

// Tipo de archivo por extensión: etiqueta, color e ícono (los usan el árbol, las tablas y el grafo).
const TIPOS = [
  {tipo: 'Power BI', color: '#d9a300', exts: ['.pbix', '.pbit', '.pbip']},
  {tipo: 'Excel', color: '#2e8b57', exts: ['.xlsx', '.xlsm', '.xls', '.xlsb', '.csv']},
  {tipo: 'Word', color: '#2f6fb3', exts: ['.docx', '.doc', '.rtf', '.odt']},
  {tipo: 'PowerPoint', color: '#d0632c', exts: ['.pptx', '.ppt']},
  {tipo: 'PDF', color: '#c8402b', exts: ['.pdf']},
  {tipo: 'Imagen', color: '#8a5cc2', exts: ['.png', '.jpg', '.jpeg', '.gif', '.bmp', '.svg', '.webp', '.tif', '.tiff']},
  {tipo: 'Plano / CAD', color: '#3a8f9e', exts: ['.dwg', '.dxf', '.dwf']},
  {tipo: 'Comprimido', color: '#6b6f76', exts: ['.zip', '.rar', '.7z']},
  {tipo: 'Texto', color: '#5f7380', exts: ['.txt', '.md', '.log', '.json', '.xml']},
  {tipo: 'Video', color: '#b5477a', exts: ['.mp4', '.avi', '.mov', '.wmv']},
  {tipo: 'Correo', color: '#3f7f9f', exts: ['.msg', '.eml']},
]
const POR_EXT = {}
for (const t of TIPOS) for (const e of t.exts) POR_EXT[e] = t
export const TIPOS_ARCHIVO = TIPOS
export function tipoArchivo(ext) {
  return POR_EXT[(ext || '').toLowerCase()] || {tipo: ext ? ext.slice(1).toUpperCase() : 'Archivo', color: '#8b98a3'}
}
export const COLOR_CARPETA = '#f2b33d'

// Ruta relativa ("a/b") como la ve Windows ("a\b").
export const rutaWin = (r) => (r || '').replaceAll('/', '\\')
export const padreDe = (r) => (r && r.includes('/') ? r.slice(0, r.lastIndexOf('/')) : '')
export const nombreDe = (r) => (r || '').split('/').pop()

const INVALIDOS = /[\\/:*?"<>|]/
export function errorNombre(n) {
  if (!n || !n.trim()) return 'Obligatorio'
  if (INVALIDOS.test(n)) return 'No puede tener \\ / : * ? " < > |'
  if (/[. ]$/.test(n)) return 'No puede terminar en punto o espacio'
  return ''
}
