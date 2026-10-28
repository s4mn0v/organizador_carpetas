<script setup>
import {computed, nextTick, onMounted, reactive, ref, watch} from 'vue'
import Arbol from './components/Arbol.vue'
import Ficha from './components/Ficha.vue'
import Dialogo from './components/Dialogo.vue'
import Icono from './components/Icono.vue'
import Selector from './components/Selector.vue'
import Grafo from './components/Grafo.vue'
import Ayuda from './components/Ayuda.vue'
import Menu from './components/Menu.vue'
import Duplicados from './components/Duplicados.vue'
import Vencidos from './components/Vencidos.vue'
import {
  AbrirArchivo, AbrirBase, AbrirPapelera, AgregarArchivos, AgregarRaiz, AgregarRaizRuta, Archivo, Buscar, Carpeta, ColorRaiz,
  ContarEliminar, CrearBase, CrearCarpeta, Depurar, Eliminar, EliminarRegistro, Estado, GuardarRegistro, Listar,
  MostrarEnCarpeta, Mover, OlvidarTamanos, QuitarRaiz, Renombrar, TamanoCarpeta,
} from '../wailsjs/go/main/App'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {
  errorNombre, esRaizRuta, fecha, fechaHora, hace, mensajeError, nombreDe, padreDe, raizDeRuta, relDe, rutaWin, SIN_DEPURAR,
  tamano, tamanoLargo, tipoArchivo,
} from './util'

/* ---------- Estado ---------- */
const estado = ref(null)
const cargando = ref(true)
const ocupado = ref(false)
const avisos = ref([])
const dialogo = ref(null)
const verAyuda = ref(false)
const vista = ref('carpeta')          // 'carpeta' | 'grafo' | 'duplicados' | 'vencidos'
const grafoDesde = ref('')
const menu = ref(null)                // menú del clic derecho {x, y, titulo, items}
const dupRef = ref(null)
const vencRef = ref(null)

// Árbol (por demanda). "" = Inicio (las carpetas raíz abiertas); "r1/…" = dentro de una raíz.
const hijos = reactive({})            // ruta -> entradas
const abiertos = reactive({})
const cargandoDir = reactive({})
const limites = reactive({})
const conArchivos = ref(true)

// Selección: carpeta o archivo
const sel = ref({ruta: '', esCarpeta: true})
const detalle = ref(null)             // DetalleCarpeta
const archivo = ref(null)             // DetalleArchivo
const tamanos = reactive({})          // ruta -> TamanoTotal (con subcarpetas): calculado o lo que lleva sumado
const fichaActiva = ref(0)
const editando = ref(false)
const orden = ref({col: 'nombre', asc: true})
const filtroTabla = ref('')

// Búsqueda
const busqueda = ref('')
const resultados = ref(null)
const buscando = ref(false)
let tBusqueda = null

// Formularios de diálogos
const form = reactive({nombre: '', destino: null, color: '#f2b33d'})

function avisar(texto, tipo = 'ok') {
  const id = Math.random()
  avisos.value.push({id, texto, tipo})
  setTimeout(() => (avisos.value = avisos.value.filter((a) => a.id !== id)), tipo === 'error' ? 7000 : 4000)
}
async function ejecutar(fn, exito) {
  ocupado.value = true
  try {
    const r = await fn()
    if (exito) avisar(exito)
    return r ?? true
  } catch (e) {
    avisar(mensajeError(e), 'error')
    return false
  } finally {
    ocupado.value = false
  }
}

/* ---------- Raíces ---------- */
const raices = computed(() => estado.value?.raices || [])
const hayRaices = computed(() => raices.value.length > 0)
const infoRaices = computed(() => Object.fromEntries(raices.value.map((r) => [r.id, r])))
const colores = computed(() => Object.fromEntries(raices.value.map((r) => [r.id, r.color])))
const raizSel = computed(() => infoRaices.value[raizDeRuta(sel.value.ruta)] || null)
const colorDe = (r) => colores.value[raizDeRuta(r)] || 'var(--ambar)'
// Ruta como en Windows con el nombre de su raíz: MANTENIMIENTO\PRIVADA\Informes
function rutaVisible(r) {
  if (!r) return 'Inicio'
  const n = infoRaices.value[raizDeRuta(r)]?.nombre || ''
  return relDe(r) ? `${n}\\${rutaWin(relDe(r))}` : n
}
const nombreVisible = (r) => (!r ? 'Inicio' : esRaizRuta(r) ? infoRaices.value[r]?.nombre || r : nombreDe(r))

/* ---------- Carga ---------- */
async function recargarEstado() {
  try {
    estado.value = await Estado()
    for (const a of estado.value.avisos || []) avisar(a)
  } catch (e) {
    avisar(mensajeError(e), 'error')
    estado.value = {raices: [], recientes: [], paleta: []}
  }
}
async function cargarDir(r, forzar = false) {
  if (hijos[r] && !forzar) return
  cargandoDir[r] = true
  try {
    hijos[r] = await Listar(r)
  } catch (e) {
    avisar(mensajeError(e), 'error')
    if (!hijos[r]) hijos[r] = []
  } finally {
    delete cargandoDir[r]
  }
}
// Vuelve a leer las carpetas indicadas (y sus padres) si ya estaban cargadas.
async function refrescarDirs(rutas) {
  const set = new Set(rutas.flatMap((r) => [r, padreDe(r)]))
  await Promise.all([...set].filter((r) => hijos[r]).map((r) => cargarDir(r, true)))
}
function olvidarBajo(r) {
  for (const o of [hijos, abiertos, limites]) for (const k of Object.keys(o)) if (k === r || k.startsWith(r + '/')) delete o[k]
}
async function reiniciar() {
  for (const o of [hijos, abiertos, limites]) for (const k of Object.keys(o)) delete o[k]
  resultados.value = null
  busqueda.value = ''
  cargando.value = true
  await recargarEstado()
  await cargarDir('')
  for (const r of raices.value) { abiertos[r.id] = true; cargarDir(r.id) }
  await seleccionar({ruta: raices.value.length === 1 ? raices.value[0].id : '', esCarpeta: true})
  cargando.value = false
}
// Al abrir una carpeta raíz se lee su Excel de fichas: se informa qué se encontró.
async function avisarFichas(id) {
  const e = infoRaices.value[id]
  if (!e) return
  if (e.error) { avisar(e.error, 'error'); return }
  if (e.baseError) avisar(`${e.nombre}: ${e.baseError}`, 'error')
  else if (!e.baseExiste) avisar(`${e.nombre} no tiene “${estado.value.nombreExcel}”. Puede crearlo desde su carpeta raíz.`)
  else {
    await cargarDir(id)
    const principales = (hijos[id] || []).filter((x) => x.esCarpeta)
    avisar(`${e.nombre}: ${principales.filter((x) => x.ficha).length} de ${principales.length} carpetas principales tienen ficha técnica.`)
  }
}
async function despuesDeAgregar(r) {
  if (!r || r === true) return
  await recargarEstado()
  await cargarDir('', true)
  abiertos[r.id] = true
  await cargarDir(r.id)
  await seleccionar({ruta: r.id, esCarpeta: true})
  avisarFichas(r.id)
}
async function agregarRaiz() {
  const r = await ejecutar(() => AgregarRaiz())
  await despuesDeAgregar(r)
}
async function usarReciente(ruta) {
  const r = await ejecutar(() => AgregarRaizRuta(ruta))
  await despuesDeAgregar(r)
}
async function quitarRaiz(id) {
  if (!(await ejecutar(() => QuitarRaiz(id)))) return
  olvidarBajo(id)
  await recargarEstado()
  await cargarDir('', true)
  if (raizDeRuta(sel.value.ruta) === id) await seleccionar({ruta: '', esCarpeta: true})
  if (grafoDesde.value && raizDeRuta(grafoDesde.value) === id) grafoDesde.value = ''
}
function pedirColor(id) {
  const r = infoRaices.value[id]
  form.color = r.color
  dialogo.value = {
    titulo: 'Color de ' + r.nombre, tipo: 'color', textoConfirmar: 'Usar este color',
    mensaje: 'Las carpetas de esta raíz se ven con este color en la barra lateral, el gráfico y las listas.',
    ok: async () => {
      if (await ejecutar(() => ColorRaiz(id, form.color))) await recargarEstado()
    },
  }
}
// La app arranca sin carpetas y pide elegir una de una vez.
onMounted(async () => {
  window.addEventListener('keydown', teclas)
  // El backend avisa lo que va sumando de cada carpeta (aunque el usuario ya esté en otra).
  EventsOn('tamano:progreso', (p) => { tamanos[p.ruta] = p })
  await reiniciar()
  if (!hayRaices.value) agregarRaiz()
})

async function recargarTodo() {
  const abiertas = Object.keys(hijos)
  // Recargar vuelve a calcular los tamaños (por si algo cambió desde el Explorador).
  await OlvidarTamanos()
  for (const k of Object.keys(tamanos)) delete tamanos[k]
  await recargarEstado()
  await Promise.all(abiertas.map((r) => cargarDir(r, true)))
  await recargarSeleccion()
}

