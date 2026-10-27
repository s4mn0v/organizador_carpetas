<script setup>
import {computed, ref, watch} from 'vue'
import Icono from './Icono.vue'

const props = defineProps({
  registro: {type: Object, required: true},
  campos: {type: Array, required: true},
  editando: Boolean,
  rutas: {type: Array, default: () => []},
  ocupado: Boolean,
  arbol: {type: Array, default: () => []},   // subcarpetas de la carpeta principal: {ruta, nombre, nivel, descripcion}
  arbolCortado: Boolean,
})
const emit = defineEmits(['editar', 'guardar', 'cancelar', 'eliminar'])

// Orden de los grupos (los que no estén aquí van al final, en el orden de la hoja Campos).
const ORDEN = ['General']

const borrador = ref({})
const ruta = ref('')
const mostrarVacios = ref(false)
const subBorrador = ref({})   // ruta subcarpeta -> descripción

watch(() => [props.registro, props.editando, props.arbol], () => {
  borrador.value = {...(props.registro.valores || {})}
  ruta.value = props.registro.ruta || ''
  subBorrador.value = Object.fromEntries(props.arbol.map((s) => [s.ruta, s.descripcion || '']))
}, {immediate: true, deep: true})

const subsVisibles = computed(() => (props.editando || mostrarVacios.value ? props.arbol : props.arbol.filter((s) => s.descripcion)))
const subsDescritas = computed(() => props.arbol.filter((s) => s.descripcion).length)

const grupos = computed(() => {
  const m = new Map()
  for (const c of props.campos) {
    if (!m.has(c.grupo)) m.set(c.grupo, [])
    m.get(c.grupo).push(c)
  }
  return [...m.entries()].sort((a, b) => idx(a[0]) - idx(b[0])).map(([nombre, campos]) => ({nombre, campos}))
})
function idx(g) { if (g === 'Otros') return 99; const i = ORDEN.indexOf(g); return i < 0 ? 50 : i }

function etiqueta(n) {
  let s = n.replace(/^(Requisito|Factor):\s*/, '').trim()
  if (s === s.toUpperCase()) s = s.charAt(0) + s.slice(1).toLowerCase()
  return s
}
const val = (c) => (props.registro.valores || {})[c.nombre] || ''
function visibles(g) {
  if (props.editando || mostrarVacios.value) return g.campos
  return g.campos.filter((c) => val(c))
}
function semaforo(c) {
  const n = c.nombre.toLowerCase()
  if (n.includes('correctiva')) return 'rojo'
  if (n.includes('preventiva')) return 'amarillo'
  if (n.includes('intervalo que cumple')) return 'verde'
  return ''
}
const nombreSemaforo = {rojo: 'No cumple — acción correctiva', amarillo: 'En riesgo — acción preventiva', verde: 'Cumple'}
const marcado = (v) => /^x$/i.test((v || '').trim()) || v === '1' || /^s[ií]$/i.test((v || '').trim())

const vacioTotal = computed(() => Object.values(props.registro.valores || {}).every((v) => !v) && !subsDescritas.value)

function guardar() {
  const subs = props.arbol.map((s) => ({ruta: s.ruta, descripcion: (subBorrador.value[s.ruta] || '').trim(), actualizado: ''}))
  emit('guardar', {...props.registro, ruta: ruta.value, valores: {...borrador.value}}, subs)
}
function alternarMarca(c) {
  borrador.value[c.nombre] = marcado(borrador.value[c.nombre]) ? '' : 'X'
}
</script>

