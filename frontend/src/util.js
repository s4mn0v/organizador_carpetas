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

/* ---------- Búsqueda por palabras (misma regla que busqueda.go) ----------
   Coincide si el nombre tiene AL MENOS UNA de las palabras, en cualquier orden, sin tildes ni mayúsculas,
   o una palabra parecida (empieza igual, está dentro de otra, 1–2 letras distintas). */
export function palabras(s) {
  return (sinTildes(s).match(/\p{L}+|\p{N}+/gu) || [])
}
export function consulta(q) {
  return [...new Set(palabras(q))].filter((p) => p.length >= 2 || /\d/.test(p))
}
function distancia(a, b) {
  if (!a.length) return b.length
  let prev = Array.from({length: b.length + 1}, (_, j) => j)
  for (let i = 1; i <= a.length; i++) {
    const cur = [i]
    for (let j = 1; j <= b.length; j++) cur[j] = Math.min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + (a[i - 1] === b[j - 1] ? 0 : 1))
    prev = cur
  }
  return prev[b.length]
}
function parecido(q, t) {
  if (q === t) return 1
  if (t.startsWith(q)) return 0.9
  if (q.length >= 3 && t.includes(q)) return 0.75
  if (t.length >= 4 && q.startsWith(t)) return 0.6
  if (q.length < 4) return 0
  const tol = q.length >= 8 ? 2 : 1
  if (Math.abs(t.length - q.length) <= tol) { const d = distancia(q, t); if (d <= tol) return 0.5 - 0.1 * d }
  if (t.length > q.length) { const d = distancia(q, t.slice(0, q.length)); if (d <= tol) return 0.4 - 0.1 * d }
  return 0
}
// Devuelve {n: palabras encontradas, p: puntaje}; n = 0 si no coincide.
export function puntuar(qs, nombre) {
  const ts = palabras(nombre)
  let n = 0, p = 0
  for (const q of qs) {
    let mejor = 0
    for (const t of ts) { const x = parecido(q, t); if (x > mejor) { mejor = x; if (x === 1) break } }
    if (mejor > 0) { n++; p += mejor }
  }
  if (!n) return {n: 0, p: 0}
  if (n === qs.length && qs.length > 1) p += 0.5
  if (qs.length > 1 && ts.join(' ').includes(qs.join(' '))) p += 0.5
  return {n, p}
}
// Filtra y ordena una lista por parecido con el texto (clave = función que da el nombre).
export function filtrarPorTexto(lista, texto, clave = (x) => x.nombre) {
  const qs = consulta(texto)
  if (!qs.length) return lista
  return lista.map((x) => ({x, s: puntuar(qs, clave(x))})).filter((r) => r.s.n > 0)
    .sort((a, b) => b.s.n - a.s.n || b.s.p - a.s.p || clave(a.x).length - clave(b.x).length).map((r) => r.x)
}

/* ---------- Rutas virtuales: "<raíz>/<ruta>" ("" = Inicio) ---------- */
export const raizDeRuta = (r) => (r || '').split('/')[0]
export const relDe = (r) => (r && r.includes('/') ? r.slice(r.indexOf('/') + 1) : '')
export const esRaizRuta = (r) => !!r && !r.includes('/')
export const esPrincipalRuta = (r) => !!r && r.split('/').length === 2

/* ---------- Retención ---------- */
export const SIN_DEPURAR = 'No depurar'
export const UNIDADES = [
  {u: 'dias', uno: 'día', varios: 'días'},
  {u: 'semanas', uno: 'semana', varios: 'semanas'},
  {u: 'meses', uno: 'mes', varios: 'meses'},
  {u: 'anos', uno: 'año', varios: 'años'},
]
// "6 meses" -> {n: 6, u: 'meses'}; null si no es un tiempo.
export function leerRetencion(v) {
  const m = sinTildes((v || '').trim()).match(/^(\d+)\s*([a-z]+)$/)
  if (!m) return null
  const u = {d: 'dias', dia: 'dias', dias: 'dias', s: 'semanas', sem: 'semanas', semana: 'semanas', semanas: 'semanas',
    m: 'meses', mes: 'meses', meses: 'meses', a: 'anos', ano: 'anos', anos: 'anos'}[m[2]]
  return u ? {n: +m[1], u} : null
}
export function textoRetencion(n, u) {
  const x = UNIDADES.find((y) => y.u === u)
  return !n || !x ? '' : `${n} ${n === 1 ? x.uno : x.varios}`
}
