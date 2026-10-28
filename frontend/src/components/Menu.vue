<script setup>
import {nextTick, onBeforeUnmount, onMounted, ref} from 'vue'
import Icono from './Icono.vue'

// Menú del clic derecho. items: [{texto, icono, accion, peligro, separador, titulo}]
const props = defineProps({
  x: {type: Number, required: true},
  y: {type: Number, required: true},
  titulo: {type: String, default: ''},
  items: {type: Array, required: true},
})
const emit = defineEmits(['cerrar'])
const caja = ref(null)
const pos = ref({left: props.x + 'px', top: props.y + 'px'})

function fuera(ev) { if (!caja.value?.contains(ev.target)) emit('cerrar') }
function tecla(ev) { if (ev.key === 'Escape') emit('cerrar') }
function elegir(it) {
  emit('cerrar')
  it.accion?.()
}
onMounted(async () => {
  await nextTick()
  // Que no se salga de la ventana.
  const r = caja.value.getBoundingClientRect()
  pos.value = {
    left: Math.max(4, Math.min(props.x, window.innerWidth - r.width - 6)) + 'px',
    top: Math.max(4, Math.min(props.y, window.innerHeight - r.height - 6)) + 'px',
  }
  caja.value.querySelector('button')?.focus()
  setTimeout(() => {
    window.addEventListener('mousedown', fuera, true)
    window.addEventListener('contextmenu', fuera, true)
  })
  window.addEventListener('keydown', tecla)
  window.addEventListener('blur', () => emit('cerrar'), {once: true})
})
onBeforeUnmount(() => {
  window.removeEventListener('mousedown', fuera, true)
  window.removeEventListener('contextmenu', fuera, true)
  window.removeEventListener('keydown', tecla)
})
</script>

<template>
  <div ref="caja" class="menu" role="menu" :style="pos" @contextmenu.prevent>
    <div v-if="titulo" class="titulo">{{ titulo }}</div>
    <template v-for="(it, i) in items" :key="i">
      <hr v-if="it.separador"/>
      <button v-else role="menuitem" :class="{peligro: it.peligro}" :title="it.titulo || ''" @click="elegir(it)">
        <Icono v-if="it.icono" :nombre="it.icono"/><span>{{ it.texto }}</span>
        <kbd v-if="it.tecla">{{ it.tecla }}</kbd>
      </button>
    </template>
  </div>
</template>

<style scoped>
.menu {
  position: fixed; z-index: 80; min-width: 210px; max-width: 320px; padding: 4px;
  background: #fff; border: 1px solid var(--linea); border-radius: var(--r-m);
  box-shadow: 0 10px 30px rgba(16, 49, 59, .22); font-size: var(--t-s);
}
.titulo { padding: 6px 10px 4px; font-size: var(--t-xs); color: var(--tinta-3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
button { display: flex; align-items: center; gap: 10px; width: 100%; padding: 6px 10px; border: 0; border-radius: var(--r-s); background: none; text-align: left; color: var(--tinta); }
button:hover, button:focus-visible { background: var(--linea-2); outline: none; }
button svg { width: 15px; height: 15px; color: var(--tinta-2); flex: none; }
button span { flex: 1; }
button.peligro { color: var(--rojo); }
button.peligro svg { color: var(--rojo); }
button.peligro:hover { background: var(--rojo-claro); }
kbd { font-family: inherit; font-size: 11px; color: var(--tinta-3); }
hr { border: 0; border-top: 1px solid var(--linea-2); margin: 4px 2px; }
</style>
