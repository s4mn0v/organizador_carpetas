<script setup>
import {computed} from 'vue'
import Icono from './Icono.vue'
import {tipoArchivo} from '../util'

// Árbol por demanda: cada carpeta se lee al abrirla (rápido aunque la raíz esté en red).
const props = defineProps({
  entradas: {type: Array, required: true},
  hijos: {type: Object, required: true},     // ruta -> entradas ya leídas
  abiertos: {type: Object, required: true},  // ruta -> true
  cargando: {type: Object, required: true},  // ruta -> true mientras se lee
  limites: {type: Object, required: true},   // ruta -> cuántas entradas mostrar
  seleccion: {type: String, default: null},
  conArchivos: Boolean,
  nivel: {type: Number, default: 0},
  ruta: {type: String, default: ''},          // carpeta a la que pertenecen estas entradas
})
const emit = defineEmits(['elegir', 'alternar', 'mas'])
const PASO = 200

const filtradas = computed(() => (props.conArchivos ? props.entradas : props.entradas.filter((e) => e.esCarpeta)))
const limite = computed(() => props.limites[props.ruta] || PASO)
const visibles = computed(() => filtradas.value.slice(0, limite.value))
const sangria = (n) => ({paddingLeft: 8 + n * 16 + 'px'})
</script>

<template>
  <ul class="arbol" role="group">
    <li v-for="e in visibles" :key="e.ruta">
      <div class="fila" :class="{activo: seleccion === e.ruta, carpeta: e.esCarpeta}" :style="sangria(nivel)"
           role="treeitem" :aria-expanded="e.esCarpeta ? !!abiertos[e.ruta] : undefined" :aria-selected="seleccion === e.ruta"
           :title="e.nombre" @click="emit('elegir', e)">
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
               :limites="limites" :seleccion="seleccion" :con-archivos="conArchivos" :nivel="nivel + 1" :ruta="e.ruta"
               @elegir="emit('elegir', $event)" @alternar="emit('alternar', $event)" @mas="emit('mas', $event)"/>
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
.fila.activo { background: rgba(242, 179, 61, .16); border-left-color: var(--ambar); color: #fff; }
.chev { display: grid; place-items: center; width: 18px; height: 18px; flex: none; padding: 0; border: 0; background: none; color: var(--riel-tenue); border-radius: 3px; }
.chev:hover { background: rgba(255, 255, 255, .1); color: #fff; }
.chev svg { width: 13px; height: 13px; transition: transform .12s; }
.chev svg.girado { transform: rotate(90deg); }
.ic { width: 16px; height: 16px; flex: none; }
.ic.carp { color: var(--ambar); }
.ic.arch { display: grid; place-items: center; color: var(--c); }
.ic.arch svg { width: 15px; height: 15px; }
.nombre { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.punto { width: 7px; height: 7px; border-radius: 50%; background: var(--ambar); flex: none; }
.info { padding: 3px 10px 3px 0; font-size: var(--t-xs); color: var(--riel-tenue); font-style: italic; margin-left: 30px; }
.mas { display: block; width: 100%; padding: 4px 10px; border: 0; background: none; color: var(--ambar); font-size: var(--t-xs); text-align: left; margin-left: 24px; }
.mas:hover { text-decoration: underline; }
</style>