/* ---------- Selección ---------- */
async function seleccionar(e, resetear = true) {
  if (editando.value && !(await confirmarDescartar())) return
  try {
    if (e.esCarpeta) {
      const d = await Carpeta(e.ruta)
      detalle.value = d
      archivo.value = null
      if (resetear) { fichaActiva.value = 0; filtroTabla.value = '' }
      if (fichaActiva.value >= d.registros.length) fichaActiva.value = 0
    } else {
      archivo.value = await Archivo(e.ruta)
      detalle.value = null
    }
    sel.value = {ruta: e.ruta, esCarpeta: e.esCarpeta}
    editando.value = false
    abrirPadres(e.ruta)
    if (e.esCarpeta) calcularTamano(e.ruta)
  } catch (err) {
    avisar(mensajeError(err), 'error')
  }
}
// Tamaño y cantidad de archivos con todas las subcarpetas. El backend lo guarda: si ya se calculó llega
// al instante; si se está calculando, sigue aunque el usuario cambie de carpeta. Solo se recalcula
// cuando algo cambia en esa carpeta (crear, mover, renombrar, eliminar, agregar, depurar) o con Recargar.
async function calcularTamano(r) {
  if (!hayRaices.value) return
  try {
    tamanos[r] = await TamanoCarpeta(r)
  } catch {
    delete tamanos[r]
  }
}
// Lo que se muestra: el total guardado o lo que lleva sumado mientras recorre (sube en vivo).
const tv = computed(() => tamanos[sel.value.ruta] || null)
const sumando = computed(() => !tv.value || tv.value.cargando)
const notaTam = computed(() => {
  const t = tv.value
  if (!t) return 'calculando…'
  if (t.enCola) return 'en espera (hay otros cálculos en curso)…'
  if (t.cargando) return `sumando… ${n(t.carpetas)} carpetas revisadas`
  const h = t.hora ? new Date(t.hora).toLocaleTimeString('es-CO', {hour: '2-digit', minute: '2-digit'}) : ''
  return `con todas sus subcarpetas${h ? ' · calculado ' + h : ''}`
})
const n = (x) => (x || 0).toLocaleString('es-CO')
async function recargarSeleccion() {
  const s = sel.value
  try {
    if (s.esCarpeta) { detalle.value = await Carpeta(s.ruta); calcularTamano(s.ruta) }
    else archivo.value = await Archivo(s.ruta)
  } catch {
    await seleccionar({ruta: padreDe(s.ruta), esCarpeta: true}) // ya no existe: se sube un nivel
  }
}
function abrirPadres(r) {
  const partes = r ? r.split('/') : []
  const tareas = []
  for (let i = 1; i < partes.length; i++) {
    const p = partes.slice(0, i).join('/')
    abiertos[p] = true
    tareas.push(cargarDir(p))
  }
  return Promise.all(tareas)
}
async function elegirArbol(e) {
  if (e.esCarpeta && !abiertos[e.ruta]) { abiertos[e.ruta] = true; cargarDir(e.ruta) }
  await seleccionar(e)
  if (vista.value === 'grafo' && e.esCarpeta) grafoDesde.value = e.ruta
}
async function alternar(r) {
  abiertos[r] = !abiertos[r]
  if (abiertos[r]) await cargarDir(r)
}
function verMas(r) { limites[r] = (limites[r] || 200) + 200 }
function confirmarDescartar() {
  return new Promise((res) => {
    dialogo.value = {
      titulo: 'Descartar cambios', mensaje: 'Tiene cambios sin guardar en la ficha. ¿Desea descartarlos?',
      textoConfirmar: 'Descartar', peligro: true,
      ok: () => { editando.value = false; res(true) }, cancel: () => res(false),
    }
  })
}
async function irACarpeta(r) {
  vista.value = 'carpeta'
  await abrirPadres(r + '/x')
  await seleccionar({ruta: r, esCarpeta: true})
}

/* ---------- Búsqueda (palabras en cualquier orden, o parecidas) ---------- */
watch(busqueda, (q) => {
  clearTimeout(tBusqueda)
  if (q.trim().length < 2) { resultados.value = null; buscando.value = false; return }
  buscando.value = true
  tBusqueda = setTimeout(async () => {
    try {
      const r = await Buscar(q, '')
      if (busqueda.value === q) resultados.value = r
    } catch (e) {
      avisar(mensajeError(e), 'error')
    } finally {
      if (busqueda.value === q) buscando.value = false
    }
  }, 400)
})

/* ---------- Derivados ---------- */
const registros = computed(() => detalle.value?.registros || [])
const ficha = computed(() => registros.value[fichaActiva.value] || null)
const migas = computed(() => {
  const r = sel.value.ruta
  const partes = r ? r.split('/') : []
  return [{nombre: 'Inicio', ruta: ''}, ...partes.map((p, i) => ({nombre: i === 0 ? nombreVisible(p) : p, ruta: partes.slice(0, i + 1).join('/')}))]
})
const entradasTabla = computed(() => {
  let l = [...(detalle.value?.entradas || [])]
  const q = filtroTabla.value.trim().toLowerCase()
  if (q) l = l.filter((e) => e.nombre.toLowerCase().includes(q))
  const {col, asc} = orden.value
  const k = {
    nombre: (e) => e.nombre.toLowerCase(), tipo: (e) => (e.esCarpeta ? '' : tipoArchivo(e.ext).tipo),
    tamano: (e) => e.tamano, fecha: (e) => e.modificado,
  }[col]
  l.sort((a, b) => {
    if (a.esCarpeta !== b.esCarpeta) return a.esCarpeta ? -1 : 1
    return (k(a) < k(b) ? -1 : k(a) > k(b) ? 1 : 0) * (asc ? 1 : -1)
  })
  return l
})
function ordenarPor(col) { orden.value = orden.value.col === col ? {col, asc: !orden.value.asc} : {col, asc: col === 'nombre' || col === 'tipo'} }
const flecha = (col) => (orden.value.col === col ? (orden.value.asc ? '▲' : '▼') : '')
const principales = computed(() => (detalle.value?.esRaiz ? detalle.value.entradas.filter((e) => e.esCarpeta) : []))
const tiposEnCarpeta = computed(() => {
  const m = new Map()
  for (const e of detalle.value?.entradas || []) if (!e.esCarpeta) { const t = tipoArchivo(e.ext); m.set(t.tipo, {n: (m.get(t.tipo)?.n || 0) + 1, color: t.color}) }
  return [...m.entries()].sort((a, b) => b[1].n - a[1].n)
})
// Carpeta en la que trabajan Duplicados y Vencidos: la elegida (o la que contiene el archivo elegido).
const alcance = computed(() => (sel.value.esCarpeta ? sel.value.ruta : padreDe(sel.value.ruta)))
const textoRet = (v) => (v === SIN_DEPURAR ? 'No se depura' : 'Conservar ' + v)

/* ---------- Acciones de archivos ---------- */
const abrirArchivo = (r) => ejecutar(() => AbrirArchivo(r))
const mostrar = (r) => ejecutar(() => MostrarEnCarpeta(r))

function pedirNuevaCarpeta(padre = null) {
  const p = padre ?? (sel.value.esCarpeta ? sel.value.ruta : padreDe(sel.value.ruta))
  if (!p) { avisar('Elija primero en cuál carpeta raíz se crea', 'error'); return }
  form.nombre = ''
  form.destino = p
  dialogo.value = {
    titulo: 'Nueva carpeta', tipo: 'nombre', textoConfirmar: 'Crear carpeta',
    mensaje: `Se crea dentro de “${nombreVisible(p)}”.`,
    ok: async () => {
      const r = await ejecutar(() => CrearCarpeta(form.destino, form.nombre), 'Carpeta creada')
      if (r && r !== true) {
        await refrescarDirs([form.destino])
        abiertos[form.destino] = true
        await recargarSeleccion()
      }
    },
  }
}
function pedirRenombrar(e) {
  if (esRaizRuta(e.ruta)) { avisar('La carpeta raíz no se renombra desde aquí', 'error'); return }
  form.nombre = nombreDe(e.ruta)
  form.destino = padreDe(e.ruta)
  dialogo.value = {
    titulo: e.esCarpeta ? 'Renombrar carpeta' : 'Renombrar archivo', tipo: 'nombre', textoConfirmar: 'Renombrar', elemento: e,
    mensaje: e.esCarpeta ? 'Las fichas de esta carpeta y de sus subcarpetas se actualizan solas en el Excel.' : 'Revise que la extensión (.pdf, .xlsx…) siga igual.',
    ok: async () => {
      const nueva = await ejecutar(() => Renombrar(e.ruta, form.nombre), 'Nombre cambiado')
      if (nueva && nueva !== true) await despuesDeMover(e, nueva)
    },
  }
}
function pedirMover(e) {
  form.destino = null
  dialogo.value = {
    titulo: e.esCarpeta ? 'Mover carpeta' : 'Mover archivo', tipo: 'mover', textoConfirmar: 'Mover aquí', elemento: e,
    mensaje: `Elija la carpeta destino para “${nombreDe(e.ruta)}”.` + (e.esCarpeta ? ' Se mueve con todo su contenido y sus fichas.' : ''),
    ok: () => moverA(e, form.destino),
  }
}
async function moverA(e, destino) {
  const nueva = await ejecutar(() => Mover(e.ruta, destino), 'Movido a ' + nombreVisible(destino))
  if (nueva && nueva !== true) await despuesDeMover(e, nueva, destino)
}
// Arrastrar y soltar en la barra lateral: se confirma antes de mover.
function soltarEn(o, destino) {
  dialogo.value = {
    titulo: o.esCarpeta ? 'Mover carpeta' : 'Mover archivo', textoConfirmar: 'Mover',
    mensaje: `¿Mover “${o.nombre}” a “${nombreVisible(destino.ruta)}”?`,
    detalles: [
      `Desde: ${rutaVisible(padreDe(o.ruta))}`,
      `Hacia: ${rutaVisible(destino.ruta)}`,
      ...(o.esCarpeta ? ['Se mueve con todo su contenido; sus fichas y explicaciones se actualizan en el Excel.'] : []),
      ...(raizDeRuta(o.ruta) !== raizDeRuta(destino.ruta) ? ['Es otra carpeta raíz: si tiene fichas en el Excel de su raíz, no se podrá mover.'] : []),
    ],
    ok: () => moverA(o, destino.ruta),
  }
}
async function despuesDeMover(e, nueva, destino = null) {
  const viejaSel = sel.value.ruta
  olvidarBajo(e.ruta)
  await refrescarDirs([e.ruta, nueva, ...(destino !== null ? [destino] : [])])
  await recargarEstado()
  if (viejaSel === e.ruta || viejaSel.startsWith(e.ruta + '/')) await seleccionar({ruta: nueva + viejaSel.slice(e.ruta.length), esCarpeta: sel.value.esCarpeta}, false)
  else await recargarSeleccion()
}
async function pedirEliminar(e, despues = null) {
  const r = e.esCarpeta ? await ejecutar(() => ContarEliminar(e.ruta)) : {carpetas: 0, archivos: 0, fichas: 0}
  if (!r) return
  const contenido = r.carpetas + r.archivos
  dialogo.value = {
    titulo: e.esCarpeta ? 'Eliminar carpeta' : 'Eliminar archivo', peligro: true, textoConfirmar: 'Eliminar',
    mensaje: `“${nombreDe(e.ruta)}” se moverá a la papelera de la aplicación (${estado.value.nombrePapelera}, en ${nombreVisible(raizDeRuta(e.ruta))}).`,
    detalles: [
      ...(e.esCarpeta ? [`Contiene ${r.carpetas} subcarpeta(s) y ${r.archivos} archivo(s)${r.cortado ? ' o más (es muy grande)' : ''}.`] : []),
      ...(r.fichas ? [`Se borran ${r.fichas} ficha(s) técnica(s) del Excel (quedan copiadas en la hoja Historial).`] : []),
      ...(r.descripciones ? [`Se borran ${r.descripciones} explicación(es) de subcarpetas de la ficha (quedan en el Historial).`] : []),
      'Se puede recuperar desde la papelera de la aplicación (botón en la parte inferior de la barra lateral).',
    ],
    escribir: e.esCarpeta && (contenido > 0 || r.fichas > 0 || r.descripciones > 0) ? nombreDe(e.ruta) : '',
    ok: async () => {
      const dst = await ejecutar(() => Eliminar(e.ruta), 'Eliminado (está en la papelera de la aplicación)')
      if (dst) {
        olvidarBajo(e.ruta)
        await refrescarDirs([e.ruta])
        await recargarEstado()
        if (sel.value.ruta === e.ruta || sel.value.ruta.startsWith(e.ruta + '/')) await seleccionar({ruta: padreDe(e.ruta), esCarpeta: true})
        else await recargarSeleccion()
        despues?.()
      }
    },
  }
}
async function agregarArchivos(destino) {
  const r = await ejecutar(() => AgregarArchivos(destino))
  if (r && r !== true && r.length) {
    avisar(`${r.length} archivo(s) agregados`)
    await refrescarDirs([destino])
    abiertos[destino] = true
    await recargarSeleccion()
  }
}

