<script setup>
import {computed} from 'vue'
import Icono from './Icono.vue'
import {raizDeRuta, tipoArchivo} from '../util'
import {arrastrado, sePuedeSoltar, sobre, terminarArrastre} from '../arrastre'

// Árbol por demanda: cada carpeta se lee al abrirla (rápido aunque la raíz esté en red).
// Clic derecho: menú de acciones. Arrastrar y soltar sobre una carpeta: mover.
const props = defineProps({
  entradas: {type: Array, required: true},
  hijos: {type: Object, required: true},     // ruta -> entradas ya leídas
  abiertos: {type: Object, required: true},  // ruta -> true
  cargando: {type: Object, required: true},  // ruta -> true mientras se lee
  limites: {type: Object, required: true},   // ruta -> cuántas entradas mostrar
  colores: {type: Object, default: () => ({})}, // id de raíz -> color de sus carpetas
  seleccion: {type: String, default: null},
  conArchivos: Boolean,
  nivel: {type: Number, default: 0},
  ruta: {type: String, default: ''},          // carpeta a la que pertenecen estas entradas
})
const emit = defineEmits(['elegir', 'alternar', 'mas', 'menu', 'soltar'])
const PASO = 200

const filtradas = computed(() => (props.conArchivos ? props.entradas : props.entradas.filter((e) => e.esCarpeta)))
const limite = computed(() => props.limites[props.ruta] || PASO)
const visibles = computed(() => filtradas.value.slice(0, limite.value))
const sangria = (n) => ({paddingLeft: 8 + n * 16 + 'px'})
const colorDe = (r) => props.colores[raizDeRuta(r)] || 'var(--ambar)'
const esRaiz = (e) => !e.ruta.includes('/')

/* ---- Arrastrar ---- */
let tAbrir = null
function empezar(ev, e) {
  if (esRaiz(e)) { ev.preventDefault(); return }
  arrastrado.value = e
  ev.dataTransfer.effectAllowed = 'move'
  ev.dataTransfer.setData('text/plain', e.ruta)
}
function encima(ev, e) {
  if (!sePuedeSoltar(e)) return
  ev.preventDefault()
  ev.dataTransfer.dropEffect = 'move'
  if (sobre.value !== e.ruta) {
    sobre.value = e.ruta
    clearTimeout(tAbrir)
    // Si se queda encima de una carpeta cerrada, se abre para poder bajar a sus subcarpetas.
    if (!props.abiertos[e.ruta]) tAbrir = setTimeout(() => { if (sobre.value === e.ruta) emit('alternar', e.ruta) }, 900)
  }
}
function salir(e) {
  if (sobre.value === e.ruta) sobre.value = null
}
function soltar(ev, e) {
  ev.preventDefault()
  clearTimeout(tAbrir)
  const o = arrastrado.value
  const ok = sePuedeSoltar(e)
  terminarArrastre()
  if (o && ok) emit('soltar', o, e)
}
</script>

