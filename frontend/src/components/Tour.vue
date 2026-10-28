<script setup>
import {computed, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import Icono from './Icono.vue'

// Guía paso a paso: oscurece la pantalla, ilumina un elemento y explica qué hace.
// pasos: [{sel: '.clase' | ['.a', '.b'] (el primero que exista) | null (centrado),
//          titulo, texto (puede tener <b>, <code>…), espera: 'texto' (el usuario debe hacer algo; no hay "Siguiente")}]
const props = defineProps({
  pasos: {type: Array, required: true},
  inicio: {type: Number, default: 0},
})
const emit = defineEmits(['cerrar', 'terminar'])

const i = ref(props.inicio)
const caja = ref(null)        // rectángulo iluminado (null = centrado)
const pop = ref(null)
const estiloPop = ref({})
const paso = computed(() => props.pasos[i.value] || {})
const ultimo = computed(() => i.value === props.pasos.length - 1)
const MARGEN = 8

function objetivo() {
  const sels = Array.isArray(paso.value.sel) ? paso.value.sel : paso.value.sel ? [paso.value.sel] : []
  for (const s of sels) {
    const el = document.querySelector(s)
    if (el && el.getClientRects().length) return el
  }
  return null
}
async function ubicar(desplazar = false) {
  await nextTick()
  const el = objetivo()
  if (el && desplazar) {
    el.scrollIntoView({block: 'nearest', inline: 'nearest'})
    await new Promise((r) => setTimeout(r, 60))
  }
  if (!el) {
    caja.value = null
  } else {
    const r = el.getBoundingClientRect()
    caja.value = {x: r.left - MARGEN, y: r.top - MARGEN, w: r.width + MARGEN * 2, h: r.height + MARGEN * 2}
  }
  await nextTick()
  colocarPop()
}
// Pone la explicación al lado del elemento, donde haya espacio.
function colocarPop() {
  const p = pop.value
  if (!p) return
  const W = window.innerWidth, H = window.innerHeight
  const pw = p.offsetWidth, ph = p.offsetHeight, sep = 14
  const c = caja.value
  if (!c) { estiloPop.value = {left: (W - pw) / 2 + 'px', top: (H - ph) / 2 + 'px'}; return }
  const lugares = [
    {ok: W - (c.x + c.w) > pw + sep, left: c.x + c.w + sep, top: c.y + c.h / 2 - ph / 2},
    {ok: H - (c.y + c.h) > ph + sep, left: c.x + c.w / 2 - pw / 2, top: c.y + c.h + sep},
    {ok: c.x > pw + sep, left: c.x - pw - sep, top: c.y + c.h / 2 - ph / 2},
    {ok: c.y > ph + sep, left: c.x + c.w / 2 - pw / 2, top: c.y - ph - sep},
  ]
  const l = lugares.find((x) => x.ok) || {left: (W - pw) / 2, top: H - ph - 20}
  estiloPop.value = {
    left: Math.max(12, Math.min(l.left, W - pw - 12)) + 'px',
    top: Math.max(12, Math.min(l.top, H - ph - 12)) + 'px',
  }
}
function ir(n) {
  if (n < 0 || n >= props.pasos.length) return
  i.value = n
}
function siguiente() { ultimo.value ? emit('terminar') : ir(i.value + 1) }
function tecla(e) {
  if (e.key === 'Escape') emit('cerrar')
  else if (e.key === 'ArrowRight' && !paso.value.espera) { e.preventDefault(); siguiente() }
  else if (e.key === 'ArrowLeft') { e.preventDefault(); ir(i.value - 1) }
}
const reubicar = () => ubicar()
let obs = null
watch(i, () => ubicar(true))
watch(() => props.pasos, () => ubicar(true))
onMounted(() => {
  ubicar(true)
  window.addEventListener('resize', reubicar)
  window.addEventListener('keydown', tecla, true)
  // Si la pantalla cambia (se abre una carpeta, aparece un panel…), el foco se reacomoda.
  obs = new MutationObserver(() => { clearTimeout(obs.t); obs.t = setTimeout(reubicar, 120) })
  obs.observe(document.body, {childList: true, subtree: true})
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', reubicar)
  window.removeEventListener('keydown', tecla, true)
  obs?.disconnect()
})
defineExpose({ir, actual: i})
</script>

<template>
  <div class="tour" role="dialog" aria-modal="false" :aria-label="paso.titulo">
    <!-- Oscurece todo menos el elemento explicado (que sigue siendo clicable) -->
    <div v-if="caja" class="foco" :class="{espera: paso.espera}"
         :style="{left: caja.x + 'px', top: caja.y + 'px', width: caja.w + 'px', height: caja.h + 'px'}"></div>
    <div v-else class="velo-tour"></div>

    <div ref="pop" class="pop" :style="estiloPop">
      <div class="cab">
        <span class="num">{{ i + 1 }} / {{ pasos.length }}</span>
        <button class="saltar" @click="emit('cerrar')">Saltar guía</button>
      </div>
      <h3>{{ paso.titulo }}</h3>
      <div class="texto" v-html="paso.texto"></div>
      <p v-if="paso.espera" class="espera-txt"><Icono nombre="chevron"/>{{ paso.espera }}</p>
      <div class="puntos" aria-hidden="true">
        <i v-for="(p, k) in pasos" :key="k" :class="{on: k === i, hecho: k < i}" @click="ir(k)"></i>
      </div>
      <div class="pie">
        <button class="btn fantasma" :disabled="i === 0" @click="ir(i - 1)">Atrás</button>
        <button v-if="!paso.espera" class="btn primario" @click="siguiente">{{ ultimo ? 'Terminar' : 'Siguiente' }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tour { position: fixed; inset: 0; z-index: 90; pointer-events: none; }
.velo-tour { position: fixed; inset: 0; background: rgba(8, 24, 30, .62); }
.foco {
  position: fixed; border-radius: 10px; box-shadow: 0 0 0 9999px rgba(8, 24, 30, .62), 0 0 0 3px var(--ambar);
  transition: left .3s ease, top .3s ease, width .3s ease, height .3s ease;
}
.foco.espera { animation: latir 1.4s ease-in-out infinite; }
@keyframes latir {
  0%, 100% { box-shadow: 0 0 0 9999px rgba(8, 24, 30, .62), 0 0 0 3px var(--ambar); }
  50% { box-shadow: 0 0 0 9999px rgba(8, 24, 30, .62), 0 0 0 8px rgba(242, 179, 61, .55); }
}
.pop {
  position: fixed; width: min(380px, calc(100vw - 24px)); padding: 16px 18px 14px; pointer-events: auto;
  background: #fff; border-radius: 12px; box-shadow: 0 18px 50px rgba(0, 0, 0, .35); border-top: 4px solid var(--ambar);
  transition: left .3s ease, top .3s ease; animation: entrar .25s ease;
}
@keyframes entrar { from { opacity: 0; transform: translateY(6px); } }
.cab { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.num { font-size: var(--t-xs); font-weight: 600; color: var(--ambar-oscuro); letter-spacing: .04em; }
.saltar { border: 0; background: none; padding: 0; font-size: var(--t-xs); color: var(--tinta-3); }
.saltar:hover { color: var(--tinta); text-decoration: underline; }
h3 { margin: 0 0 6px; font-size: var(--t-l); }
.texto { font-size: var(--t-s); color: var(--tinta-2); line-height: 1.5; }
.texto :deep(p) { margin: 0 0 6px; }
.texto :deep(ul) { margin: 4px 0 6px; padding-left: 18px; }
.texto :deep(code) { font-size: .92em; padding: 0 4px; border-radius: 3px; background: var(--linea-2); }
.espera-txt { display: flex; align-items: center; gap: 4px; margin: 8px 0 0; padding: 6px 10px; border-radius: var(--r-s); background: var(--ambar-claro); font-size: var(--t-s); font-weight: 600; color: var(--ambar-oscuro); }
.espera-txt svg { width: 14px; height: 14px; }
.puntos { display: flex; flex-wrap: wrap; gap: 5px; margin: 12px 0 10px; }
.puntos i { width: 7px; height: 7px; border-radius: 50%; background: var(--linea); cursor: pointer; }
.puntos i.hecho { background: #f6d58e; }
.puntos i.on { background: var(--ambar); width: 18px; border-radius: 99px; }
.pie { display: flex; justify-content: space-between; }
</style>