/* ---------- Clic derecho ---------- */
function abrirMenu(ev, e) {
  const r = e.ruta
  const sep = {separador: true}
  let items
  if (!r) {
    items = [
      {texto: 'Agregar carpeta raíz', icono: 'carpeta-mas', accion: agregarRaiz},
      {texto: 'Volver a leer todo', icono: 'recargar', tecla: 'F5', accion: recargarTodo},
    ]
  } else if (esRaizRuta(r)) {
    const info = infoRaices.value[r]
    items = [
      {texto: 'Ver la carpeta raíz', icono: 'carpeta-abierta', accion: () => elegirArbol(e)},
      {texto: 'Nueva carpeta', icono: 'carpeta-mas', accion: () => pedirNuevaCarpeta(r)},
      {texto: 'Agregar archivos', icono: 'subir', accion: () => agregarArchivos(r)},
      sep,
      {texto: 'Ver en el gráfico', icono: 'grafo', accion: () => verGrafo(r)},
      {texto: 'Buscar duplicados aquí', icono: 'copias', accion: () => irVista('duplicados', e)},
      {texto: 'Revisar vencidos aquí', icono: 'reloj', accion: () => irVista('vencidos', e)},
      sep,
      {texto: 'Cambiar color', icono: 'paleta', accion: () => pedirColor(r)},
      {texto: 'Mostrar en el Explorador', icono: 'abrir', accion: () => mostrar(r)},
      ...(info?.baseExiste ? [{texto: 'Abrir Excel de fichas', icono: 'excel', accion: () => ejecutar(() => AbrirBase(r))}] : []),
      sep,
      {texto: 'Quitar de la lista', icono: 'quitar', titulo: 'Solo la cierra en la aplicación; no borra nada', accion: () => quitarRaiz(r)},
    ]
  } else if (e.esCarpeta) {
    items = [
      {texto: 'Ver la carpeta', icono: 'carpeta-abierta', accion: () => elegirArbol(e)},
      {texto: 'Nueva subcarpeta', icono: 'carpeta-mas', accion: () => pedirNuevaCarpeta(r)},
      {texto: 'Agregar archivos', icono: 'subir', accion: () => agregarArchivos(r)},
      sep,
      {texto: 'Ver en el gráfico', icono: 'grafo', accion: () => verGrafo(r)},
      {texto: 'Buscar duplicados aquí', icono: 'copias', accion: () => irVista('duplicados', e)},
      {texto: 'Revisar vencidos aquí', icono: 'reloj', accion: () => irVista('vencidos', e)},
      sep,
      {texto: 'Renombrar', icono: 'editar', tecla: 'F2', accion: () => pedirRenombrar(e)},
      {texto: 'Mover a…', icono: 'mover', accion: () => pedirMover(e)},
      {texto: 'Mostrar en el Explorador', icono: 'abrir', accion: () => mostrar(r)},
      sep,
      {texto: 'Eliminar', icono: 'borrar', peligro: true, accion: () => pedirEliminar(e)},
    ]
  } else {
    items = [
      {texto: 'Abrir', icono: 'abrir', accion: () => abrirArchivo(r)},
      {texto: 'Ver detalles', icono: 'archivo', accion: () => elegirArbol(e)},
      {texto: 'Mostrar en el Explorador', icono: 'carpeta-abierta', accion: () => mostrar(r)},
      sep,
      {texto: 'Renombrar', icono: 'editar', tecla: 'F2', accion: () => pedirRenombrar(e)},
      {texto: 'Mover a…', icono: 'mover', accion: () => pedirMover(e)},
      sep,
      {texto: 'Eliminar', icono: 'borrar', peligro: true, accion: () => pedirEliminar(e)},
    ]
  }
  menu.value = {x: ev.clientX, y: ev.clientY, titulo: e.nombre || nombreVisible(r), items}
}
async function irVista(v, e) {
  await seleccionar({ruta: e.ruta, esCarpeta: true})
  vista.value = v
}

/* ---------- Fichas ---------- */
async function guardarFicha(r, subs = []) {
  const nuevo = !r.id
  const res = await ejecutar(() => GuardarRegistro(r, subs), nuevo ? 'Ficha creada en el Excel' : 'Cambios guardados en el Excel')
  if (!res || res === true) return
  editando.value = false
  await recargarEstado()
  await refrescarDirs([res.ruta])
  await recargarSeleccion()
  fichaActiva.value = Math.max(0, registros.value.findIndex((x) => x.id === res.id))
}
function nuevaFicha() {
  const r = {id: 0, ruta: sel.value.ruta, actualizado: '', valores: {'Nombre de la carpeta': detalle.value?.carpeta?.nombre || ''}}
  detalle.value = {...detalle.value, registros: [...registros.value, r]}
  fichaActiva.value = detalle.value.registros.length - 1
  editando.value = true
}
function cancelarEdicion() {
  editando.value = false
  if (ficha.value && !ficha.value.id) {
    detalle.value = {...detalle.value, registros: registros.value.filter((x) => x.id)}
    fichaActiva.value = 0
  }
}
function pedirEliminarFicha(r = ficha.value) {
  dialogo.value = {
    titulo: 'Eliminar ficha', peligro: true, textoConfirmar: 'Eliminar ficha',
    mensaje: `Se borra la ficha de “${rutaVisible(r.ruta)}” del Excel. La carpeta no se toca.`,
    detalles: ['Los datos quedan copiados en la hoja Historial del Excel.'],
    ok: async () => {
      if (!r.id) { cancelarEdicion(); return }
      if (await ejecutar(() => EliminarRegistro(raizDeRuta(r.ruta), r.id), 'Ficha eliminada')) {
        fichaActiva.value = 0
        await recargarEstado()
        await refrescarDirs([r.ruta])
        await recargarSeleccion()
      }
    },
  }
}

/* ---------- Excel y papelera (de la raíz de lo elegido) ---------- */
async function crearBase(id) {
  if (await ejecutar(() => CrearBase(id), 'Excel de fichas creado en la raíz')) { await recargarEstado(); await recargarSeleccion() }
}
const abrirExcel = () => raizSel.value && ejecutar(() => AbrirBase(raizSel.value.id))
const abrirPapelera = () => raizSel.value && ejecutar(() => AbrirPapelera(raizSel.value.id))

/* ---------- Vistas ---------- */
function verGrafo(desde = null) {
  grafoDesde.value = desde ?? (sel.value.esCarpeta ? sel.value.ruta : padreDe(sel.value.ruta))
  vista.value = 'grafo'
}
async function elegirDesdeGrafo(n) {
  await abrirPadres(n.ruta)
  await seleccionar({ruta: n.ruta, esCarpeta: n.esCarpeta})
}
// Eliminar una copia desde Duplicados.
function eliminarCopia(ruta) {
  pedirEliminar({ruta, esCarpeta: false}, () => dupRef.value?.quitar(ruta))
}
// Depurar desde Vencidos: los archivos marcados van a la papelera de su raíz.
function pedirDepurar(rutas, bytes) {
  const n = rutas.length
  dialogo.value = {
    titulo: 'Depurar archivos vencidos', peligro: true, textoConfirmar: `Mover ${n} a la papelera`,
    mensaje: `Se moverán ${n} archivo(s) (${tamanoLargo(bytes)}) a la papelera de la aplicación de su carpeta raíz.`,
    detalles: [
      'Quedan en _papelera_organizador\\<fecha> depuracion\\… con la misma ruta, por si hay que recuperarlos.',
      'Cada archivo queda anotado en la hoja Historial del Excel de fichas.',
    ],
    escribir: n >= 20 ? 'DEPURAR' : '',
    ok: async () => {
      const r = await ejecutar(() => Depurar(rutas))
      if (!r || r === true) return
      vencRef.value?.quitar(r.movidas)
      avisar(`${r.movidos} archivo(s) movidos a la papelera (${tamanoLargo(r.tamano)})`)
      if (r.errores.length) avisar(`${r.errores.length} no se pudieron mover: ${r.errores.slice(0, 3).join(' · ')}${r.errores.length > 3 ? '…' : ''}`, 'error')
      await refrescarDirs(r.movidas.map(padreDe))
      await recargarEstado()
    },
  }
}

