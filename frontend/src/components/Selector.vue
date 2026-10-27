<script setup>
import {onMounted, reactive} from 'vue'
import Icono from './Icono.vue'
import {Listar} from '../../wailsjs/go/main/App'
import {mensajeError} from '../util'

// Elige una carpeta destino navegando el árbol (solo carpetas, carga por demanda).
const props = defineProps({
  modelValue: {type: String, default: null}, // ruta elegida ('' = raíz, null = ninguna)
  nombreRaiz: {type: String, default: 'Raíz'},
  bloqueada: {type: String, default: ''},    // carpeta que se está moviendo: ni ella ni sus hijas son destino válido
})
const emit = defineEmits(['update:modelValue'])
const hijos = reactive({})
const abiertos = reactive({'': true})
const errores = reactive({})

async function cargar(r) {
  if (hijos[r]) return
  try {
    hijos[r] = (await Listar(r)).filter((e) => e.esCarpeta)
  } catch (e) {
    errores[r] = mensajeError(e)
    hijos[r] = []
  }
}
async function alternar(r) {
  abiertos[r] = !abiertos[r]
  if (abiertos[r]) await cargar(r)
}
const bloqueado = (r) => props.bloqueada && (r === props.bloqueada || r.startsWith(props.bloqueada + '/'))
function elegir(r) {
  if (!bloqueado(r)) emit('update:modelValue', r)
}
// Lista plana de lo visible (más simple que un componente recursivo).
function filas() {
  const out = []
  const recorrer = (r, nivel) => {
    for (const e of hijos[r] || []) {
      out.push({...e, nivel})
      if (abiertos[e.ruta]) {
        if (!hijos[e.ruta]) out.push({info: 'Leyendo…', nivel: nivel + 1, ruta: e.ruta + '/…'})
        else if (!hijos[e.ruta].length) out.push({info: 'Sin subcarpetas', nivel: nivel + 1, ruta: e.ruta + '/∅'})
        else recorrer(e.ruta, nivel + 1)
      }
    }
  }
  recorrer('', 1)
  return out
}
onMounted(() => cargar(''))
</script>

<template>
  <div class="selector" role="tree">
    <div class="fila" :class="{activo: modelValue === ''}" @click="elegir('')">
      <span class="chev"></span><Icono nombre="casa" class="ic"/><span class="nombre">{{ nombreRaiz }}</span>
    </div>
    <template v-for="f in filas()" :key="f.ruta">
      <div v-if="f.info" class="info" :style="{paddingLeft: 10 + f.nivel * 16 + 'px'}">{{ f.info }}</div>
      <div v-else class="fila" :class="{activo: modelValue === f.ruta, bloq: bloqueado(f.ruta)}"
           :style="{paddingLeft: 4 + f.nivel * 16 + 'px'}" :title="bloqueado(f.ruta) ? 'No se puede mover dentro de sí misma' : f.ruta"
           @click="elegir(f.ruta)">
        <button class="chev" @click.stop="alternar(f.ruta)"><Icono nombre="chevron" :class="{girado: abiertos[f.ruta]}"/></button>
        <Icono :nombre="abiertos[f.ruta] ? 'carpeta-abierta' : 'carpeta'" class="ic"/>
        <span class="nombre">{{ f.nombre }}</span>
      </div>
    </template>
    <p v-if="errores['']" class="err">{{ errores[''] }}</p>
  </div>
</template>

<style scoped>
.selector { height: 280px; overflow-y: auto; border: 1px solid var(--linea); border-radius: var(--r-s); background: #fff; padding: 4px 0; }
.fila { display: flex; align-items: center; gap: 6px; min-height: 28px; padding-right: 10px; font-size: var(--t-s); cursor: pointer; }
.fila:hover { background: var(--linea-2); }
.fila.activo { background: var(--ambar-claro); box-shadow: inset 3px 0 0 var(--ambar); font-weight: 600; }
.fila.bloq { opacity: .45; cursor: not-allowed; }
.chev { display: grid; place-items: center; width: 18px; height: 18px; flex: none; padding: 0; border: 0; background: none; color: var(--tinta-3); }
.chev svg { width: 13px; height: 13px; transition: transform .12s; }
.chev svg.girado { transform: rotate(90deg); }
.ic { width: 16px; height: 16px; flex: none; color: var(--ambar-oscuro); }
.nombre { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.info { font-size: var(--t-xs); color: var(--tinta-3); font-style: italic; padding: 2px 0; }
.err { margin: 6px 10px; color: var(--rojo); font-size: var(--t-xs); }
</style>
