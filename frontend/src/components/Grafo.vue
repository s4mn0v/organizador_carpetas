<script setup>
import {computed, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {forceCenter, forceCollide, forceLink, forceManyBody, forceSimulation, forceX, forceY} from 'd3-force'
import Icono from './Icono.vue'
import {Grafo as LeerGrafo} from '../../wailsjs/go/main/App'
import {COLOR_CARPETA, mensajeError, nombreDe, rutaWin, sinTildes, tamano, tipoArchivo} from '../util'

// Vista gráfica (estilo Obsidian): cada carpeta y archivo es un punto unido a la carpeta que lo contiene.
const props = defineProps({
  desde: {type: String, default: ''},        // carpeta desde la que se dibuja
  nombreRaiz: {type: String, default: 'Raíz'},
  seleccion: {type: String, default: null},
})
const emit = defineEmits(['elegir', 'abrir', 'desde'])

const lienzo = ref(null)
const caja = ref(null)
const profundidad = ref(3)
const conArchivos = ref(true)
const filtro = ref('')
const cargando = ref(false)
const error = ref('')
const info = ref(null)       // datos del grafo (cortado, cantidad)
const hover = ref(null)
const elegido = ref(null)

let nodos = [], enlaces = [], vecinos = new Map(), sim = null
let t = {x: 0, y: 0, k: 1}   // transformación (pan/zoom)
let ancho = 0, alto = 0, dpr = 1, pedido = 0, rafPendiente = false
let arrastre = null          // {nodo} o {pan: true, x, y}
let movido = false
let obs = null

const COLOR_RAIZ = '#e8f1f3'
function radio(n) {
  if (n.nivel === 0) return 13
  if (!n.esCarpeta) return 3.6
  return Math.min(5 + Math.sqrt(n.hijos || 0) * 1.5, 18)
}
function color(n) {
  if (n.nivel === 0) return COLOR_RAIZ
  return n.esCarpeta ? COLOR_CARPETA : tipoArchivo(n.ext).color
}

async function cargar() {
  const mio = ++pedido
  cargando.value = true
  error.value = ''
  try {
    const g = await LeerGrafo(props.desde, profundidad.value, conArchivos.value)
    if (mio !== pedido) return
    armar(g)
  } catch (e) {
    if (mio === pedido) error.value = mensajeError(e)
  } finally {
    if (mio === pedido) cargando.value = false
  }
}

function armar(g) {
  sim?.stop()
  const viejos = new Map(nodos.map((n) => [n.id, n]))
  nodos = g.nodos.map((n) => {
    const v = viejos.get(n.id)
    return {...n, nombre: n.nivel === 0 && !n.id ? props.nombreRaiz : n.nombre, x: v?.x, y: v?.y}
  })
  const porId = new Map(nodos.map((n) => [n.id, n]))
  enlaces = g.enlaces.filter((l) => porId.has(l.origen) && porId.has(l.destino)).map((l) => ({source: l.origen, target: l.destino}))
  vecinos = new Map(nodos.map((n) => [n.id, new Set()]))
  for (const l of enlaces) { vecinos.get(l.source).add(l.target); vecinos.get(l.target).add(l.source) }
  info.value = {nodos: nodos.length, cortado: g.cortado, maximo: g.maximo, carpetas: nodos.filter((n) => n.esCarpeta).length}
  const raiz = nodos[0]
  if (raiz) { raiz.fx = 0; raiz.fy = 0 }

  sim = forceSimulation(nodos)
    .force('link', forceLink(enlaces).id((d) => d.id).distance((l) => (l.target.esCarpeta ? 46 : 22)).strength(0.9))
    .force('carga', forceManyBody().strength((d) => (d.esCarpeta ? -120 : -22)).distanceMax(420))
    .force('choque', forceCollide((d) => radio(d) + 2))
    .force('x', forceX(0).strength(0.02))
    .force('y', forceY(0).strength(0.02))
    .force('centro', forceCenter(0, 0).strength(0.05))
    .alphaDecay(0.035)
    .on('tick', pedirDibujo)
    .on('end', () => { if (raiz) { raiz.fx = null; raiz.fy = null } })
  if (!viejos.size) { t = {x: ancho / 2, y: alto / 2, k: nodos.length > 600 ? 0.55 : 0.9} }
  setTimeout(ajustar, 1300)
}

/* ---------- Dibujo ---------- */
function pedirDibujo() {
  if (rafPendiente) return
  rafPendiente = true
  requestAnimationFrame(() => { rafPendiente = false; dibujar() })
}
const coincide = (n) => !filtro.value || sinTildes(n.nombre).includes(sinTildes(filtro.value))

function dibujar() {
  const c = lienzo.value
  if (!c) return
  const ctx = c.getContext('2d')
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.fillStyle = '#0d2730'
  ctx.fillRect(0, 0, ancho, alto)
  ctx.translate(t.x, t.y)
  ctx.scale(t.k, t.k)

  const foco = hover.value || elegido.value
  const cerca = foco ? vecinos.get(foco.id) : null
  const hayFiltro = !!filtro.value
  const atenuado = (n) => (foco && n !== foco && !cerca?.has(n.id)) || (hayFiltro && !coincide(n))

  // Enlaces
  ctx.lineWidth = 1 / t.k
  for (const l of enlaces) {
    const s = l.source, d = l.target
    if (s.x == null || d.x == null) continue
    const activo = foco && (s === foco || d === foco)
    ctx.strokeStyle = activo ? 'rgba(242,179,61,.75)' : (foco || hayFiltro ? 'rgba(201,219,224,.06)' : 'rgba(201,219,224,.16)')
    ctx.beginPath(); ctx.moveTo(s.x, s.y); ctx.lineTo(d.x, d.y); ctx.stroke()
  }
  // Nodos
  for (const n of nodos) {
    if (n.x == null) continue
    const r = radio(n)
    ctx.globalAlpha = atenuado(n) ? 0.18 : 1
    ctx.beginPath(); ctx.arc(n.x, n.y, r, 0, Math.PI * 2)
    ctx.fillStyle = color(n); ctx.fill()
    if (n.ficha) { ctx.lineWidth = 2 / t.k + 1; ctx.strokeStyle = '#ffffff'; ctx.stroke() }
    if (n.id === props.seleccion || n === elegido.value) {
      ctx.lineWidth = 2.5 / t.k; ctx.strokeStyle = '#f2b33d'
      ctx.beginPath(); ctx.arc(n.x, n.y, r + 4 / t.k + 2, 0, Math.PI * 2); ctx.stroke()
    }
  }
  // Etiquetas: según el zoom, para no saturar
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  const tam = 11 / Math.max(t.k, 0.6)
  ctx.font = `500 ${tam}px "IBM Plex Sans", "Segoe UI", sans-serif`
  for (const n of nodos) {
    if (n.x == null) continue
    const importante = n.nivel === 0 || n === foco || cerca?.has(n.id) || (hayFiltro && coincide(n))
    const porZoom = n.esCarpeta ? (n.nivel <= 1 || t.k > 0.75) : t.k > 1.7
    if (!importante && !porZoom) continue
    if (atenuado(n) && !importante) continue
    ctx.globalAlpha = atenuado(n) ? 0.25 : 1
    const txt = n.nombre.length > 34 ? n.nombre.slice(0, 32) + '…' : n.nombre
    ctx.fillStyle = n.esCarpeta ? '#f3f7f8' : '#b9cdd3'
    ctx.fillText(txt, n.x, n.y + radio(n) + 3)
  }
  ctx.globalAlpha = 1
}

/* ---------- Interacción ---------- */
function aMundo(ev) {
  const b = lienzo.value.getBoundingClientRect()
  return {x: (ev.clientX - b.left - t.x) / t.k, y: (ev.clientY - b.top - t.y) / t.k}
}
function nodoEn(ev) {
  if (!sim) return null
  const p = aMundo(ev)
  const n = sim.find(p.x, p.y, 24 / t.k + 4)
  return n && Math.hypot(n.x - p.x, n.y - p.y) <= radio(n) + 6 / t.k ? n : null
}
function abajo(ev) {
  movido = false
  const n = nodoEn(ev)
  if (n) {
    arrastre = {nodo: n}
    sim.alphaTarget(0.25).restart()
  } else {
    arrastre = {pan: true, x: ev.clientX - t.x, y: ev.clientY - t.y}
  }
  lienzo.value.setPointerCapture(ev.pointerId)
}
function mover(ev) {
  if (arrastre?.nodo) {
    const p = aMundo(ev)
    arrastre.nodo.fx = p.x; arrastre.nodo.fy = p.y
    movido = true
    return
  }
  if (arrastre?.pan) {
    t.x = ev.clientX - arrastre.x; t.y = ev.clientY - arrastre.y
    movido = true
    pedirDibujo()
    return
  }
  const n = nodoEn(ev)
  if (n !== hover.value) { hover.value = n; lienzo.value.style.cursor = n ? 'pointer' : 'grab'; pedirDibujo() }
}
function arriba(ev) {
  if (arrastre?.nodo) {
    const n = arrastre.nodo
    if (n.nivel !== 0) { n.fx = null; n.fy = null }
    sim.alphaTarget(0)
    if (!movido) { elegido.value = n; emit('elegir', {ruta: n.id, esCarpeta: n.esCarpeta}) }
  } else if (!movido) {
    elegido.value = null
  }
  arrastre = null
  pedirDibujo()
}
function doble(ev) {
  const n = nodoEn(ev)
  if (!n) return
  if (n.esCarpeta) emit('desde', n.id)
  else emit('abrir', n.id)
}
function rueda(ev) {
  ev.preventDefault()
  const b = lienzo.value.getBoundingClientRect()
  const mx = ev.clientX - b.left, my = ev.clientY - b.top
  const k = Math.min(6, Math.max(0.08, t.k * Math.exp(-ev.deltaY * 0.0015)))
  t.x = mx - ((mx - t.x) / t.k) * k
  t.y = my - ((my - t.y) / t.k) * k
  t.k = k
  pedirDibujo()
}
function ajustar() {
  const ps = nodos.filter((n) => n.x != null && (!filtro.value || coincide(n)))
  if (!ps.length) return
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
  for (const n of ps) { x0 = Math.min(x0, n.x); y0 = Math.min(y0, n.y); x1 = Math.max(x1, n.x); y1 = Math.max(y1, n.y) }
  const m = 60
  const k = Math.min(2.2, Math.max(0.08, Math.min((ancho - m * 2) / (x1 - x0 || 1), (alto - m * 2) / (y1 - y0 || 1))))
  t = {k, x: ancho / 2 - ((x0 + x1) / 2) * k, y: alto / 2 - ((y0 + y1) / 2) * k}
  pedirDibujo()
}

function medir() {
  const c = lienzo.value, b = caja.value
  if (!c || !b) return
  dpr = window.devicePixelRatio || 1
  const r = b.getBoundingClientRect()
  const primero = !ancho
  ancho = r.width; alto = r.height
  c.width = Math.round(ancho * dpr); c.height = Math.round(alto * dpr)
  c.style.width = ancho + 'px'; c.style.height = alto + 'px'
  if (primero) t = {x: ancho / 2, y: alto / 2, k: 0.9}
  pedirDibujo()
}

const leyenda = computed(() => {
  if (!info.value) return []
  const m = new Map()
  for (const n of nodos) if (!n.esCarpeta) { const ti = tipoArchivo(n.ext); m.set(ti.tipo, (m.get(ti.tipo) || 0) + 1) }
  return [...m.entries()].sort((a, b) => b[1] - a[1]).slice(0, 8).map(([tipo, n]) => ({tipo, n, color: tipoArchivo(nodos.find((x) => !x.esCarpeta && tipoArchivo(x.ext).tipo === tipo)?.ext).color}))
})
const coincidencias = computed(() => (filtro.value && info.value ? nodos.filter(coincide).length : 0))

watch(() => [props.desde, profundidad.value, conArchivos.value], cargar)
watch(filtro, () => { pedirDibujo(); if (filtro.value) setTimeout(ajustar, 50) })
watch(() => props.seleccion, pedirDibujo)

onMounted(async () => {
  await nextTick()
  medir()
  obs = new ResizeObserver(medir)
  obs.observe(caja.value)
  cargar()
})
onBeforeUnmount(() => { sim?.stop(); obs?.disconnect() })
defineExpose({recargar: cargar})
</script>

<template>
  <section class="grafo">
    <header class="barra">
      <div class="titulo">
        <Icono nombre="grafo"/>
        <div>
          <strong>Vista gráfica</strong>
          <span class="desde" :title="rutaWin(desde) || nombreRaiz">desde {{ desde ? nombreDe(desde) : nombreRaiz }}</span>
        </div>
      </div>
      <div class="controles">
        <button v-if="desde" class="btn fantasma claro" title="Dibujar desde la raíz" @click="emit('desde', '')"><Icono nombre="casa"/>Raíz</button>
        <label class="ctl">Niveles
          <select v-model.number="profundidad"><option v-for="n in 6" :key="n" :value="n">{{ n }}</option></select>
        </label>
        <label class="ctl chk"><input type="checkbox" v-model="conArchivos"/> Archivos</label>
        <label class="buscar"><Icono nombre="buscar"/><input v-model="filtro" placeholder="Resaltar…"/></label>
        <button class="btn fantasma claro ic" title="Ajustar a la vista" @click="ajustar"><Icono nombre="centrar"/></button>
        <button class="btn fantasma claro ic" title="Volver a leer" @click="cargar"><Icono nombre="recargar"/></button>
      </div>
    </header>

    <div ref="caja" class="lienzo">
      <canvas ref="lienzo" @pointerdown="abajo" @pointermove="mover" @pointerup="arriba" @pointerleave="hover = null; pedirDibujo()"
              @dblclick="doble" @wheel="rueda"></canvas>

      <div v-if="cargando" class="estado"><span class="girando"><Icono nombre="recargar"/></span>Leyendo carpetas…</div>
      <div v-else-if="error" class="estado error">{{ error }}</div>

      <aside v-if="info" class="leyenda">
        <div><i :style="{background: '#e8f1f3'}"></i>Inicio del grafo</div>
        <div><i :style="{background: COLOR_CARPETA}"></i>Carpeta <small>(más grande = más contenido)</small></div>
        <div><i class="anillo" :style="{background: COLOR_CARPETA}"></i>Con ficha técnica</div>
        <div v-for="l in leyenda" :key="l.tipo"><i :style="{background: l.color}"></i>{{ l.tipo }} <small>{{ l.n }}</small></div>
        <p class="cifra">{{ info.carpetas }} carpetas · {{ info.nodos - info.carpetas }} archivos<template v-if="filtro"> · {{ coincidencias }} coinciden</template></p>
        <p v-if="info.cortado" class="cortado">Se muestran los primeros {{ info.maximo }} elementos. Baje los niveles o dibuje desde una subcarpeta.</p>
      </aside>

      <aside v-if="hover || elegido" class="ficha-nodo">
        <template v-for="n in [hover || elegido]" :key="n.id">
          <strong>{{ n.nombre }}</strong>
          <span class="ruta">{{ rutaWin(n.id) || nombreRaiz }}</span>
          <span class="tipo">{{ n.esCarpeta ? `Carpeta · ${n.hijos || 0} elementos${n.ficha ? ' · con ficha' : ''}` : `${tipoArchivo(n.ext).tipo} · ${tamano(n.tamano)}` }}</span>
          <span class="pista">{{ n.esCarpeta ? 'Doble clic: dibujar desde aquí' : 'Doble clic: abrir el archivo' }}</span>
        </template>
      </aside>
      <p class="ayuda-grafo">Rueda: zoom · Arrastrar: mover · Clic: seleccionar · Doble clic: entrar / abrir</p>
    </div>
  </section>
</template>

<style scoped>
.grafo { display: flex; flex-direction: column; height: 100%; background: #0d2730; color: #e8f1f3; }
.barra { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 16px; border-bottom: 1px solid #1d4250; flex-wrap: wrap; }
.titulo { display: flex; align-items: center; gap: 10px; }
.titulo > svg { width: 22px; height: 22px; color: var(--ambar); }
.titulo strong { display: block; font-size: var(--t-l); }
.desde { font-size: var(--t-xs); color: var(--riel-tenue); }
.controles { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.ctl { display: flex; align-items: center; gap: 6px; font-size: var(--t-s); color: var(--riel-tinta); }
.ctl select { background: #133640; color: #fff; border: 1px solid #22495a; border-radius: var(--r-s); padding: 3px 6px; }
.chk input { accent-color: var(--ambar); }
.buscar { display: flex; align-items: center; gap: 6px; height: 30px; padding: 0 8px; border-radius: var(--r-s); background: #133640; border: 1px solid #22495a; }
.buscar:focus-within { border-color: var(--ambar); }
.buscar svg { width: 14px; height: 14px; color: var(--riel-tenue); }
.buscar input { width: 150px; border: 0; background: none; color: #fff; outline: none; font-size: var(--t-s); }
.btn.claro { color: var(--riel-tinta); }
.btn.claro:hover { background: rgba(255, 255, 255, .08); }
.btn.ic { width: 30px; height: 30px; padding: 0; justify-content: center; }
.lienzo { position: relative; flex: 1; min-height: 0; overflow: hidden; }
canvas { display: block; cursor: grab; touch-action: none; }
.estado { position: absolute; top: 14px; left: 50%; transform: translateX(-50%); display: flex; align-items: center; gap: 8px; padding: 6px 14px; border-radius: 999px; background: rgba(13, 39, 48, .9); border: 1px solid #22495a; font-size: var(--t-s); }
.estado.error { color: #ff9b88; border-color: #6a2a20; }
.girando { display: inline-grid; animation: girar 1.2s linear infinite; }
.girando svg { width: 14px; height: 14px; }
@keyframes girar { to { transform: rotate(360deg); } }
.leyenda { position: absolute; left: 14px; bottom: 14px; display: grid; gap: 4px; max-width: 260px; padding: 10px 12px; border-radius: var(--r-m); background: rgba(13, 39, 48, .88); border: 1px solid #1d4250; font-size: var(--t-xs); color: var(--riel-tinta); }
.leyenda i { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 8px; vertical-align: -1px; }
.leyenda i.anillo { box-shadow: 0 0 0 2px #fff; }
.leyenda small { color: var(--riel-tenue); }
.cifra { margin: 4px 0 0; color: var(--riel-tenue); }
.cortado { margin: 2px 0 0; color: #f2c86b; }
.ficha-nodo { position: absolute; right: 14px; top: 14px; display: grid; gap: 2px; max-width: 340px; padding: 10px 14px; border-radius: var(--r-m); background: rgba(13, 39, 48, .92); border: 1px solid #22495a; font-size: var(--t-s); pointer-events: none; }
.ficha-nodo .ruta { color: var(--riel-tenue); font-size: var(--t-xs); overflow-wrap: anywhere; }
.ficha-nodo .tipo { color: var(--riel-tinta); }
.ficha-nodo .pista { color: var(--ambar); font-size: var(--t-xs); margin-top: 2px; }
.ayuda-grafo { position: absolute; right: 14px; bottom: 10px; margin: 0; font-size: 11px; color: #5f8590; pointer-events: none; }
</style>