/* ---------- Diálogos ---------- */
const errorForm = computed(() => {
  const d = dialogo.value
  if (!d) return ''
  if (d.tipo === 'nombre') {
    const e = errorNombre(form.nombre)
    if (e) return e
    const hermanos = hijos[form.destino] || []
    const actual = d.elemento ? nombreDe(d.elemento.ruta).toLowerCase() : null
    const n = form.nombre.trim().toLowerCase()
    if (n !== actual && hermanos.some((h) => h.nombre.toLowerCase() === n)) return 'Ya existe un elemento con ese nombre aquí'
    return ''
  }
  if (d.tipo === 'mover') {
    if (form.destino === null || form.destino === '') return 'Elija la carpeta destino'
    if (d.elemento && padreDe(d.elemento.ruta) === form.destino) return 'Ya está en esa carpeta'
  }
  if (d.tipo === 'color' && !/^#[0-9a-f]{6}$/i.test(form.color)) return 'Color no válido'
  return ''
})
async function cerrarDialogo(confirmado) {
  const d = dialogo.value
  if (!d) return
  if (!confirmado) { d.cancel?.(); dialogo.value = null; return }
  dialogo.value = null
  await d.ok?.()
}
function teclas(e) {
  if (e.key === 'F1') { e.preventDefault(); verAyuda.value = true; return }
  if (dialogo.value || verAyuda.value || menu.value) return
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') { e.preventDefault(); document.getElementById('buscar')?.focus(); return }
  const enCampo = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName)
  if (e.key === 'F2' && !enCampo && sel.value.ruta && !esRaizRuta(sel.value.ruta)) { e.preventDefault(); pedirRenombrar(sel.value) }
  if (e.key === 'F5') { e.preventDefault(); recargarTodo() }
}
watch(dialogo, async (d) => {
  if (d?.tipo === 'nombre') { await nextTick(); const i = document.querySelector('.form-dialogo input'); i?.focus(); i?.select() }
})
</script>

