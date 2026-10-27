<script setup>
import {computed, nextTick, onMounted, reactive, ref, watch} from 'vue'
import Arbol from './components/Arbol.vue'
import Ficha from './components/Ficha.vue'
import Dialogo from './components/Dialogo.vue'
import Icono from './components/Icono.vue'
import Selector from './components/Selector.vue'
import Grafo from './components/Grafo.vue'
import Ayuda from './components/Ayuda.vue'
import {
  AbrirArchivo, AbrirBase, AbrirPapelera, AgregarArchivos, Archivo, Buscar, Carpeta, ContarEliminar, CrearBase,
  CrearCarpeta, ElegirRaiz, Eliminar, EliminarRegistro, Estado, GuardarRegistro, Listar, MostrarEnCarpeta, Mover,
  Renombrar, UsarRaiz,
} from '../wailsjs/go/main/App'
import {errorNombre, fecha, fechaHora, hace, mensajeError, nombreDe, padreDe, rutaWin, tamano, tamanoLargo, tipoArchivo} from './util'

/* ---------- Estado ---------- */
const estado = ref(null)
const cargando = ref(true)
const ocupado = ref(false)
const avisos = ref([])
const dialogo = ref(null)
const verAyuda = ref(false)
const vista = ref('carpeta')          // 'carpeta' | 'grafo'
const grafoDesde = ref('')

// Árbol (por demanda)
const hijos = reactive({})            // ruta -> entradas
const abiertos = reactive({})
const cargandoDir = reactive({})
const limites = reactive({})
const conArchivos = ref(true)

// Selección: carpeta o archivo
const sel = ref({ruta: '', esCarpeta: true})
const detalle = ref(null)             // DetalleCarpeta
const archivo = ref(null)             // DetalleArchivo
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
const form = reactive({nombre: '', destino: null, resumen: null})

