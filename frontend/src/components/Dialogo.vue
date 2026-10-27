<script setup>
import {ref, computed, onMounted, nextTick} from 'vue'
import Icono from './Icono.vue'

defineOptions({inheritAttrs: false})
const props = defineProps({
  titulo: String,
  mensaje: String,
  detalles: {type: Array, default: () => []},
  textoConfirmar: {type: String, default: 'Aceptar'},
  peligro: Boolean,
  escribir: {type: String, default: ''}, // texto que el usuario debe escribir para confirmar
  ocupado: Boolean,
  bloquear: Boolean, // deshabilita confirmar (formulario incompleto)
})
const emit = defineEmits(['confirmar', 'cancelar'])
const escrito = ref('')
const caja = ref(null)
const valido = computed(() => !props.bloquear && (!props.escribir || escrito.value.trim() === props.escribir.trim()))

onMounted(async () => {
  await nextTick()
  caja.value?.querySelector('input, select, textarea, .btn.cancelar')?.focus()
})
</script>

<template>
  <div class="velo" @keydown.esc="emit('cancelar')" @click.self="emit('cancelar')">
    <div ref="caja" class="dialogo" :class="{peligro}" role="alertdialog" aria-modal="true" :aria-label="titulo">
      <header>
        <span v-if="peligro" class="icono-alerta"><Icono nombre="alerta"/></span>
        <h2>{{ titulo }}</h2>
      </header>
      <p v-if="mensaje" class="mensaje">{{ mensaje }}</p>
      <ul v-if="detalles.length" class="detalles">
        <li v-for="(d, i) in detalles" :key="i">{{ d }}</li>
      </ul>
      <slot/>
      <label v-if="escribir" class="escribir">
        Para confirmar, escriba <strong>{{ escribir }}</strong>
        <input v-model="escrito" class="campo-input" autocomplete="off" @keydown.enter="valido && emit('confirmar')"/>
      </label>
      <footer>
        <button class="btn cancelar" @click="emit('cancelar')">Cancelar</button>
        <button class="btn" :class="peligro ? 'peligro-lleno' : 'primario'" :disabled="!valido || ocupado"
                @click="emit('confirmar')">{{ ocupado ? 'Procesando…' : textoConfirmar }}</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.velo {
  position: fixed; inset: 0; z-index: 50;
  display: grid; place-items: center;
  background: rgba(16, 49, 59, .45);
}
.dialogo {
  width: min(520px, calc(100vw - 32px));
  max-height: calc(100vh - 64px);
  overflow: auto;
  padding: 22px 24px 18px;
  background: var(--superficie);
  border-radius: var(--r-m);
  border-top: 4px solid var(--riel);
  box-shadow: 0 24px 60px rgba(16, 49, 59, .3);
}
.dialogo.peligro { border-top-color: var(--rojo); }
header { display: flex; align-items: center; gap: 10px; }
h2 { margin: 0; font-size: var(--t-l); font-weight: 600; }
.icono-alerta { color: var(--rojo); display: inline-flex; }
.icono-alerta svg { width: 22px; height: 22px; }
.mensaje { margin: 10px 0 0; color: var(--tinta-2); }
.detalles {
  margin: 12px 0 0; padding: 10px 12px 10px 28px;
  background: var(--papel); border-radius: var(--r-s);
  font-size: var(--t-s); color: var(--tinta-2);
  max-height: 180px; overflow: auto;
}
.escribir { display: grid; gap: 6px; margin-top: 16px; font-size: var(--t-s); color: var(--tinta-2); }
footer { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
</style>
