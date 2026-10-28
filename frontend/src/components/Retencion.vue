<script setup>
import {computed} from 'vue'
import {leerRetencion, SIN_DEPURAR, textoRetencion, UNIDADES} from '../util'

// Editor del tiempo de retención: cuánto deben quedarse los archivos en una carpeta.
// Valor guardado: "6 meses", "2 años", "No depurar" o "" (sin regla / hereda).
const props = defineProps({
  modelValue: {type: String, default: ''},
  heredada: {type: String, default: ''},  // subcarpeta: la que aplica si no tiene propia
  subcarpeta: Boolean,
  compacto: Boolean,
})
const emit = defineEmits(['update:modelValue'])

const t = computed(() => leerRetencion(props.modelValue))
const modo = computed(() => {
  const v = (props.modelValue || '').trim()
  if (!v) return ''
  if (v.toLowerCase() === SIN_DEPURAR.toLowerCase()) return 'no'
  return t.value ? 'tiempo' : 'otro'
})
function cambiarModo(m) {
  if (m === '') emit('update:modelValue', '')
  else if (m === 'no') emit('update:modelValue', SIN_DEPURAR)
  else if (m === 'tiempo') emit('update:modelValue', textoRetencion(t.value?.n || 12, t.value?.u || 'meses'))
}
function cambiarN(n) {
  n = Math.max(1, Math.min(1200, parseInt(n) || 1))
  emit('update:modelValue', textoRetencion(n, t.value?.u || 'meses'))
}
function cambiarU(u) {
  emit('update:modelValue', textoRetencion(t.value?.n || 1, u))
}
</script>

<template>
  <div class="retencion" :class="{compacto}">
    <select class="campo-input modo" :value="modo" @change="cambiarModo($event.target.value)" aria-label="Tipo de retención">
      <option value="">{{ subcarpeta ? (heredada ? `Igual que la carpeta superior (${heredada})` : 'Igual que la carpeta superior (sin regla)') : 'Sin definir' }}</option>
      <option value="tiempo">Conservar por un tiempo…</option>
      <option value="no">No depurar nunca</option>
      <option v-if="modo === 'otro'" value="otro">{{ modelValue }} (no es un tiempo válido)</option>
    </select>
    <template v-if="modo === 'tiempo'">
      <input class="campo-input num" type="number" min="1" max="1200" :value="t.n" aria-label="Cantidad"
             @input="cambiarN($event.target.value)"/>
      <select class="campo-input unidad" :value="t.u" aria-label="Unidad" @change="cambiarU($event.target.value)">
        <option v-for="x in UNIDADES" :key="x.u" :value="x.u">{{ x.varios }}</option>
      </select>
    </template>
  </div>
</template>

<style scoped>
.retencion { display: flex; gap: 6px; align-items: center; flex-wrap: nowrap; max-width: 620px; width: 100%; }
.retencion .campo-input { width: auto; }
.modo { min-width: 0; flex: 1 1 260px; }
.num { flex: 0 0 76px; width: 76px; }
.unidad { flex: 0 0 120px; }
.compacto .modo { flex-basis: 200px; }
.compacto .campo-input { font-size: var(--t-xs); padding-top: 3px; padding-bottom: 3px; }
</style>