<template>
  <article class="ficha" :class="{editando}">
    <header class="barra">
      <div class="meta">
        <span>Ficha #{{ registro.id || 'nueva' }}</span>
        <span v-if="registro.actualizado" class="muted">Actualizada {{ registro.actualizado }}</span>
      </div>
      <div class="acciones">
        <template v-if="!editando">
          <label class="vacios"><input type="checkbox" v-model="mostrarVacios"/> Mostrar campos vacíos</label>
          <button class="btn" @click="emit('editar')"><Icono nombre="editar"/>Editar ficha</button>
          <button class="btn peligro" @click="emit('eliminar')"><Icono nombre="borrar"/>Eliminar ficha</button>
        </template>
        <template v-else>
          <button class="btn" :disabled="ocupado" @click="emit('cancelar')">Cancelar</button>
          <button class="btn primario" :disabled="ocupado" @click="guardar">{{ ocupado ? 'Guardando…' : 'Guardar en Excel' }}</button>
        </template>
      </div>
    </header>

    <p v-if="!editando && vacioTotal" class="vacia">
      Esta ficha no tiene información todavía. Use “Editar ficha” para llenarla.
    </p>

    <div v-if="editando && rutas.length" class="grupo">
      <h3>Carpeta asociada</h3>
      <select v-model="ruta" class="campo-input">
        <option value="">Sin carpeta</option>
        <option v-for="r in rutas" :key="r" :value="r">{{ r.replace('/*', '/') }}</option>
      </select>
    </div>

    <template v-for="g in grupos" :key="g.nombre">
      <section v-if="visibles(g).length" class="grupo" :class="'g-' + idx(g.nombre)">
        <h3>{{ g.nombre }}</h3>

        <!-- Requisitos: marcas -->
        <div v-if="g.campos[0]?.tipo === 'marca'" class="chips">
          <template v-for="c in visibles(g)" :key="c.nombre">
            <button v-if="editando" type="button" class="chip" :class="{on: marcado(borrador[c.nombre])}"
                    :aria-pressed="marcado(borrador[c.nombre])" @click="alternarMarca(c)">{{ etiqueta(c.nombre) }}</button>
            <span v-else class="chip" :class="{on: marcado(val(c))}">{{ etiqueta(c.nombre) }}</span>
          </template>
        </div>

        <!-- Histórico: rejilla de años -->
        <div v-else-if="g.nombre === 'Histórico de promedios'" class="anios">
          <label v-for="c in visibles(g)" :key="c.nombre" class="anio">
            <span>{{ etiqueta(c.nombre).replace(/promedio\s*/i, '').replace(/año\s*/i, '') }}</span>
            <input v-if="editando" v-model="borrador[c.nombre]" class="campo-input" inputmode="decimal"/>
            <strong v-else>{{ val(c) || '—' }}</strong>
          </label>
        </div>

        <dl v-else class="campos">
          <div v-for="c in visibles(g)" :key="c.nombre" class="campo"
               :class="[{ancho: c.tipo === 'largo' || semaforo(c)}, semaforo(c) && 'sem-' + semaforo(c)]">
            <dt>
              <span v-if="semaforo(c)" class="luz"></span>
              {{ semaforo(c) ? nombreSemaforo[semaforo(c)] : etiqueta(c.nombre) }}
            </dt>
            <dd v-if="!editando">{{ val(c) || '—' }}</dd>
            <dd v-else>
              <textarea v-if="c.tipo === 'largo' || semaforo(c)" v-model="borrador[c.nombre]" class="campo-input" rows="3"></textarea>
              <select v-else-if="c.tipo === 'lista'" v-model="borrador[c.nombre]" class="campo-input">
                <option value=""></option>
                <option v-for="o in c.opciones" :key="o" :value="o">{{ o }}</option>
                <option v-if="borrador[c.nombre] && !c.opciones.includes(borrador[c.nombre])" :value="borrador[c.nombre]">{{ borrador[c.nombre] }}</option>
              </select>
              <input v-else v-model="borrador[c.nombre]" class="campo-input"
                     :inputmode="c.tipo === 'numero' ? 'decimal' : 'text'"/>
            </dd>
          </div>
        </dl>
      </section>
    </template>
    <!-- Subcarpetas: explicación de cada una -->
    <section class="grupo subcarpetas">
      <h3>Subcarpetas <span class="cuenta">{{ subsDescritas }} de {{ arbol.length }} descritas</span></h3>
      <p v-if="!arbol.length" class="nota-sub">Esta carpeta no tiene subcarpetas.</p>
      <p v-else-if="!subsVisibles.length" class="nota-sub">
        Ninguna subcarpeta tiene explicación todavía. Use “Editar ficha” para describir para qué sirve cada una.
      </p>
      <ul v-else class="lista-sub">
        <li v-for="s in subsVisibles" :key="s.ruta" :style="{'--n': s.nivel - 1}" :class="{editando}">
          <div class="nom-sub" :title="s.ruta"><Icono nombre="carpeta" class="ic-sub"/>{{ s.nombre }}</div>
          <textarea v-if="editando" v-model="subBorrador[s.ruta]" class="campo-input" rows="1"
                    :placeholder="'¿Qué se guarda en ' + s.nombre + '?'"></textarea>
          <p v-else class="desc-sub" :class="{muted: !s.descripcion}">{{ s.descripcion || 'Sin descripción' }}</p>
        </li>
      </ul>
      <p v-if="arbolCortado" class="nota-sub">Hay demasiadas subcarpetas: se muestran las primeras 600.</p>
    </section>
  </article>