function avisar(texto, tipo = 'ok') {
  const id = Math.random()
  avisos.value.push({id, texto, tipo})
  setTimeout(() => (avisos.value = avisos.value.filter((a) => a.id !== id)), tipo === 'error' ? 7000 : 3500)
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

/* ---------- Carga ---------- */
async function recargarEstado() {
  try {
    estado.value = await Estado()
  } catch (e) {
    avisar(mensajeError(e), 'error')
    estado.value = {raiz: '', recientes: [], campos: [], sinCarpeta: []}
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
async function reiniciar() {
  for (const k of Object.keys(hijos)) delete hijos[k]
  for (const k of Object.keys(abiertos)) delete abiertos[k]
  for (const k of Object.keys(limites)) delete limites[k]
  resultados.value = null
  busqueda.value = ''
  cargando.value = true
  await recargarEstado()
  if (estado.value?.raiz) {
    await cargarDir('')
    await seleccionar({ruta: '', esCarpeta: true})
  }
  cargando.value = false
}
onMounted(reiniciar)

async function recargarTodo() {
  const abiertas = Object.keys(hijos)
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
  } catch (err) {
    avisar(mensajeError(err), 'error')
  }
}
async function recargarSeleccion() {
  const s = sel.value
  try {
    if (s.esCarpeta) detalle.value = await Carpeta(s.ruta)
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

/* ---------- Búsqueda ---------- */
watch(busqueda, (q) => {
  clearTimeout(tBusqueda)
  if (q.trim().length < 2) { resultados.value = null; buscando.value = false; return }
  buscando.value = true
  tBusqueda = setTimeout(async () => {
    try {
      const r = await Buscar(q)
      if (busqueda.value === q) resultados.value = r
    } catch (e) {
      avisar(mensajeError(e), 'error')
    } finally {
      if (busqueda.value === q) buscando.value = false
    }
  }, 400)
})

/* ---------- Derivados ---------- */
const nombreRaiz = computed(() => estado.value?.nombreRaiz || 'Raíz')
const registros = computed(() => detalle.value?.registros || [])
const ficha = computed(() => registros.value[fichaActiva.value] || null)
const migas = computed(() => {
  const r = sel.value.ruta
  const partes = r ? r.split('/') : []
  return [{nombre: nombreRaiz.value, ruta: ''}, ...partes.map((p, i) => ({nombre: p, ruta: partes.slice(0, i + 1).join('/')}))]
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
const principales = computed(() => (sel.value.ruta === '' && detalle.value ? detalle.value.entradas.filter((e) => e.esCarpeta) : []))
const tiposEnCarpeta = computed(() => {
  const m = new Map()
  for (const e of detalle.value?.entradas || []) if (!e.esCarpeta) { const t = tipoArchivo(e.ext); m.set(t.tipo, {n: (m.get(t.tipo)?.n || 0) + 1, color: t.color}) }
  return [...m.entries()].sort((a, b) => b[1].n - a[1].n)
})

/* ---------- Acciones de archivos ---------- */
const abrirArchivo = (r) => ejecutar(() => AbrirArchivo(r))
const mostrar = (r) => ejecutar(() => MostrarEnCarpeta(r))

function pedirNuevaCarpeta(padre = null) {
  const p = padre ?? (sel.value.esCarpeta ? sel.value.ruta : padreDe(sel.value.ruta))
  form.nombre = ''
  form.destino = p
  dialogo.value = {
    titulo: 'Nueva carpeta', tipo: 'nombre', textoConfirmar: 'Crear carpeta',
    mensaje: `Se crea dentro de “${p ? nombreDe(p) : nombreRaiz.value}”.`,
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
    ok: async () => {
      const nueva = await ejecutar(() => Mover(e.ruta, form.destino), 'Movido a ' + (form.destino ? nombreDe(form.destino) : nombreRaiz.value))
      if (nueva && nueva !== true) await despuesDeMover(e, nueva, form.destino)
    },
  }
}
async function despuesDeMover(e, nueva, destino = null) {
  const viejaSel = sel.value.ruta
  // Las rutas guardadas en el árbol bajo la carpeta movida ya no sirven.
  for (const k of Object.keys(hijos)) if (k === e.ruta || k.startsWith(e.ruta + '/')) delete hijos[k]
  for (const k of Object.keys(abiertos)) if (k === e.ruta || k.startsWith(e.ruta + '/')) delete abiertos[k]
  await refrescarDirs([e.ruta, nueva, ...(destino !== null ? [destino] : [])])
  await recargarEstado()
  if (viejaSel === e.ruta || viejaSel.startsWith(e.ruta + '/')) await seleccionar({ruta: nueva + viejaSel.slice(e.ruta.length), esCarpeta: sel.value.esCarpeta}, false)
  else await recargarSeleccion()
}
async function pedirEliminar(e) {
  const r = e.esCarpeta ? await ejecutar(() => ContarEliminar(e.ruta)) : {carpetas: 0, archivos: 0, fichas: 0}
  if (!r) return
  const contenido = r.carpetas + r.archivos
  dialogo.value = {
    titulo: e.esCarpeta ? 'Eliminar carpeta' : 'Eliminar archivo', peligro: true, textoConfirmar: 'Eliminar',
    mensaje: `“${nombreDe(e.ruta)}” se moverá a la papelera de la aplicación (${estado.value.nombrePapelera}, en la raíz).`,
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
        for (const k of Object.keys(hijos)) if (k === e.ruta || k.startsWith(e.ruta + '/')) delete hijos[k]
        await refrescarDirs([e.ruta])
        await recargarEstado()
        if (sel.value.ruta === e.ruta || sel.value.ruta.startsWith(e.ruta + '/')) await seleccionar({ruta: padreDe(e.ruta), esCarpeta: true})
        else await recargarSeleccion()
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
    mensaje: `Se borra la ficha de “${rutaWin(r.ruta) || nombreRaiz.value}” del Excel. La carpeta no se toca.`,
    detalles: ['Los datos quedan copiados en la hoja Historial del Excel.'],
    ok: async () => {
      if (!r.id) { cancelarEdicion(); return }
      if (await ejecutar(() => EliminarRegistro(r.id), 'Ficha eliminada')) {
        fichaActiva.value = 0
        await recargarEstado()
        await refrescarDirs([r.ruta])
        await recargarSeleccion()
      }
    },
  }
}

/* ---------- Raíz y Excel ---------- */
async function cambiarRaiz() {
  if (await ejecutar(() => ElegirRaiz())) await reiniciar()
}
async function usarReciente(r) {
  if (await ejecutar(() => UsarRaiz(r))) await reiniciar()
}
async function crearBase() {
  if (await ejecutar(() => CrearBase(), 'Excel de fichas creado en la raíz')) { await recargarEstado(); await recargarSeleccion() }
}
const abrirExcel = () => ejecutar(() => AbrirBase())
const abrirPapelera = () => ejecutar(() => AbrirPapelera())

/* ---------- Vista gráfica ---------- */
function verGrafo(desde = null) {
  grafoDesde.value = desde ?? (sel.value.esCarpeta ? sel.value.ruta : padreDe(sel.value.ruta))
  vista.value = 'grafo'
}
async function elegirDesdeGrafo(n) {
  await abrirPadres(n.ruta)
  await seleccionar({ruta: n.ruta, esCarpeta: n.esCarpeta})
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
    if (form.destino === null) return 'Elija la carpeta destino'
    if (d.elemento && padreDe(d.elemento.ruta) === form.destino) return 'Ya está en esa carpeta'
  }
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
  if (dialogo.value || verAyuda.value) return
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') { e.preventDefault(); document.getElementById('buscar')?.focus(); return }
  const enCampo = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName)
  if (e.key === 'F2' && !enCampo && sel.value.ruta) { e.preventDefault(); pedirRenombrar(sel.value) }
  if (e.key === 'F5') { e.preventDefault(); recargarTodo() }
}
watch(dialogo, async (d) => {
  if (d?.tipo === 'nombre') { await nextTick(); const i = document.querySelector('.form-dialogo input'); i?.focus(); i?.select() }
})
</script>

<template>
  <div class="app" @keydown="teclas" tabindex="-1">
    <!-- ================= SIDEBAR ================= -->
    <aside class="sidebar">
      <div class="marca">
        <strong>Organizador de carpetas</strong>
        <button v-if="estado?.raiz" class="carpeta-raiz" :title="estado.raiz + ' — clic para cambiar'" @click="cambiarRaiz">
          <Icono nombre="carpeta"/><span>{{ estado.raiz }}</span>
        </button>
      </div>

      <template v-if="estado?.raiz">
        <div class="vistas" role="tablist">
          <button role="tab" :aria-selected="vista === 'carpeta'" :class="{activo: vista === 'carpeta'}" @click="vista = 'carpeta'"><Icono nombre="lista"/>Carpetas</button>
          <button role="tab" :aria-selected="vista === 'grafo'" :class="{activo: vista === 'grafo'}" @click="verGrafo()"><Icono nombre="grafo"/>Vista gráfica</button>
        </div>
        <label class="buscador">
          <Icono nombre="buscar"/>
          <input id="buscar" v-model="busqueda" placeholder="Buscar carpetas y archivos" autocomplete="off"/>
          <button v-if="busqueda" class="limpiar" aria-label="Limpiar búsqueda" @click="busqueda = ''"><Icono nombre="cerrar"/></button>
        </label>
        <div class="opciones-arbol">
          <label><input type="checkbox" v-model="conArchivos"/> Mostrar archivos</label>
          <button class="enlace-claro" title="Cerrar todas las carpetas" @click="Object.keys(abiertos).forEach((k) => delete abiertos[k])">Contraer todo</button>
        </div>

        <nav class="lista" aria-label="Carpetas y archivos">
          <!-- Resultados de búsqueda -->
          <template v-if="busqueda.trim().length >= 2">
            <p v-if="buscando" class="nada">Buscando en todas las carpetas…</p>
            <template v-else-if="resultados">
              <p class="nada">{{ resultados.entradas.length }} resultado(s){{ resultados.cortada ? ' (se muestran los primeros)' : '' }}</p>
              <button v-for="e in resultados.entradas" :key="e.ruta" class="resultado" :class="{activo: sel.ruta === e.ruta}" @click="seleccionar(e)">
                <Icono v-if="e.esCarpeta" nombre="carpeta" class="ic carp"/>
                <span v-else class="ic arch" :style="{'--c': tipoArchivo(e.ext).color}"><Icono nombre="archivo"/></span>
                <span class="txt"><b>{{ e.nombre }}</b><small>{{ rutaWin(padreDe(e.ruta)) || nombreRaiz }}</small></span>
              </button>
            </template>
          </template>
          <!-- Árbol -->
          <template v-else>
            <div class="fila-raiz" :class="{activo: sel.ruta === ''}" @click="seleccionar({ruta: '', esCarpeta: true})">
              <Icono nombre="casa"/><span>{{ nombreRaiz }}</span>
            </div>
            <p v-if="cargandoDir[''] && !hijos['']" class="nada">Leyendo…</p>
            <Arbol v-else :entradas="hijos[''] || []" :hijos="hijos" :abiertos="abiertos" :cargando="cargandoDir" :limites="limites"
                   :seleccion="sel.ruta" :con-archivos="conArchivos" ruta="" @elegir="elegirArbol" @alternar="alternar" @mas="verMas"/>
          </template>
        </nav>

        <footer class="pie">
          <button class="btn nueva" @click="pedirNuevaCarpeta()"><Icono nombre="carpeta-mas"/>Nueva carpeta</button>
          <div class="botones">
            <button class="btn fantasma claro" title="Volver a leer (F5)" @click="recargarTodo"><Icono nombre="recargar"/>Recargar</button>
            <button class="btn fantasma claro" :disabled="!estado?.baseExiste" @click="abrirExcel"><Icono nombre="excel"/>Abrir Excel</button>
            <button class="btn fantasma claro" :disabled="!estado?.papeleraHay" title="Lo eliminado desde la aplicación" @click="abrirPapelera"><Icono nombre="papelera"/>Papelera</button>
            <button class="btn fantasma claro" title="Cómo usar la aplicación (F1)" @click="verAyuda = true"><Icono nombre="ayuda"/>Ayuda</button>
          </div>
        </footer>
      </template>
    </aside>

    <!-- ================= MAIN ================= -->
    <main class="principal" :class="{oscuro: vista === 'grafo' && estado?.raiz}">
      <!-- Sin raíz -->
      <section v-if="!cargando && !estado?.raiz" class="pantalla-vacia">
        <h1>Elija la carpeta que quiere organizar</h1>
        <p>Puede ser una carpeta local o de red, por ejemplo <code>Z:\MANTENIMIENTO\PRIVADA\14. Dirección Mtto</code>. En esa carpeta se guardará el Excel de fichas técnicas.</p>
        <button class="btn primario" @click="cambiarRaiz"><Icono nombre="carpeta"/>Elegir carpeta</button>
        <div v-if="estado?.recientes?.length" class="recientes">
          <p class="muted">Usadas recientemente</p>
          <button v-for="r in estado.recientes" :key="r" class="reciente" @click="usarReciente(r)"><Icono nombre="carpeta"/>{{ r }}</button>
        </div>
      </section>

      <!-- Vista gráfica -->
      <Grafo v-else-if="vista === 'grafo' && estado?.raiz" :desde="grafoDesde" :nombre-raiz="nombreRaiz" :seleccion="sel.ruta"
             @elegir="elegirDesdeGrafo" @abrir="abrirArchivo" @desde="(r) => (grafoDesde = r)"/>

      <!-- Archivo -->
      <section v-else-if="archivo && !sel.esCarpeta" class="detalle">
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
        <div class="tarjeta-arch">
          <dl>
            <div><dt>Tamaño</dt><dd>{{ tamanoLargo(archivo.archivo.tamano) }}</dd></div>
            <div><dt>Modificado</dt><dd>{{ fechaHora(archivo.archivo.modificado) }} <span class="muted">({{ hace(archivo.archivo.modificado) }})</span></dd></div>
            <div v-if="archivo.creado"><dt>Creado</dt><dd>{{ fechaHora(archivo.creado) }}</dd></div>
            <div class="ancho"><dt>Ubicación</dt><dd class="ruta">{{ estado.raiz }}\{{ rutaWin(archivo.archivo.ruta) }}</dd></div>
          </dl>
          <div class="acciones-arch">
            <button class="btn primario" @click="abrirArchivo(archivo.archivo.ruta)"><Icono nombre="abrir"/>Abrir</button>
            <button class="btn" @click="mostrar(archivo.archivo.ruta)"><Icono nombre="carpeta"/>Mostrar en el Explorador</button>
            <button class="btn fantasma" @click="seleccionar({ruta: padreDe(archivo.archivo.ruta), esCarpeta: true})"><Icono nombre="carpeta-abierta"/>Ver la carpeta que lo contiene</button>
          </div>
        </div>
      </section>

      <!-- Carpeta -->
      <section v-else-if="detalle" class="detalle">
        <nav class="migas" aria-label="Ubicación">
          <template v-for="(m, i) in migas" :key="m.ruta">
            <button v-if="i < migas.length - 1" @click="seleccionar({ruta: m.ruta, esCarpeta: true})">{{ m.nombre }}</button>
            <span v-if="i < migas.length - 1" class="sep">\</span>
          </template>
        </nav>
        <header class="cabecera">
          <div class="titulo-carp">
            <Icono :nombre="sel.ruta ? 'carpeta-abierta' : 'casa'" class="ic-carp"/>
            <h1>{{ detalle.carpeta.nombre }}</h1>
          </div>
          <div class="acciones">
            <button class="btn" @click="pedirNuevaCarpeta(sel.ruta)"><Icono nombre="carpeta-mas"/>Nueva subcarpeta</button>
            <button class="btn" @click="agregarArchivos(sel.ruta)"><Icono nombre="subir"/>Agregar archivos</button>
            <button class="btn" @click="verGrafo(sel.ruta)"><Icono nombre="grafo"/>Ver en gráfico</button>
            <template v-if="sel.ruta">
              <button class="btn" title="F2" @click="pedirRenombrar(sel)"><Icono nombre="editar"/>Renombrar</button>
              <button class="btn" @click="pedirMover(sel)"><Icono nombre="mover"/>Mover</button>
              <button class="btn peligro" @click="pedirEliminar(sel)"><Icono nombre="borrar"/>Eliminar</button>
            </template>
            <button class="btn fantasma ic" title="Mostrar en el Explorador" @click="mostrar(sel.ruta)"><Icono nombre="abrir"/></button>
          </div>
        </header>

        <!-- Avisos de la raíz -->
        <template v-if="!sel.ruta">
          <div v-if="!estado.baseExiste" class="aviso">
            <Icono nombre="excel"/>
            <div>
              <strong>Esta carpeta todavía no tiene Excel de fichas técnicas.</strong>
              <p>Se creará <code>{{ estado.nombreExcel }}</code> aquí, en la raíz. Las fichas de todas las carpetas y subcarpetas se guardan en él.</p>
            </div>
            <button class="btn primario" @click="crearBase">Crear Excel de fichas</button>
          </div>
          <div v-else-if="estado.baseError" class="aviso error"><Icono nombre="alerta"/><div>{{ estado.baseError }}</div></div>
          <div v-if="estado.noPrincipal?.length" class="aviso">
            <Icono nombre="alerta"/>
            <div>
              <strong>{{ estado.noPrincipal.length }} ficha(s) no están en una carpeta principal.</strong>
              <p>Las fichas técnicas solo van en las carpetas que están en la raíz; las subcarpetas se explican dentro de esas fichas. Puede eliminarlas (quedan copiadas en el Historial del Excel).</p>
              <ul class="sueltas">
                <li v-for="r in estado.noPrincipal" :key="r.id"><code>{{ rutaWin(r.ruta) || '(raíz)' }}</code>
                  <button class="enlace" @click="pedirEliminarFicha(r)">Eliminar ficha</button></li>
              </ul>
            </div>
          </div>
          <div v-if="estado.sinCarpeta?.length" class="aviso">
            <Icono nombre="alerta"/>
            <div>
              <strong>{{ estado.sinCarpeta.length }} ficha(s) del Excel apuntan a carpetas que ya no existen.</strong>
              <p>Pudieron renombrarse o moverse por fuera de la aplicación. Corrija la ruta en el Excel o elimine la ficha.</p>
              <ul class="sueltas">
                <li v-for="r in estado.sinCarpeta" :key="r.id"><code>{{ rutaWin(r.ruta) }}</code>
                  <button class="enlace" @click="pedirEliminarFicha(r)">Eliminar ficha</button></li>
              </ul>
            </div>
          </div>
        </template>

        <dl class="cifras">
          <div><dt>Subcarpetas</dt><dd>{{ detalle.subcarpetas }}</dd></div>
          <div><dt>Archivos</dt><dd>{{ detalle.archivos }}</dd></div>
          <div><dt>Tamaño de los archivos</dt><dd>{{ tamanoLargo(detalle.tamanoTotal) }}</dd></div>
          <div><dt>Modificada</dt><dd class="pequeno">{{ fecha(detalle.carpeta.modificado) }}<small>{{ hace(detalle.carpeta.modificado) }}</small></dd></div>
          <div v-if="detalle.esPrincipal"><dt>Ficha técnica</dt><dd class="pequeno" :class="detalle.carpeta.ficha ? 'si' : 'no'">{{ detalle.carpeta.ficha ? 'Sí' : 'No tiene' }}</dd></div>
          <div v-else-if="sel.ruta"><dt>Explicada en la ficha</dt><dd class="pequeno" :class="detalle.descripcion ? 'si' : 'no'">{{ detalle.descripcion ? 'Sí' : 'No' }}</dd></div>
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
          <Ficha v-if="ficha" :registro="ficha" :campos="estado.campos" :editando="editando" :ocupado="ocupado"
                 :arbol="detalle.arbol" :arbol-cortado="detalle.arbolCortado"
                 @editar="editando = true" @cancelar="cancelarEdicion" @guardar="guardarFicha" @eliminar="pedirEliminarFicha()"/>
          <div v-else class="sin-ficha">
            <p v-if="estado.baseExiste">Esta carpeta principal no tiene ficha técnica todavía. En la ficha también se explican sus {{ detalle.arbol.length }} subcarpeta(s).</p>
            <p v-else>Para registrar fichas técnicas, primero cree el Excel de fichas en la raíz.</p>
            <button v-if="estado.baseExiste && !estado.baseError" class="btn primario" @click="nuevaFicha"><Icono nombre="mas"/>Crear ficha técnica</button>
            <button v-else-if="!estado.baseExiste" class="btn" @click="seleccionar({ruta: '', esCarpeta: true})">Ir a la raíz</button>
          </div>
        </section>

        <!-- Subcarpeta: se explica dentro de la ficha de su carpeta principal -->
        <section v-else-if="sel.ruta" class="bloque">
          <div class="bloque-cab"><h2>Ficha técnica</h2></div>
          <div class="ficha-padre">
            <Icono nombre="carpeta-abierta"/>
            <div>
              <p class="quien">Esta subcarpeta se describe en la ficha de <b>{{ detalle.principal }}</b>.</p>
              <p v-if="detalle.descripcion" class="desc">{{ detalle.descripcion }}</p>
              <p v-else class="muted">{{ detalle.principalFicha ? 'Todavía no tiene explicación en esa ficha.' : 'Esa carpeta principal todavía no tiene ficha técnica.' }}</p>
            </div>
            <button class="btn" @click="seleccionar({ruta: detalle.principal, esCarpeta: true})">
              {{ detalle.principalFicha ? (detalle.descripcion ? 'Ver la ficha' : 'Ir a la ficha y describirla') : 'Ir a la carpeta principal' }}
            </button>
          </div>
        </section>

        <!-- Raíz: estado de las fichas de las carpetas principales -->
        <section v-else-if="estado.baseExiste && detalle.subcarpetas" class="bloque">
          <div class="bloque-cab">
            <h2>Fichas técnicas <span class="muted">{{ principales.filter((e) => e.ficha).length }} de {{ principales.length }} carpetas principales</span></h2>
          </div>
          <div class="principales">
            <button v-for="e in principales" :key="e.ruta" class="carp-principal" :class="{con: e.ficha}" @click="elegirArbol(e)">
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
                <tr v-for="e in entradasTabla" :key="e.ruta" class="fila-ent" @dblclick="e.esCarpeta ? elegirArbol(e) : abrirArchivo(e.ruta)">
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
        <Selector v-model="form.destino" :nombre-raiz="nombreRaiz" :bloqueada="dialogo.elemento?.esCarpeta ? dialogo.elemento.ruta : ''"/>
        <p v-if="form.destino !== null" class="destino">Destino: <code>{{ form.destino ? rutaWin(form.destino) : nombreRaiz }}</code></p>
        <small v-if="form.destino !== null && errorForm" class="err">{{ errorForm }}</small>
      </div>
    </Dialogo>

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
.carpeta-raiz {
  display: flex; align-items: center; gap: 6px; max-width: 100%; padding: 3px 6px; margin-left: -6px;
  border: 0; border-radius: var(--r-s); background: none; color: var(--riel-tenue); font-size: var(--t-xs); text-align: left;
}
.carpeta-raiz span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; direction: rtl; text-align: left; }
.carpeta-raiz:hover { background: rgba(255, 255, 255, .06); color: var(--riel-tinta); }
.carpeta-raiz svg { width: 13px; height: 13px; flex: none; }
.vistas { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; margin: 4px 12px 8px; padding: 3px; border-radius: var(--r-m); background: rgba(0, 0, 0, .2); }
.vistas button { display: flex; align-items: center; justify-content: center; gap: 6px; height: 30px; border: 0; border-radius: var(--r-s); background: none; color: var(--riel-tinta); font-size: var(--t-s); }
.vistas button svg { width: 15px; height: 15px; }
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
.fila-raiz svg { width: 16px; height: 16px; color: var(--ambar); }
.fila-raiz:hover { background: rgba(255, 255, 255, .05); }
.fila-raiz.activo { background: rgba(242, 179, 61, .16); border-left-color: var(--ambar); }
.resultado { display: flex; align-items: flex-start; gap: 8px; width: 100%; padding: 5px 12px; border: 0; border-left: 3px solid transparent; background: none; color: var(--riel-tinta); text-align: left; font-size: var(--t-s); }
.resultado:hover { background: rgba(255, 255, 255, .05); }
.resultado.activo { background: rgba(242, 179, 61, .16); border-left-color: var(--ambar); }
.resultado .txt { display: grid; min-width: 0; }
.resultado b { font-weight: 500; color: #fff; overflow-wrap: anywhere; }
.resultado small { color: var(--riel-tenue); font-size: var(--t-xs); overflow-wrap: anywhere; }
.ic { width: 16px; height: 16px; flex: none; }
.ic.carp { color: var(--ambar); }
.ic.arch { display: inline-grid; place-items: center; color: var(--c); }
.ic.arch svg { width: 15px; height: 15px; }
.pie { padding: 10px 12px 12px; border-top: 1px solid rgba(255, 255, 255, .08); display: grid; gap: 8px; }
.btn.nueva { justify-content: center; background: var(--ambar); color: #3d2a00; border-color: var(--ambar); font-weight: 600; }
.btn.nueva:hover { background: #f5c25f; }
.botones { display: grid; grid-template-columns: repeat(4, auto); gap: 2px; justify-content: space-between; }
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
.ic-carp { width: 30px; height: 30px; color: var(--ambar-oscuro); flex: none; }
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
.cifras dd small { font-size: var(--t-xs); font-weight: 400; color: var(--tinta-3); }
.cifras dd.si { color: var(--verde); }
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
.punto { width: 7px; height: 7px; border-radius: 50%; background: var(--ambar); flex: none; }
.ficha-padre { display: flex; align-items: center; gap: 14px; padding: 16px 18px; background: var(--superficie); border: 1px solid var(--linea); border-left: 4px solid var(--ambar); border-radius: var(--r-m); }
.ficha-padre > svg { width: 26px; height: 26px; color: var(--ambar-oscuro); flex: none; }
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
.tarjeta-arch { display: grid; gap: 18px; padding: 20px 22px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); border-left: 6px solid var(--ambar); }
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