<template>
  <ul class="arbol" role="group">
    <li v-for="e in visibles" :key="e.ruta">
      <div class="fila" :class="{activo: seleccion === e.ruta, carpeta: e.esCarpeta, raiz: esRaiz(e), soltar: sobre === e.ruta, origen: arrastrado?.ruta === e.ruta}"
           :style="[sangria(nivel), {'--cf': colorDe(e.ruta)}]"
           role="treeitem" :aria-expanded="e.esCarpeta ? !!abiertos[e.ruta] : undefined" :aria-selected="seleccion === e.ruta"
           :title="e.nombre" :draggable="!esRaiz(e)"
           @click="emit('elegir', e)" @contextmenu.prevent="emit('menu', $event, e)"
           @dragstart="empezar($event, e)" @dragend="terminarArrastre" @dragover="encima($event, e)" @dragleave="salir(e)" @drop="soltar($event, e)">
        <button v-if="e.esCarpeta" class="chev" :aria-label="abiertos[e.ruta] ? 'Cerrar' : 'Abrir'" @click.stop="emit('alternar', e.ruta)">
          <Icono nombre="chevron" :class="{girado: abiertos[e.ruta]}"/>
        </button>
        <span v-else class="chev"></span>
        <Icono v-if="e.esCarpeta" :nombre="abiertos[e.ruta] ? 'carpeta-abierta' : 'carpeta'" class="ic carp"/>
        <span v-else class="ic arch" :style="{'--c': tipoArchivo(e.ext).color}"><Icono nombre="archivo"/></span>
        <span class="nombre">{{ e.nombre }}</span>
        <span v-if="e.esCarpeta && e.ficha" class="punto" title="Tiene ficha técnica"></span>
      </div>
      <template v-if="e.esCarpeta && abiertos[e.ruta]">
        <div v-if="cargando[e.ruta] && !hijos[e.ruta]" class="info" :style="sangria(nivel + 1)">Leyendo…</div>
        <div v-else-if="hijos[e.ruta] && !(conArchivos ? hijos[e.ruta].length : hijos[e.ruta].some((x) => x.esCarpeta))"
             class="info" :style="sangria(nivel + 1)">{{ hijos[e.ruta].length ? 'Solo archivos (ocultos)' : 'Vacía' }}</div>
        <Arbol v-else-if="hijos[e.ruta]" :entradas="hijos[e.ruta]" :hijos="hijos" :abiertos="abiertos" :cargando="cargando"
               :limites="limites" :colores="colores" :seleccion="seleccion" :con-archivos="conArchivos" :nivel="nivel + 1" :ruta="e.ruta"
               @elegir="emit('elegir', $event)" @alternar="emit('alternar', $event)" @mas="emit('mas', $event)"
               @menu="(ev, x) => emit('menu', ev, x)" @soltar="(o, d) => emit('soltar', o, d)"/>
      </template>
    </li>
    <li v-if="filtradas.length > limite">
      <button class="mas" :style="sangria(nivel)" @click="emit('mas', ruta)">
        Ver {{ Math.min(PASO, filtradas.length - limite) }} más ({{ filtradas.length - limite }} restantes)
      </button>
    </li>
  </ul>
</template>

<style scoped>
.arbol { list-style: none; margin: 0; padding: 0; }
.fila {
  display: flex; align-items: center; gap: 6px; min-height: 28px; padding-right: 10px;
  color: var(--riel-tinta); font-size: var(--t-s); cursor: pointer; border-left: 3px solid transparent; user-select: none;
}
.fila:hover { background: rgba(255, 255, 255, .05); }
.fila.carpeta { color: #e8f1f3; }
.fila.raiz { min-height: 32px; font-weight: 600; color: #fff; }
.fila.raiz .ic.carp { width: 18px; height: 18px; }
.fila.activo { background: color-mix(in srgb, var(--cf) 18%, transparent); border-left-color: var(--cf); color: #fff; }
.fila.soltar { background: color-mix(in srgb, var(--cf) 30%, transparent); box-shadow: inset 0 0 0 1px var(--cf); }
.fila.origen { opacity: .45; }
.chev { display: grid; place-items: center; width: 18px; height: 18px; flex: none; padding: 0; border: 0; background: none; color: var(--riel-tenue); border-radius: 3px; }
.chev:hover { background: rgba(255, 255, 255, .1); color: #fff; }
.chev svg { width: 13px; height: 13px; transition: transform .12s; }
.chev svg.girado { transform: rotate(90deg); }
.ic { width: 16px; height: 16px; flex: none; }
.ic.carp { color: var(--cf); }
.ic.arch { display: grid; place-items: center; color: var(--c); }
.ic.arch svg { width: 15px; height: 15px; }
.nombre { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.punto { width: 7px; height: 7px; border-radius: 50%; background: var(--cf); flex: none; box-shadow: 0 0 0 1.5px rgba(255,255,255,.5); }
.info { padding: 3px 10px 3px 0; font-size: var(--t-xs); color: var(--riel-tenue); font-style: italic; margin-left: 30px; }
.mas { display: block; width: 100%; padding: 4px 10px; border: 0; background: none; color: var(--ambar); font-size: var(--t-xs); text-align: left; margin-left: 24px; }
.mas:hover { text-decoration: underline; }
</style>