</template>

<style scoped>
.ficha { background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); }
.ficha.editando { border-color: var(--foco); box-shadow: 0 0 0 3px rgba(43, 122, 155, .1); }
.barra {
  display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap;
  padding: 10px 16px; border-bottom: 1px solid var(--linea-2);
  position: sticky; top: 0; z-index: 2; background: inherit; border-radius: var(--r-m) var(--r-m) 0 0;
}
.meta { display: flex; gap: 14px; font-size: var(--t-s); font-weight: 600; }
.meta .muted { font-weight: 400; }
.acciones { display: flex; gap: 8px; align-items: center; }
.vacios { display: flex; gap: 6px; align-items: center; font-size: var(--t-s); color: var(--tinta-2); margin-right: 6px; }
.vacia { margin: 0; padding: 18px 16px; color: var(--tinta-2); }
.grupo { padding: 14px 16px 16px; border-bottom: 1px solid var(--linea-2); }
.grupo:last-child { border-bottom: 0; }
.subcarpetas h3 .cuenta { margin-left: 6px; font-weight: 400; color: var(--tinta-3); }
.nota-sub { margin: 0; font-size: var(--t-s); color: var(--tinta-2); }
.lista-sub { list-style: none; margin: 0; padding: 0; display: grid; gap: 2px; }
.lista-sub li { display: grid; grid-template-columns: minmax(180px, 32%) 1fr; gap: 12px; align-items: start; padding: 6px 0 6px calc(var(--n) * 18px); border-bottom: 1px dashed var(--linea-2); }
.lista-sub li:last-child { border-bottom: 0; }
.nom-sub { display: flex; align-items: center; gap: 6px; font-size: var(--t-s); font-weight: 500; overflow-wrap: anywhere; padding-top: 2px; }
.ic-sub { width: 15px; height: 15px; flex: none; color: var(--ambar-oscuro); }
.desc-sub { margin: 0; font-size: var(--t-s); white-space: pre-wrap; }
.lista-sub textarea { min-height: 34px; resize: vertical; font-size: var(--t-s); }
h3 { margin: 0 0 10px; font-size: var(--t-s); font-weight: 600; color: var(--riel); }
.campos {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px 24px; margin: 0;
}
.campo { min-width: 0; }
.campo.ancho { grid-column: 1 / -1; }
dt { font-size: var(--t-xs); color: var(--tinta-3); margin-bottom: 2px; display: flex; align-items: center; gap: 6px; }
dd { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; }
.luz { width: 10px; height: 10px; border-radius: 50%; flex: none; }
.sem-rojo .luz { background: var(--rojo); }
.sem-amarillo .luz { background: var(--amarillo); }
.sem-verde .luz { background: var(--verde); }
.sem-rojo, .sem-amarillo, .sem-verde { padding: 8px 12px; border-radius: var(--r-s); }
.sem-rojo { background: var(--rojo-claro); }
.sem-amarillo { background: var(--amarillo-claro); }
.sem-verde { background: var(--verde-claro); }
.sem-rojo dt, .sem-amarillo dt, .sem-verde dt { color: var(--tinta-2); font-weight: 600; }
.chips { display: flex; flex-wrap: wrap; gap: 6px; }
.chip {
  padding: 3px 10px; border-radius: 999px; font-size: var(--t-s);
  border: 1px dashed var(--linea); color: var(--tinta-3); background: none;
}
.chip.on { border: 1px solid var(--riel); background: var(--riel); color: #fff; }
button.chip:hover { border-color: var(--riel); }
.anios { display: grid; grid-template-columns: repeat(auto-fill, minmax(84px, 1fr)); gap: 8px; }
.anio { display: grid; gap: 2px; padding: 6px 8px; background: var(--papel); border-radius: var(--r-s); }
.anio span { font-size: var(--t-xs); color: var(--tinta-3); }
.anio strong { font-weight: 600; }
</style>
