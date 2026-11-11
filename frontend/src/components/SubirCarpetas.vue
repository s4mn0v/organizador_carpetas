<script setup>
// Subir una o varias carpetas completas a la carpeta elegida.
// Se agregan con "Agregar carpeta…" (selector de Windows) o arrastrándolas desde el Explorador.
import {onBeforeUnmount, onMounted, ref} from 'vue'
import Icono from './Icono.vue'
import {CancelarSubida, ElegirCarpetaOrigen, SubirCarpetas} from '../../wailsjs/go/main/App'
import {EventsOn, OnFileDrop, OnFileDropOff} from '../../wailsjs/runtime/runtime'
import {mensajeError, tamano} from '../util'

const props = defineProps({
  destino: {type: String, required: true},        // ruta virtual
  destinoVisible: {type: String, required: true},
})
const emit = defineEmits(['cerrar', 'listo'])

const origenes = ref([])          // rutas de Windows
const subiendo = ref(false)
const prog = ref(null)
const error = ref('')
let quitarProg = null

const nombre = (p) => p.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || p
function agregar(rutas) {
  for (const r of rutas) if (!origenes.value.some((o) => o.toLowerCase() === r.toLowerCase())) origenes.value.push(r)
}
async function elegir() {
  error.value = ''
  try {
    const r = await ElegirCarpetaOrigen()
    if (r) agregar([r])
  } catch (e) { error.value = mensajeError(e) }
}
async function subir() {
  if (!origenes.value.length || subiendo.value) return
  subiendo.value = true
  error.value = ''
  try {
    const r = await SubirCarpetas(props.destino, origenes.value)
    emit('listo', r)
  } catch (e) {
    error.value = mensajeError(e)
  } finally {
    subiendo.value = false
  }
}
function cerrar() { if (!subiendo.value) emit('cerrar') }

onMounted(() => {
  quitarProg = EventsOn('subir:progreso', (p) => { prog.value = p })
  // Carpetas arrastradas desde el Explorador (solo cuenta si se sueltan en la zona marcada).
  OnFileDrop((x, y, rutas) => { if (!subiendo.value) agregar(rutas || []) }, true)
})
onBeforeUnmount(() => { quitarProg && quitarProg(); OnFileDropOff() })
</script>

<template>
  <div class="sc-fondo" @click.self="cerrar">
    <div class="sc-caja" role="dialog" aria-label="Subir carpetas">
      <header class="sc-cab">
        <div>
          <strong>Subir carpetas</strong>
          <small>a <code>{{ destinoVisible }}</code></small>
        </div>
        <button class="btn fantasma ic" title="Cerrar" :disabled="subiendo" @click="cerrar"><Icono nombre="cerrar"/></button>
      </header>

      <div class="sc-zona" :class="{vacia: !origenes.length}" style="--wails-drop-target: drop">
        <template v-if="!origenes.length">
          <Icono nombre="carpeta-mas"/>
          <p>Arrastre aquí una o varias carpetas desde el Explorador<br><small>o use “Agregar carpeta…”</small></p>
        </template>
        <ul v-else class="sc-lista">
          <li v-for="(o, i) in origenes" :key="o">
            <Icono nombre="carpeta"/>
            <span class="sc-nom"><b>{{ nombre(o) }}</b><small>{{ o }}</small></span>
            <button class="btn fantasma ic" title="Quitar de la lista" :disabled="subiendo" @click="origenes.splice(i, 1)"><Icono nombre="cerrar"/></button>
          </li>
        </ul>
      </div>

      <p v-if="subiendo && prog" class="sc-prog">
        <span class="girador"></span>
        Copiando <b>{{ prog.carpeta }}</b>: {{ prog.archivos.toLocaleString('es-CO') }} archivo(s), {{ tamano(prog.tamano) }}
        <small>{{ prog.actual }}</small>
      </p>
      <p v-if="error" class="sc-error">{{ error }}</p>
      <p class="sc-nota">Se copia todo su contenido; los originales no se tocan. Si ya existe una carpeta con el mismo nombre, la nueva queda como “nombre (2)”.</p>

      <footer class="sc-pie">
        <button class="btn" :disabled="subiendo" @click="elegir"><Icono nombre="carpeta-mas"/>Agregar carpeta…</button>
        <span class="sc-esp"></span>
        <button v-if="subiendo" class="btn peligro" @click="CancelarSubida()">Detener</button>
        <button v-else class="btn" @click="cerrar">Cancelar</button>
        <button class="btn nueva" :disabled="!origenes.length || subiendo" @click="subir">
          <Icono nombre="subir"/>Subir {{ origenes.length > 1 ? origenes.length + ' carpetas' : 'carpeta' }}
        </button>
      </footer>
    </div>
  </div>
</template>

<style>
.sc-fondo { position: fixed; inset: 0; z-index: 90; display: grid; place-items: center; background: rgba(16, 49, 59, .35); }
.sc-caja { width: min(620px, calc(100vw - 40px)); max-height: calc(100vh - 60px); display: flex; flex-direction: column; gap: 10px; padding: 16px 18px;
  border-radius: var(--r-m); background: var(--superficie); box-shadow: 0 20px 60px rgba(0, 0, 0, .3); }
.sc-cab { display: flex; justify-content: space-between; align-items: flex-start; }
.sc-cab div { display: grid; }
.sc-cab small { color: var(--tinta-3); font-size: var(--t-xs); }
.sc-zona { min-height: 150px; max-height: 45vh; overflow-y: auto; border: 2px dashed var(--linea); border-radius: var(--r-m); padding: 8px; }
.sc-zona.vacia { display: grid; place-items: center; text-align: center; color: var(--tinta-3); }
.sc-zona.vacia svg { width: 34px; height: 34px; color: var(--ambar); }
.sc-zona.vacia p { margin: 6px 0 0; font-size: var(--t-s); }
.sc-zona.wails-drop-target-active { border-color: var(--ambar); background: var(--ambar-claro); }
.sc-lista { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
.sc-lista li { display: flex; align-items: center; gap: 8px; padding: 4px 6px; border-radius: var(--r-s); background: var(--papel); }
.sc-lista li > svg { width: 18px; height: 18px; color: var(--ambar); flex: none; }
.sc-nom { flex: 1; min-width: 0; display: grid; }
.sc-nom b { font-size: var(--t-s); font-weight: 600; }
.sc-nom small { font-size: 11px; color: var(--tinta-3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-prog { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin: 0; font-size: var(--t-s); color: var(--tinta-2); }
.sc-prog small { flex-basis: 100%; color: var(--tinta-3); font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-error { margin: 0; padding: 6px 8px; border-radius: var(--r-s); background: var(--rojo-claro); color: var(--rojo); font-size: var(--t-s); }
.sc-nota { margin: 0; color: var(--tinta-3); font-size: 11px; }
.sc-pie { display: flex; align-items: center; gap: 6px; }
.sc-esp { flex: 1; }
</style>