<template>
  <div class="app">
    <!-- ================= SIDEBAR ================= -->
    <aside class="sidebar">
      <div class="marca">
        <strong>Organizador de carpetas</strong>
        <button class="agregar-raiz" title="Abrir otra carpeta raíz (cada una con su propio Excel de fichas)" @click="agregarRaiz">
          <Icono nombre="carpeta-mas"/><span>Agregar carpeta raíz</span>
        </button>
      </div>

      <template v-if="hayRaices">
        <div class="vistas" role="tablist">
          <button role="tab" :aria-selected="vista === 'carpeta'" :class="{activo: vista === 'carpeta'}" @click="vista = 'carpeta'"><Icono nombre="lista"/>Carpetas</button>
          <button role="tab" :aria-selected="vista === 'grafo'" :class="{activo: vista === 'grafo'}" @click="verGrafo()"><Icono nombre="grafo"/>Gráfica</button>
          <button role="tab" :aria-selected="vista === 'duplicados'" :class="{activo: vista === 'duplicados'}" @click="vista = 'duplicados'"><Icono nombre="copias"/>Duplicados</button>
          <button role="tab" :aria-selected="vista === 'vencidos'" :class="{activo: vista === 'vencidos'}" @click="vista = 'vencidos'"><Icono nombre="reloj"/>Vencidos</button>
        </div>
        <label class="buscador">
          <Icono nombre="buscar"/>
          <input id="buscar" v-model="busqueda" placeholder="Buscar: una o varias palabras" autocomplete="off"/>
          <button v-if="busqueda" class="limpiar" aria-label="Limpiar búsqueda" @click="busqueda = ''"><Icono nombre="cerrar"/></button>
        </label>
        <div class="opciones-arbol">
          <label><input type="checkbox" v-model="conArchivos"/> Mostrar archivos</label>
          <button class="enlace-claro" title="Cerrar todas las carpetas" @click="Object.keys(abiertos).forEach((k) => delete abiertos[k])">Contraer todo</button>
        </div>

        <nav class="lista" aria-label="Carpetas y archivos">
          <!-- Resultados de búsqueda -->
          <template v-if="busqueda.trim().length >= 2">
            <p v-if="buscando" class="nada">Buscando en todas las carpetas raíz…</p>
            <template v-else-if="resultados">
              <p class="nada">{{ resultados.total }} resultado(s){{ resultados.total > resultados.entradas.length ? `, se muestran los ${resultados.entradas.length} más parecidos` : '' }}{{ resultados.cortada ? ' (búsqueda cortada por tiempo)' : '' }}</p>
              <button v-for="e in resultados.entradas" :key="e.ruta" class="resultado" :class="{activo: sel.ruta === e.ruta}" :style="{'--cf': colorDe(e.ruta)}"
                      @click="seleccionar(e)" @contextmenu.prevent="abrirMenu($event, e)">
                <Icono v-if="e.esCarpeta" nombre="carpeta" class="ic carp"/>
                <span v-else class="ic arch" :style="{'--c': tipoArchivo(e.ext).color}"><Icono nombre="archivo"/></span>
                <span class="txt"><b>{{ e.nombre }}</b><small>{{ rutaVisible(padreDe(e.ruta) || raizDeRuta(e.ruta)) }}</small></span>
              </button>
            </template>
          </template>
          <!-- Árbol -->
          <template v-else>
            <div class="fila-raiz" :class="{activo: sel.ruta === ''}" @click="seleccionar({ruta: '', esCarpeta: true})"
                 @contextmenu.prevent="abrirMenu($event, {ruta: '', esCarpeta: true, nombre: 'Inicio'})">
              <Icono nombre="casa"/><span>Inicio</span><small>{{ raices.length }} carpeta(s) raíz</small>
            </div>
            <p v-if="cargandoDir[''] && !hijos['']" class="nada">Leyendo…</p>
            <Arbol v-else :entradas="hijos[''] || []" :hijos="hijos" :abiertos="abiertos" :cargando="cargandoDir" :limites="limites"
                   :colores="colores" :seleccion="sel.ruta" :con-archivos="conArchivos" ruta=""
                   @elegir="elegirArbol" @alternar="alternar" @mas="verMas" @menu="abrirMenu" @soltar="soltarEn"/>
            <p class="pista-arbol">Clic derecho: más acciones · Arrastre para mover</p>
          </template>
        </nav>

        <footer class="pie">
          <button class="btn nueva" :disabled="!sel.ruta" @click="pedirNuevaCarpeta()"><Icono nombre="carpeta-mas"/>Nueva carpeta</button>
          <div class="botones">
            <button class="btn fantasma claro" title="Volver a leer (F5)" @click="recargarTodo"><Icono nombre="recargar"/>Recargar</button>
            <button class="btn fantasma claro" :disabled="!raizSel?.baseExiste" :title="raizSel ? 'Excel de fichas de ' + raizSel.nombre : 'Elija una carpeta raíz'" @click="abrirExcel"><Icono nombre="excel"/>Abrir Excel</button>
            <button class="btn fantasma claro" :disabled="!raizSel?.papeleraHay" :title="raizSel ? 'Lo eliminado en ' + raizSel.nombre : 'Elija una carpeta raíz'" @click="abrirPapelera"><Icono nombre="papelera"/>Papelera</button>
            <button class="btn fantasma claro" title="Cómo usar la aplicación (F1)" @click="verAyuda = true"><Icono nombre="ayuda"/>Ayuda</button>
          </div>
        </footer>
      </template>
    </aside>

    <!-- ================= MAIN ================= -->
    <main class="principal" :class="{oscuro: vista === 'grafo' && hayRaices}">
      <!-- Sin carpetas -->
      <section v-if="!cargando && !hayRaices" class="pantalla-vacia">
        <h1>Elija la carpeta que quiere organizar</h1>
        <p>Puede ser una carpeta local o de red, y puede abrir varias (por ejemplo <code>MANTENIMIENTO\PRIVADA</code> y <code>RRHH\PRIVADA</code>): cada una lleva su propio Excel de fichas técnicas (<code>{{ estado?.nombreExcel || 'Fichas tecnicas carpetas.xlsx' }}</code>) y su color.</p>
        <button class="btn primario" @click="agregarRaiz"><Icono nombre="carpeta"/>Elegir carpeta</button>
        <div v-if="estado?.recientes?.length" class="recientes">
          <p class="muted">Usadas recientemente</p>
          <button v-for="r in estado.recientes" :key="r" class="reciente" @click="usarReciente(r)"><Icono nombre="carpeta"/>{{ r }}</button>
        </div>
      </section>

      <template v-else-if="hayRaices">
        <!-- Vista gráfica -->
        <Grafo v-if="vista === 'grafo'" :desde="grafoDesde" :raices="infoRaices" :seleccion="sel.ruta"
               @elegir="elegirDesdeGrafo" @abrir="abrirArchivo" @desde="(r) => (grafoDesde = r)"/>

        <!-- Duplicados y vencidos: se mantienen vivos para no perder los resultados al cambiar de pestaña -->
        <Duplicados v-show="vista === 'duplicados'" ref="dupRef" :alcance="alcance" :raices="infoRaices"
                    @abrir="abrirArchivo" @mostrar="mostrar" @ir="irACarpeta" @eliminar="eliminarCopia"/>
        <Vencidos v-show="vista === 'vencidos'" ref="vencRef" :alcance="alcance" :raices="infoRaices"
                  @abrir="abrirArchivo" @mostrar="mostrar" @ir="irACarpeta" @depurar="pedirDepurar"/>

        <template v-if="vista === 'carpeta'">
          <!-- Archivo -->
          <section v-if="archivo && !sel.esCarpeta" class="detalle">
            <nav class="migas" aria-label="Ubicación">
              <template v-for="(m, i) in migas" :key="m.ruta">
                <button v-if="i < migas.length - 1" @click="seleccionar({ruta: m.ruta, esCarpeta: true})">{{ m.nombre }}</button>
                <span v-if="i < migas.length - 1" class="sep">\</span>
              </template>
            </nav>
            <header class="cabecera">
              <div class="titulo-arch">
                <span class="ic-grande" :style="{'--c': tipoArchivo(archivo.archivo.ext).color}"><Icono nombre="archivo"/></span>
                <div>
                  <h1>{{ archivo.archivo.nombre }}</h1>
                  <span class="chip-tipo" :style="{'--c': tipoArchivo(archivo.archivo.ext).color}">{{ tipoArchivo(archivo.archivo.ext).tipo }}</span>
                </div>
              </div>
              <div class="acciones">
                <button class="btn" @click="pedirRenombrar(sel)"><Icono nombre="editar"/>Renombrar</button>
                <button class="btn" @click="pedirMover(sel)"><Icono nombre="mover"/>Mover</button>
                <button class="btn peligro" @click="pedirEliminar(sel)"><Icono nombre="borrar"/>Eliminar</button>
              </div>
            </header>
            <div class="tarjeta-arch" :style="{'--cf': colorDe(sel.ruta)}">
              <dl>
                <div><dt>Tamaño</dt><dd>{{ tamanoLargo(archivo.archivo.tamano) }}</dd></div>
                <div><dt>Modificado</dt><dd>{{ fechaHora(archivo.archivo.modificado) }} <span class="muted">({{ hace(archivo.archivo.modificado) }})</span></dd></div>
                <div v-if="archivo.creado"><dt>Creado</dt><dd>{{ fechaHora(archivo.creado) }}</dd></div>
                <div v-if="archivo.retencion"><dt>Retención de su carpeta</dt>
                  <dd>{{ textoRet(archivo.retencion) }}
                    <span v-if="archivo.vence" class="estado-ret" :class="{vencido: archivo.vence < new Date().toISOString()}">
                      {{ archivo.vence < new Date().toISOString() ? 'vencido desde el ' : 'vence el ' }}{{ fecha(archivo.vence) }}
                    </span>
                  </dd></div>
                <div class="ancho"><dt>Ubicación</dt><dd class="ruta">{{ archivo.ubicacion }}</dd></div>
              </dl>
              <div class="acciones-arch">
                <button class="btn primario" @click="abrirArchivo(archivo.archivo.ruta)"><Icono nombre="abrir"/>Abrir</button>
                <button class="btn" @click="mostrar(archivo.archivo.ruta)"><Icono nombre="carpeta"/>Mostrar en el Explorador</button>
                <button class="btn fantasma" @click="seleccionar({ruta: padreDe(archivo.archivo.ruta), esCarpeta: true})"><Icono nombre="carpeta-abierta"/>Ver la carpeta que lo contiene</button>
              </div>
            </div>
          </section>

          <!-- Inicio: todas las carpetas raíz -->
          <section v-else-if="detalle?.esInicio" class="detalle">
            <header class="cabecera">
              <div class="titulo-carp"><Icono nombre="casa" class="ic-carp"/><h1>Inicio</h1></div>
              <div class="acciones">
                <button class="btn primario" @click="agregarRaiz"><Icono nombre="carpeta-mas"/>Agregar carpeta raíz</button>
                <button class="btn" @click="verGrafo('')"><Icono nombre="grafo"/>Ver en gráfico</button>
              </div>
            </header>
            <dl class="cifras">
              <div><dt>Carpetas raíz abiertas</dt><dd>{{ raices.length }}</dd></div>
              <div><dt>Archivos (en todas)</dt><dd :class="{sumando}">{{ n(tv?.archivos) }}<small>{{ sumando ? 'sumando…' : 'en todas las subcarpetas' }}</small></dd></div>
              <div><dt>Tamaño total</dt><dd :class="{sumando}">{{ tamanoLargo(tv?.tamano) }}<small>{{ notaTam }}</small></dd></div>
            </dl>
            <div class="raices">
              <article v-for="r in raices" :key="r.id" class="tarjeta-raiz" :style="{'--cf': r.color}">
                <button class="cab-raiz" @click="elegirArbol({ruta: r.id, esCarpeta: true})">
                  <Icono nombre="carpeta" class="ic-r"/>
                  <span><b>{{ r.nombre }}</b><small>{{ r.ruta }}</small></span>
                </button>
                <p v-if="r.error" class="estado-r error">{{ r.error }}</p>
                <p v-else-if="r.baseError" class="estado-r error">{{ r.baseError }}</p>
                <p v-else-if="!r.baseExiste" class="estado-r">Sin Excel de fichas</p>
                <p v-else class="estado-r ok">{{ r.registros }} ficha(s) técnica(s) en su Excel</p>
                <div class="acc-raiz">
                  <button class="btn fantasma ic" title="Cambiar color" @click="pedirColor(r.id)"><Icono nombre="paleta"/></button>
                  <button class="btn fantasma ic" title="Mostrar en el Explorador" @click="mostrar(r.id)"><Icono nombre="abrir"/></button>
                  <button v-if="r.baseExiste" class="btn fantasma ic" title="Abrir Excel de fichas" @click="ejecutar(() => AbrirBase(r.id))"><Icono nombre="excel"/></button>
                  <button class="btn fantasma ic" title="Quitar de la lista (no borra nada)" @click="quitarRaiz(r.id)"><Icono nombre="quitar"/></button>
                </div>
              </article>
            </div>
            <div v-if="estado.recientes.filter((x) => !raices.some((r) => r.ruta.toLowerCase() === x.toLowerCase())).length" class="recientes">
              <p class="muted">Agregar una usada recientemente</p>
              <button v-for="x in estado.recientes.filter((x) => !raices.some((r) => r.ruta.toLowerCase() === x.toLowerCase()))" :key="x" class="reciente" @click="usarReciente(x)">
                <Icono nombre="carpeta-mas"/>{{ x }}
              </button>
            </div>
          </section>

          <!-- Carpeta -->
          <section v-else-if="detalle" class="detalle" :style="{'--cf': colorDe(sel.ruta)}">
            <nav class="migas" aria-label="Ubicación">
              <template v-for="(m, i) in migas" :key="m.ruta">
                <button v-if="i < migas.length - 1" @click="seleccionar({ruta: m.ruta, esCarpeta: true})">{{ m.nombre }}</button>
                <span v-if="i < migas.length - 1" class="sep">\</span>
              </template>
            </nav>
            <header class="cabecera">
              <div class="titulo-carp">
                <Icono nombre="carpeta-abierta" class="ic-carp"/>
                <div>
                  <h1>{{ detalle.carpeta.nombre }}</h1>
                  <small v-if="detalle.esRaiz" class="muted ruta-raiz">{{ raizSel?.ruta }}</small>
                </div>
              </div>
              <div class="acciones">
                <button class="btn" @click="pedirNuevaCarpeta(sel.ruta)"><Icono nombre="carpeta-mas"/>Nueva subcarpeta</button>
                <button class="btn" @click="agregarArchivos(sel.ruta)"><Icono nombre="subir"/>Agregar archivos</button>
                <button class="btn" @click="verGrafo(sel.ruta)"><Icono nombre="grafo"/>Ver en gráfico</button>
                <template v-if="!detalle.esRaiz">
                  <button class="btn" title="F2" @click="pedirRenombrar(sel)"><Icono nombre="editar"/>Renombrar</button>
                  <button class="btn" @click="pedirMover(sel)"><Icono nombre="mover"/>Mover</button>
                  <button class="btn peligro" @click="pedirEliminar(sel)"><Icono nombre="borrar"/>Eliminar</button>
                </template>
                <template v-else>
                  <button class="btn" @click="pedirColor(sel.ruta)"><Icono nombre="paleta"/>Color</button>
                  <button class="btn" title="Solo la cierra en la aplicación; no borra nada" @click="quitarRaiz(sel.ruta)"><Icono nombre="quitar"/>Quitar</button>
                </template>
                <button class="btn fantasma ic" title="Mostrar en el Explorador" @click="mostrar(sel.ruta)"><Icono nombre="abrir"/></button>
              </div>
            </header>

            <!-- Avisos de la raíz -->
            <template v-if="detalle.esRaiz && raizSel">
              <div v-if="!raizSel.baseExiste" class="aviso">
                <Icono nombre="excel"/>
                <div>
                  <strong>Esta carpeta raíz todavía no tiene Excel de fichas técnicas.</strong>
                  <p>Se creará <code>{{ estado.nombreExcel }}</code> aquí. Las fichas de sus carpetas principales se guardan en él (cada carpeta raíz tiene el suyo).</p>
                </div>
                <button class="btn primario" @click="crearBase(raizSel.id)">Crear Excel de fichas</button>
              </div>
              <div v-else-if="raizSel.baseError" class="aviso error"><Icono nombre="alerta"/><div>{{ raizSel.baseError }}</div></div>
              <div v-if="raizSel.noPrincipal?.length" class="aviso">
                <Icono nombre="alerta"/>
                <div>
                  <strong>{{ raizSel.noPrincipal.length }} ficha(s) no están en una carpeta principal.</strong>
                  <p>Las fichas técnicas solo van en las carpetas que están en la raíz; las subcarpetas se explican dentro de esas fichas. Puede eliminarlas (quedan copiadas en el Historial del Excel).</p>
                  <ul class="sueltas">
                    <li v-for="r in raizSel.noPrincipal" :key="r.id"><code>{{ rutaVisible(r.ruta) }}</code>
                      <button class="enlace" @click="pedirEliminarFicha(r)">Eliminar ficha</button></li>
                  </ul>
                </div>
              </div>
              <div v-if="raizSel.sinCarpeta?.length" class="aviso">
                <Icono nombre="alerta"/>
                <div>
                  <strong>{{ raizSel.sinCarpeta.length }} ficha(s) del Excel apuntan a carpetas que ya no existen.</strong>
                  <p>Pudieron renombrarse o moverse por fuera de la aplicación. Corrija la ruta en el Excel o elimine la ficha.</p>
                  <ul class="sueltas">
                    <li v-for="r in raizSel.sinCarpeta" :key="r.id"><code>{{ rutaVisible(r.ruta) }}</code>
                      <button class="enlace" @click="pedirEliminarFicha(r)">Eliminar ficha</button></li>
                  </ul>
                </div>
              </div>
            </template>

            <dl class="cifras">
              <div><dt>Subcarpetas</dt><dd>{{ detalle.subcarpetas }}<small>{{ tv ? `${n(tv.carpetas)}${sumando ? '…' : ''} en todos los niveles` : 'directas' }}</small></dd></div>
              <div><dt>Archivos</dt>
                <dd :class="{sumando}">{{ n(tv?.archivos) }}<small>{{ sumando ? `sumando… (${detalle.archivos} aquí)` : `${detalle.archivos} aquí, el resto en subcarpetas` }}</small></dd></div>
              <div><dt>Tamaño de la carpeta</dt>
                <dd :class="{sumando}">{{ tamanoLargo(tv?.tamano) }}<small>{{ notaTam }}</small></dd></div>
              <div><dt>Modificada</dt><dd class="pequeno">{{ fecha(detalle.carpeta.modificado) }}<small>{{ hace(detalle.carpeta.modificado) }}</small></dd></div>
              <div v-if="detalle.esPrincipal"><dt>Ficha técnica</dt><dd class="pequeno" :class="detalle.carpeta.ficha ? 'si' : 'no'">{{ detalle.carpeta.ficha ? 'Sí' : 'No tiene' }}</dd></div>
              <div v-else-if="!detalle.esRaiz"><dt>Explicada en la ficha</dt><dd class="pequeno" :class="detalle.descripcion ? 'si' : 'no'">{{ detalle.descripcion ? 'Sí' : 'No' }}</dd></div>
              <div v-if="detalle.retencion"><dt>Retención</dt>
                <dd class="pequeno">{{ textoRet(detalle.retencion) }}<small>{{ detalle.retencionDe === sel.ruta ? 'definida aquí' : 'heredada de ' + nombreDe(detalle.retencionDe) }}</small></dd></div>
            </dl>

            <!-- Ficha técnica: solo en carpetas principales -->
            <section v-if="detalle.esPrincipal" class="bloque">
              <div class="bloque-cab">
                <h2>Ficha técnica</h2>
                <div v-if="registros.length > 1" class="pestanas" role="tablist">
                  <button v-for="(r, i) in registros" :key="r.id || 'nueva'" role="tab" :aria-selected="i === fichaActiva"
                          :class="{activo: i === fichaActiva}" @click="!editando && (fichaActiva = i)">Ficha {{ i + 1 }}</button>
                </div>
              </div>
              <Ficha v-if="ficha" :registro="ficha" :campos="raizSel?.campos || []" :editando="editando" :ocupado="ocupado"
                     :arbol="detalle.arbol" :arbol-cortado="detalle.arbolCortado"
                     @editar="editando = true" @cancelar="cancelarEdicion" @guardar="guardarFicha" @eliminar="pedirEliminarFicha()"/>
              <div v-else class="sin-ficha">
                <p v-if="raizSel?.baseExiste">Esta carpeta principal no tiene ficha técnica todavía. En la ficha también se explican sus {{ detalle.arbol.length }} subcarpeta(s) y cuánto tiempo deben quedarse sus archivos.</p>
                <p v-else>Para registrar fichas técnicas, primero cree el Excel de fichas en su carpeta raíz.</p>
                <button v-if="raizSel?.baseExiste && !raizSel?.baseError" class="btn primario" @click="nuevaFicha"><Icono nombre="mas"/>Crear ficha técnica</button>
                <button v-else-if="!raizSel?.baseExiste" class="btn" @click="seleccionar({ruta: raizDeRuta(sel.ruta), esCarpeta: true})">Ir a la carpeta raíz</button>
              </div>
            </section>

            <!-- Subcarpeta: se explica dentro de la ficha de su carpeta principal -->
            <section v-else-if="!detalle.esRaiz" class="bloque">
              <div class="bloque-cab"><h2>Ficha técnica</h2></div>
              <div class="ficha-padre">
                <Icono nombre="carpeta-abierta"/>
                <div>
                  <p class="quien">Esta subcarpeta se describe en la ficha de <b>{{ nombreDe(detalle.principal) }}</b>.</p>
                  <p v-if="detalle.descripcion" class="desc">{{ detalle.descripcion }}</p>
                  <p v-else class="muted">{{ detalle.principalFicha ? 'Todavía no tiene explicación en esa ficha.' : 'Esa carpeta principal todavía no tiene ficha técnica.' }}</p>
                  <p v-if="detalle.retencion" class="ret-sub">Retención: <b>{{ textoRet(detalle.retencion) }}</b>
                    <span class="muted">({{ detalle.retencionDe === sel.ruta ? 'propia' : 'heredada de ' + nombreDe(detalle.retencionDe) }})</span></p>
                </div>
                <button class="btn" @click="seleccionar({ruta: detalle.principal, esCarpeta: true})">
                  {{ detalle.principalFicha ? (detalle.descripcion ? 'Ver la ficha' : 'Ir a la ficha y describirla') : 'Ir a la carpeta principal' }}
                </button>
              </div>
            </section>

            <!-- Raíz: estado de las fichas de las carpetas principales -->
            <section v-else-if="raizSel?.baseExiste && detalle.subcarpetas" class="bloque">
              <div class="bloque-cab">
                <h2>Fichas técnicas <span class="muted">{{ principales.filter((e) => e.ficha).length }} de {{ principales.length }} carpetas principales</span></h2>
              </div>
              <div class="principales">
                <button v-for="e in principales" :key="e.ruta" class="carp-principal" :class="{con: e.ficha}" @click="elegirArbol(e)" @contextmenu.prevent="abrirMenu($event, e)">
                  <Icono nombre="carpeta" class="ic carp"/>
                  <span class="nom">{{ e.nombre }}</span>
                  <span class="estado-ficha">{{ e.ficha ? 'Con ficha' : 'Sin ficha' }}</span>
                </button>
              </div>
            </section>

            <!-- Contenido -->
            <section class="bloque">
              <div class="bloque-cab">
                <h2>Contenido <span class="muted">{{ detalle.entradas.length }}</span></h2>
                <div class="tipos">
                  <span v-for="[t, v] in tiposEnCarpeta" :key="t" class="chip-tipo" :style="{'--c': v.color}">{{ t }} {{ v.n }}</span>
                </div>
                <label v-if="detalle.entradas.length > 8" class="filtro"><Icono nombre="buscar"/><input v-model="filtroTabla" placeholder="Filtrar en esta carpeta"/></label>
              </div>
              <div v-if="detalle.entradas.length" class="tabla-caja">
                <table class="tabla">
                  <thead>
                    <tr>
                      <th @click="ordenarPor('nombre')">Nombre <i>{{ flecha('nombre') }}</i></th>
                      <th @click="ordenarPor('tipo')">Tipo <i>{{ flecha('tipo') }}</i></th>
                      <th class="der" @click="ordenarPor('tamano')">Tamaño <i>{{ flecha('tamano') }}</i></th>
                      <th class="der" @click="ordenarPor('fecha')">Modificado <i>{{ flecha('fecha') }}</i></th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="e in entradasTabla" :key="e.ruta" class="fila-ent" @dblclick="e.esCarpeta ? elegirArbol(e) : abrirArchivo(e.ruta)" @contextmenu.prevent="abrirMenu($event, e)">
                      <td class="nom">
                        <button class="nombre-ent" @click="elegirArbol(e)">
                          <Icono v-if="e.esCarpeta" nombre="carpeta" class="ic carp"/>
                          <span v-else class="ic arch" :style="{'--c': tipoArchivo(e.ext).color}"><Icono nombre="archivo"/></span>
                          <span>{{ e.nombre }}</span>
                          <span v-if="e.ficha" class="punto" title="Tiene ficha técnica"></span>
                        </button>
                      </td>
                      <td class="tipo">{{ e.esCarpeta ? 'Carpeta' : tipoArchivo(e.ext).tipo }}</td>
                      <td class="der">{{ e.esCarpeta ? '' : tamano(e.tamano) }}</td>
                      <td class="der" :title="fechaHora(e.modificado)">{{ fecha(e.modificado) }}</td>
                      <td class="acc">
                        <button v-if="!e.esCarpeta" class="btn fantasma ic" title="Abrir" @click="abrirArchivo(e.ruta)"><Icono nombre="abrir"/></button>
                        <button class="btn fantasma ic" title="Renombrar" @click="pedirRenombrar(e)"><Icono nombre="editar"/></button>
                        <button class="btn fantasma ic" title="Mover" @click="pedirMover(e)"><Icono nombre="mover"/></button>
                        <button class="btn fantasma ic rojo" title="Eliminar" @click="pedirEliminar(e)"><Icono nombre="borrar"/></button>
                      </td>
                    </tr>
                    <tr v-if="!entradasTabla.length"><td colspan="5" class="muted">Nada coincide con “{{ filtroTabla }}”.</td></tr>
                  </tbody>
                </table>
              </div>
              <div v-else class="sin-ficha">
                <p>La carpeta está vacía.</p>
                <div class="fila-btn">
                  <button class="btn" @click="pedirNuevaCarpeta(sel.ruta)"><Icono nombre="carpeta-mas"/>Nueva subcarpeta</button>
                  <button class="btn" @click="agregarArchivos(sel.ruta)"><Icono nombre="subir"/>Agregar archivos</button>
                </div>
              </div>
            </section>
          </section>
        </template>
      </template>
    </main>

    <!-- Diálogos -->
    <Dialogo v-if="dialogo" v-bind="dialogo" :ocupado="ocupado" :bloquear="!!errorForm && !!dialogo.tipo"
             @confirmar="cerrarDialogo(true)" @cancelar="cerrarDialogo(false)">
      <div v-if="dialogo.tipo === 'nombre'" class="form-dialogo">
        <label>Nombre
          <input v-model="form.nombre" class="campo-input" :class="{malo: form.nombre && errorForm}" @keydown.enter="!errorForm && cerrarDialogo(true)"/>
          <small v-if="form.nombre && errorForm" class="err">{{ errorForm }}</small>
        </label>
      </div>
      <div v-else-if="dialogo.tipo === 'mover'" class="form-dialogo">
        <Selector v-model="form.destino" :colores="colores" :bloqueada="dialogo.elemento?.esCarpeta ? dialogo.elemento.ruta : ''"/>
        <p v-if="form.destino" class="destino">Destino: <code>{{ rutaVisible(form.destino) }}</code></p>
        <small v-if="form.destino && errorForm" class="err">{{ errorForm }}</small>
      </div>
      <div v-else-if="dialogo.tipo === 'color'" class="form-dialogo">
        <div class="paleta">
          <button v-for="c in estado.paleta" :key="c" class="muestra" :class="{on: form.color.toLowerCase() === c}" :style="{background: c}"
                  :aria-label="c" @click="form.color = c"></button>
          <label class="otro-color" title="Otro color"><input type="color" v-model="form.color"/>Otro…</label>
        </div>
        <p class="vista-color"><Icono nombre="carpeta" :style="{color: form.color}"/>Así se verán sus carpetas</p>
      </div>
    </Dialogo>

    <Menu v-if="menu" v-bind="menu" @cerrar="menu = null"/>
    <Ayuda v-if="verAyuda" @cerrar="verAyuda = false"/>

    <div class="avisos" aria-live="polite">
      <div v-for="a in avisos" :key="a.id" class="toast" :class="a.tipo">{{ a.texto }}</div>
    </div>
  </div>
</template>

<style>
.app { display: grid; grid-template-columns: 380px 1fr; height: 100vh; outline: none; }

/* ---- Sidebar ---- */
.sidebar { display: flex; flex-direction: column; min-height: 0; background: var(--riel); color: var(--riel-tinta); }
.marca { padding: 16px 16px 8px; display: grid; gap: 6px; }
.marca strong { color: #fff; font-size: var(--t-l); font-weight: 600; }
.agregar-raiz {
  display: flex; align-items: center; gap: 6px; width: fit-content; padding: 3px 8px; margin-left: -8px;
  border: 0; border-radius: var(--r-s); background: none; color: var(--riel-tenue); font-size: var(--t-xs); text-align: left;
}
.agregar-raiz:hover { background: rgba(255, 255, 255, .06); color: #fff; }
.agregar-raiz svg { width: 14px; height: 14px; flex: none; }
.vistas { display: grid; grid-template-columns: repeat(4, 1fr); gap: 3px; margin: 4px 12px 8px; padding: 3px; border-radius: var(--r-m); background: rgba(0, 0, 0, .2); }
.vistas button { display: flex; align-items: center; justify-content: center; gap: 5px; height: 30px; padding: 0 4px; border: 0; border-radius: var(--r-s); background: none; color: var(--riel-tinta); font-size: var(--t-xs); }
.vistas button svg { width: 14px; height: 14px; flex: none; }
.vistas button.activo { background: var(--ambar); color: #3d2a00; font-weight: 600; }
.vistas button:not(.activo):hover { background: rgba(255, 255, 255, .06); }
.buscador {
  display: flex; align-items: center; gap: 8px; margin: 0 12px 6px; padding: 0 10px;
  height: 34px; border-radius: var(--r-s); background: rgba(255, 255, 255, .07); border: 1px solid rgba(255, 255, 255, .08);
}
.buscador:focus-within { border-color: var(--ambar); }
.buscador svg { width: 15px; height: 15px; color: var(--riel-tenue); flex: none; }
.buscador input { flex: 1; min-width: 0; border: 0; background: none; color: #fff; outline: none; font-size: var(--t-s); }
.buscador input::placeholder { color: var(--riel-tenue); }
.limpiar { border: 0; background: none; color: var(--riel-tenue); padding: 0; display: flex; }
.opciones-arbol { display: flex; justify-content: space-between; align-items: center; padding: 0 14px 6px; font-size: var(--t-xs); color: var(--riel-tenue); }
.opciones-arbol input { accent-color: var(--ambar); vertical-align: -2px; }
.enlace-claro { border: 0; background: none; padding: 0; color: var(--riel-tenue); font-size: var(--t-xs); }
.enlace-claro:hover { color: #fff; text-decoration: underline; }
.lista { flex: 1; overflow: auto; padding-bottom: 12px; }
.nada { padding: 8px 16px; margin: 0; font-size: var(--t-xs); color: var(--riel-tenue); }
.fila-raiz { display: flex; align-items: center; gap: 8px; min-height: 30px; padding: 0 12px 0 10px; color: #fff; font-weight: 600; font-size: var(--t-s); cursor: pointer; border-left: 3px solid transparent; }
.fila-raiz svg { width: 16px; height: 16px; color: var(--riel-tinta); }
.fila-raiz small { margin-left: auto; font-weight: 400; font-size: var(--t-xs); color: var(--riel-tenue); }
.pista-arbol { margin: 10px 16px 0; font-size: 11px; color: #5f8590; }
.fila-raiz:hover { background: rgba(255, 255, 255, .05); }
.fila-raiz.activo { background: rgba(242, 179, 61, .16); border-left-color: var(--ambar); }
.resultado { display: flex; align-items: flex-start; gap: 8px; width: 100%; padding: 5px 12px; border: 0; border-left: 3px solid transparent; background: none; color: var(--riel-tinta); text-align: left; font-size: var(--t-s); }
.resultado:hover { background: rgba(255, 255, 255, .05); }
.resultado.activo { background: color-mix(in srgb, var(--cf) 18%, transparent); border-left-color: var(--cf); }
.resultado .txt { display: grid; min-width: 0; }
.resultado b { font-weight: 500; color: #fff; overflow-wrap: anywhere; }
.resultado small { color: var(--riel-tenue); font-size: var(--t-xs); overflow-wrap: anywhere; }
.ic { width: 16px; height: 16px; flex: none; }
.ic.carp { color: var(--cf, var(--ambar)); }
.ic.arch { display: inline-grid; place-items: center; color: var(--c); }
.ic.arch svg { width: 15px; height: 15px; }
.pie { padding: 10px 12px 12px; border-top: 1px solid rgba(255, 255, 255, .08); display: grid; gap: 8px; }
.btn.nueva { justify-content: center; background: var(--ambar); color: #3d2a00; border-color: var(--ambar); font-weight: 600; }
.btn.nueva:hover { background: #f5c25f; }
.botones { display: grid; grid-template-columns: repeat(4, auto); gap: 2px; justify-content: space-between; }
.btn.nueva:disabled { opacity: .5; }
.botones .btn { padding-left: 6px; padding-right: 6px; }
.btn.claro { color: var(--riel-tinta); }
.btn.claro:hover { background: rgba(255, 255, 255, .07); }

/* ---- Main ---- */
.principal { overflow-y: auto; min-width: 0; }
.principal.oscuro { overflow: hidden; background: #0d2730; }
.pantalla-vacia, .detalle { max-width: 1180px; margin: 0 auto; padding: 24px 40px 64px; }
.pantalla-vacia { display: grid; gap: 12px; justify-items: start; padding-top: 16vh; }
.pantalla-vacia p { color: var(--tinta-2); max-width: 64ch; margin: 0; }
.recientes { display: grid; gap: 4px; margin-top: 18px; }
.recientes p { margin: 0 0 4px; font-size: var(--t-s); }
.reciente { display: flex; align-items: center; gap: 8px; padding: 6px 10px; border: 1px solid var(--linea); border-radius: var(--r-s); background: var(--superficie); text-align: left; font-size: var(--t-s); }
.reciente:hover { border-color: var(--tinta-3); }
.reciente svg { width: 15px; height: 15px; color: var(--ambar-oscuro); }
h1 { margin: 0; font-size: var(--t-xxl); font-weight: 600; line-height: 1.15; letter-spacing: -.01em; overflow-wrap: anywhere; }
h2 { margin: 0; font-size: var(--t-l); font-weight: 600; }
code { font-size: .92em; padding: 0 4px; border-radius: 3px; background: var(--linea-2); }
.migas { display: flex; flex-wrap: wrap; align-items: center; gap: 2px; margin-bottom: 6px; font-size: var(--t-s); color: var(--tinta-3); min-height: 20px; }
.migas button { border: 0; background: none; padding: 1px 4px; border-radius: 3px; color: var(--tinta-2); font-size: var(--t-s); }
.migas button:hover { background: var(--linea-2); color: var(--tinta); }
.migas .sep { color: var(--linea); }
.cabecera { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; margin-bottom: 18px; flex-wrap: wrap; }
.cabecera .acciones { display: flex; gap: 6px; flex-wrap: wrap; }
.titulo-carp { display: flex; align-items: center; gap: 10px; min-width: 0; }
.ic-carp { width: 30px; height: 30px; color: var(--cf, var(--ambar-oscuro)); flex: none; }
.ruta-raiz { display: block; font-size: var(--t-xs); overflow-wrap: anywhere; }
.titulo-arch { display: flex; align-items: center; gap: 12px; min-width: 0; }
.ic-grande { display: grid; place-items: center; width: 48px; height: 48px; flex: none; border-radius: var(--r-m); background: color-mix(in srgb, var(--c) 14%, white); color: var(--c); }
.ic-grande svg { width: 28px; height: 28px; }
.chip-tipo { display: inline-block; margin-top: 4px; padding: 1px 8px; border-radius: 999px; font-size: var(--t-xs); font-weight: 600; color: var(--c); background: color-mix(in srgb, var(--c) 12%, white); }
.btn.ic { width: 32px; height: 32px; padding: 0; justify-content: center; }
.btn.ic.rojo:hover { color: var(--rojo); background: var(--rojo-claro); }
.cifras { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 1px; margin: 0 0 24px; background: var(--linea); border: 1px solid var(--linea); border-radius: var(--r-m); overflow: hidden; }
.cifras div { background: var(--superficie); padding: 12px 16px; }
.cifras dt { font-size: var(--t-s); color: var(--tinta-2); }
.cifras dd { margin: 2px 0 0; font-size: 26px; font-weight: 600; }
.cifras dd.pequeno { font-size: var(--t-l); display: grid; }
.cifras dd small { display: block; margin-top: 2px; font-size: var(--t-xs); font-weight: 400; color: var(--tinta-3); }
.cifras dd.si { color: var(--verde); }
/* Total que todavía se está sumando: el número sube en vivo */
.cifras dd.sumando { color: var(--foco); }
.cifras dd.sumando small::after { content: ''; display: inline-block; width: 8px; height: 8px; margin-left: 6px; border: 2px solid var(--foco); border-right-color: transparent; border-radius: 50%; animation: girar-c .8s linear infinite; vertical-align: -1px; }
@keyframes girar-c { to { transform: rotate(360deg); } }
.cifras dd.no { color: var(--tinta-3); }
.aviso { display: flex; align-items: center; gap: 12px; padding: 12px 16px; margin-bottom: 16px; background: var(--ambar-claro); border: 1px solid #f0d9a0; border-radius: var(--r-m); }
.aviso > svg { width: 22px; height: 22px; color: var(--ambar-oscuro); flex: none; }
.aviso > div { flex: 1; }
.aviso p { margin: 2px 0 0; color: var(--tinta-2); font-size: var(--t-s); }
.aviso.error { background: var(--rojo-claro); border-color: #efc2b9; }
.aviso.error > svg { color: var(--rojo); }
.sueltas { margin: 6px 0 0; padding-left: 18px; font-size: var(--t-s); display: grid; gap: 2px; }
.enlace { padding: 0; margin-left: 8px; border: 0; background: none; color: var(--foco); font-size: var(--t-xs); text-decoration: underline; }
.bloque { margin-bottom: 28px; }
.bloque-cab { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; flex-wrap: wrap; }
.bloque-cab h2 .muted { font-weight: 400; font-size: var(--t-s); margin-left: 4px; }
.tipos { display: flex; gap: 4px; flex-wrap: wrap; flex: 1; }
.tipos .chip-tipo { margin: 0; }
.filtro { display: flex; align-items: center; gap: 6px; height: 32px; padding: 0 10px; border: 1px solid var(--linea); border-radius: var(--r-s); background: #fff; }
.filtro svg { width: 14px; height: 14px; color: var(--tinta-3); }
.filtro input { border: 0; outline: none; font-size: var(--t-s); width: 180px; }
.pestanas { display: flex; gap: 2px; }
.pestanas button { padding: 4px 10px; border: 1px solid var(--linea); border-radius: var(--r-s); background: #fff; font-size: var(--t-s); }
.pestanas button.activo { background: var(--riel); color: #fff; border-color: var(--riel); }
.sin-ficha { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 16px 18px; border: 1px dashed var(--linea); border-radius: var(--r-m); background: var(--superficie); flex-wrap: wrap; }
.sin-ficha p { margin: 0; color: var(--tinta-2); }
.fila-btn { display: flex; gap: 8px; }
.tabla-caja { background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); overflow: auto; }
.tabla { width: 100%; border-collapse: collapse; font-size: var(--t-s); }
.tabla th { text-align: left; padding: 8px 12px; background: #f6f8f9; color: var(--tinta-2); font-weight: 600; border-bottom: 1px solid var(--linea); cursor: pointer; user-select: none; white-space: nowrap; position: sticky; top: 0; }
.tabla th i { font-style: normal; font-size: 10px; color: var(--tinta-3); }
.tabla td { padding: 4px 12px; border-bottom: 1px solid var(--linea-2); vertical-align: middle; }
.tabla tr:last-child td { border-bottom: 0; }
.tabla tbody tr:hover { background: #f8fafb; }
.tabla .der { text-align: right; white-space: nowrap; }
.tabla .tipo { color: var(--tinta-2); white-space: nowrap; }
.tabla .acc { white-space: nowrap; text-align: right; width: 1%; }
.tabla .acc .btn.ic { width: 28px; height: 28px; opacity: .55; }
.tabla tr:hover .acc .btn.ic { opacity: 1; }
.nombre-ent { display: flex; align-items: center; gap: 8px; padding: 4px 0; border: 0; background: none; text-align: left; font-size: var(--t-s); color: var(--tinta); }
.nombre-ent:hover span:not(.punto) { text-decoration: underline; }
.nombre-ent span { overflow-wrap: anywhere; }
.punto { width: 7px; height: 7px; border-radius: 50%; background: var(--cf, var(--ambar)); flex: none; }
.ficha-padre { display: flex; align-items: center; gap: 14px; padding: 16px 18px; background: var(--superficie); border: 1px solid var(--linea); border-left: 4px solid var(--cf, var(--ambar)); border-radius: var(--r-m); }
.ficha-padre > svg { width: 26px; height: 26px; color: var(--cf, var(--ambar-oscuro)); flex: none; }
.ficha-padre .ret-sub { margin-top: 6px; font-size: var(--t-s); }
.ficha-padre > div { flex: 1; min-width: 0; }
.ficha-padre p { margin: 0; }
.ficha-padre .quien { font-size: var(--t-s); color: var(--tinta-2); }
.ficha-padre .desc { margin-top: 4px; white-space: pre-wrap; }
.principales { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 8px; }
.carp-principal { display: flex; align-items: center; gap: 8px; padding: 10px 12px; border: 1px solid var(--linea); border-radius: var(--r-m); background: var(--superficie); text-align: left; font-size: var(--t-s); }
.carp-principal:hover { border-color: var(--tinta-3); }
.carp-principal .nom { flex: 1; min-width: 0; overflow-wrap: anywhere; font-weight: 500; }
.estado-ficha { font-size: var(--t-xs); padding: 1px 8px; border-radius: 999px; background: var(--linea-2); color: var(--tinta-3); white-space: nowrap; }
.carp-principal.con .estado-ficha { background: var(--verde-claro); color: var(--verde); font-weight: 600; }
.tarjeta-arch { display: grid; gap: 18px; padding: 20px 22px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); border-left: 6px solid var(--cf, var(--ambar)); }
.estado-ret { display: inline-block; margin-left: 6px; padding: 0 8px; border-radius: 99px; font-size: var(--t-xs); background: var(--verde-claro); color: var(--verde); }
.estado-ret.vencido { background: var(--rojo-claro); color: var(--rojo); font-weight: 600; }
.raices { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 10px; margin-bottom: 24px; }
.tarjeta-raiz { display: grid; gap: 6px; padding: 12px 14px; background: var(--superficie); border: 1px solid var(--linea); border-top: 4px solid var(--cf); border-radius: var(--r-m); }
.cab-raiz { display: flex; align-items: center; gap: 10px; border: 0; background: none; padding: 0; text-align: left; }
.cab-raiz:hover b { text-decoration: underline; }
.cab-raiz span { display: grid; min-width: 0; }
.cab-raiz b { font-size: var(--t-l); font-weight: 600; overflow-wrap: anywhere; }
.cab-raiz small { font-size: var(--t-xs); color: var(--tinta-3); overflow-wrap: anywhere; }
.ic-r { width: 28px; height: 28px; color: var(--cf); flex: none; }
.estado-r { margin: 0; font-size: var(--t-s); color: var(--tinta-2); }
.estado-r.ok { color: var(--verde); }
.estado-r.error { color: var(--rojo); }
.acc-raiz { display: flex; gap: 2px; justify-content: flex-end; }
.paleta { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.muestra { width: 30px; height: 30px; border-radius: 50%; border: 3px solid #fff; box-shadow: 0 0 0 1px var(--linea); }
.muestra.on { box-shadow: 0 0 0 2px var(--tinta); }
.otro-color { display: flex !important; align-items: center; gap: 6px; font-size: var(--t-s); }
.otro-color input { width: 34px; height: 30px; padding: 0; border: 0; background: none; }
.vista-color { display: flex; align-items: center; gap: 8px; margin: 0; font-size: var(--t-s); color: var(--tinta-2); }
.vista-color svg { width: 26px; height: 26px; }
.tarjeta-arch dl { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 14px 24px; margin: 0; }
.tarjeta-arch dt { font-size: var(--t-xs); color: var(--tinta-3); }
.tarjeta-arch dd { margin: 2px 0 0; font-size: var(--t-m); }
.tarjeta-arch .ancho { grid-column: 1 / -1; }
.tarjeta-arch .ruta { font-size: var(--t-s); color: var(--tinta-2); overflow-wrap: anywhere; }
.acciones-arch { display: flex; gap: 8px; flex-wrap: wrap; }
.form-dialogo { display: grid; gap: 10px; margin-top: 14px; }
.form-dialogo label { display: grid; gap: 4px; font-size: var(--t-s); color: var(--tinta-2); }
.form-dialogo .err { color: var(--rojo); font-size: var(--t-xs); }
.form-dialogo .destino { margin: 0; font-size: var(--t-s); }
.campo-input.malo { border-color: var(--rojo); }
.avisos { position: fixed; right: 20px; bottom: 20px; display: grid; gap: 8px; z-index: 60; }
.toast { max-width: 440px; padding: 10px 14px; border-radius: var(--r-m); background: var(--riel); color: #fff; font-size: var(--t-s); box-shadow: 0 8px 24px rgba(16, 49, 59, .25); border-left: 4px solid var(--verde); }
.toast.error { border-left-color: var(--rojo); background: #3a1d17; }
</style>
