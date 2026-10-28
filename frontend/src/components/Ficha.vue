<script setup>
import {computed, ref, watch} from 'vue'
import Icono from './Icono.vue'
import Retencion from './Retencion.vue'
import {padreDe, SIN_DEPURAR} from '../util'

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
const subBorrador = ref({})   // ruta subcarpeta -> {descripcion, retencion}

watch(() => [props.registro, props.editando, props.arbol], () => {
  borrador.value = {...(props.registro.valores || {})}
  ruta.value = props.registro.ruta || ''
  subBorrador.value = Object.fromEntries(props.arbol.map((s) => [s.ruta, {descripcion: s.descripcion || '', retencion: s.retencion || ''}]))
}, {immediate: true, deep: true})

const subsFiltradas = computed(() => (props.editando || mostrarVacios.value ? props.arbol : props.arbol.filter((s) => s.descripcion || s.retencion)))
// Con muchas subcarpetas se muestran las primeras 5 y un botón para ver todas / contraer.
const LIMITE_SUBS = 5
const verTodas = ref(false)
const subsVisibles = computed(() => (verTodas.value ? subsFiltradas.value : subsFiltradas.value.slice(0, LIMITE_SUBS)))
watch(() => props.registro?.ruta, () => { verTodas.value = false })
const subsDescritas = computed(() => props.arbol.filter((s) => s.descripcion).length)
const subsConRetencion = computed(() => props.arbol.filter((s) => s.retencion).length)

// Retención que heredaría una subcarpeta con lo que hay en el formulario (sube hasta la carpeta principal).
const campoRet = computed(() => props.campos.find((c) => c.tipo === 'retencion')?.nombre || 'Retención de archivos')
function heredadaEn(ruta) {
  for (let p = padreDe(ruta); p; p = padreDe(p)) {
    if (p === props.registro.ruta) return (props.editando ? borrador.value[campoRet.value] : val({nombre: campoRet.value})) || ''
    const v = props.editando ? subBorrador.value[p]?.retencion : props.arbol.find((x) => x.ruta === p)?.retencion
    if (v) return v
  }
  return ''
}
const textoRet = (v) => (v === SIN_DEPURAR ? 'no se depura' : v)

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
  const subs = props.arbol.map((s) => ({ruta: s.ruta, descripcion: (subBorrador.value[s.ruta]?.descripcion || '').trim(), retencion: subBorrador.value[s.ruta]?.retencion || '', actualizado: ''}))
  emit('guardar', {...props.registro, ruta: ruta.value, valores: {...borrador.value}}, subs)
}
// Multilista: varios valores en una celda, separados por ";" (p. ej. "Informes; Formatos").
const partes = (v) => (v || '').split(';').map((x) => x.trim()).filter(Boolean)
const nuevoValor = ref({})   // campo -> texto del valor que se está agregando
function tiene(c, o) { return partes(borrador.value[c.nombre]).some((x) => x.toLowerCase() === o.toLowerCase()) }
function alternarValor(c, o) {
  const l = partes(borrador.value[c.nombre])
  const i = l.findIndex((x) => x.toLowerCase() === o.toLowerCase())
  if (i >= 0) l.splice(i, 1); else l.push(o)
  borrador.value[c.nombre] = l.join('; ')
}
function agregarValor(c) {
  const o = (nuevoValor.value[c.nombre] || '').replaceAll(';', ',').trim()
  if (o && !tiene(c, o)) alternarValor(c, o)
  nuevoValor.value[c.nombre] = ''
}
// Opciones + los valores que ya tiene y no están en la lista (escritos a mano).
const opcionesDe = (c) => [...c.opciones, ...partes(borrador.value[c.nombre]).filter((x) => !c.opciones.some((o) => o.toLowerCase() === x.toLowerCase()))]

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
               :class="[{ancho: c.tipo === 'largo' || c.tipo === 'multilista' || c.tipo === 'retencion' || semaforo(c)}, semaforo(c) && 'sem-' + semaforo(c)]">
            <dt>
              <span v-if="semaforo(c)" class="luz"></span>
              {{ semaforo(c) ? nombreSemaforo[semaforo(c)] : etiqueta(c.nombre) }}
            </dt>
            <dd v-if="!editando && c.tipo === 'multilista'">
              <span v-if="!partes(val(c)).length">—</span>
              <span v-else class="valores"><span v-for="x in partes(val(c))" :key="x" class="valor">{{ x }}</span></span>
            </dd>
            <dd v-else-if="!editando && c.tipo === 'retencion'" class="ret-vista">
              <span v-if="val(c)" class="valor ret">{{ val(c) === 'No depurar' ? 'No se depura' : 'Conservar ' + val(c) }}</span>
              <span v-else>—</span>
              <small class="muted">{{ val(c) && val(c) !== 'No depurar'
                ? 'Los archivos modificados hace más de ese tiempo aparecen en la pestaña Vencidos. Las subcarpetas lo heredan si no tienen uno propio.'
                : 'Defina cuánto tiempo deben quedarse los archivos para poder depurarlos en la pestaña Vencidos.' }}</small>
            </dd>
            <dd v-else-if="!editando">{{ val(c) || '—' }}</dd>
            <dd v-else-if="c.tipo === 'retencion'" class="ret-edit">
              <Retencion v-model="borrador[c.nombre]"/>
              <small class="muted">Se compara con la fecha de modificación de cada archivo. Aplica a esta carpeta y a sus subcarpetas que no tengan uno propio.</small>
            </dd>
            <dd v-else-if="c.tipo === 'multilista'" class="multi">
              <div class="opciones-multi" role="group" :aria-label="etiqueta(c.nombre)">
                <button v-for="o in opcionesDe(c)" :key="o" type="button" class="opcion" :class="{on: tiene(c, o)}"
                        :aria-pressed="tiene(c, o)" @click="alternarValor(c, o)">
                  <span class="caja">{{ tiene(c, o) ? '✓' : '' }}</span>{{ o }}
                </button>
              </div>
              <div class="otro">
                <input v-model="nuevoValor[c.nombre]" class="campo-input" placeholder="Otro tipo…" @keydown.enter.prevent="agregarValor(c)"/>
                <button type="button" class="btn" :disabled="!(nuevoValor[c.nombre] || '').trim()" @click="agregarValor(c)">Agregar</button>
              </div>
              <small class="muted">{{ partes(borrador[c.nombre]).length }} seleccionado(s)</small>
            </dd>
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
      <h3>Subcarpetas <span class="cuenta">{{ subsDescritas }} de {{ arbol.length }} descritas<template v-if="subsConRetencion"> · {{ subsConRetencion }} con retención propia</template></span></h3>
      <p v-if="!arbol.length" class="nota-sub">Esta carpeta no tiene subcarpetas.</p>
      <p v-else-if="!subsVisibles.length" class="nota-sub">
        Ninguna subcarpeta tiene explicación todavía. Use “Editar ficha” para describir para qué sirve cada una.
      </p>
      <ul v-else class="lista-sub">
        <li v-for="s in subsVisibles" :key="s.ruta" :style="{'--n': s.nivel - 1}" :class="{editando}">
          <div class="nom-sub" :title="s.ruta"><Icono nombre="carpeta" class="ic-sub"/>{{ s.nombre }}</div>
          <div v-if="editando && subBorrador[s.ruta]" class="edit-sub">
            <textarea v-model="subBorrador[s.ruta].descripcion" class="campo-input" rows="1"
                      :placeholder="'¿Qué se guarda en ' + s.nombre + '?'"></textarea>
            <label class="ret-sub"><span>Retención</span>
              <Retencion v-model="subBorrador[s.ruta].retencion" subcarpeta compacto :heredada="heredadaEn(s.ruta)"/>
            </label>
          </div>
          <div v-else class="vista-sub">
            <p class="desc-sub" :class="{muted: !s.descripcion}">{{ s.descripcion || 'Sin descripción' }}</p>
            <span v-if="s.retencion" class="valor ret pequeno">Retención: {{ textoRet(s.retencion) }}</span>
            <span v-else-if="heredadaEn(s.ruta)" class="hereda">Retención: {{ textoRet(heredadaEn(s.ruta)) }} (heredada)</span>
          </div>
        </li>
      </ul>
      <button v-if="subsFiltradas.length > LIMITE_SUBS" type="button" class="ver-todas" :aria-expanded="verTodas" @click="verTodas = !verTodas">
        <Icono nombre="chevron" :class="{girado: verTodas}"/>
        {{ verTodas ? 'Contraer' : `Mostrar todas (${subsFiltradas.length - LIMITE_SUBS} más)` }}
      </button>
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
.ver-todas { display: inline-flex; align-items: center; gap: 6px; margin-top: 8px; padding: 5px 10px; border: 1px solid var(--linea); border-radius: var(--r-s); background: #fff; font-size: var(--t-s); color: var(--tinta-2); }
.ver-todas:hover { border-color: var(--tinta-3); color: var(--tinta); }
.ver-todas svg { width: 13px; height: 13px; transform: rotate(90deg); transition: transform .15s; }
.ver-todas svg.girado { transform: rotate(-90deg); }
.valores { display: flex; flex-wrap: wrap; gap: 6px; }
.valor { padding: 2px 10px; border-radius: 999px; background: var(--ambar-claro); border: 1px solid #f0d9a0; font-size: var(--t-s); }
.multi { display: grid; gap: 8px; }
.opciones-multi { display: flex; flex-wrap: wrap; gap: 6px; }
.opcion { display: inline-flex; align-items: center; gap: 6px; padding: 4px 10px 4px 6px; border: 1px solid var(--linea); border-radius: 999px; background: #fff; font-size: var(--t-s); color: var(--tinta-2); }
.opcion:hover { border-color: var(--tinta-3); }
.opcion .caja { display: grid; place-items: center; width: 16px; height: 16px; border: 1.5px solid var(--linea); border-radius: 4px; font-size: 11px; font-weight: 700; color: #fff; }
.opcion.on { border-color: var(--ambar-oscuro); background: var(--ambar-claro); color: var(--tinta); font-weight: 500; }
.opcion.on .caja { background: var(--ambar-oscuro); border-color: var(--ambar-oscuro); }
.otro { display: flex; gap: 6px; max-width: 420px; }
.otro .campo-input { flex: 1; }
.lista-sub textarea { min-height: 34px; resize: vertical; font-size: var(--t-s); width: 100%; }
.edit-sub { display: grid; gap: 6px; }
.ret-sub { display: flex; align-items: center; gap: 8px; font-size: var(--t-xs); color: var(--tinta-3); }
.vista-sub { display: grid; gap: 4px; justify-items: start; }
.ret-vista, .ret-edit { display: grid; gap: 6px; justify-items: start; }
.valor.ret { background: #e6f0f7; border-color: #bcd5e6; }
.valor.pequeno { font-size: var(--t-xs); padding: 0 8px; }
.hereda { font-size: var(--t-xs); color: var(--tinta-3); }
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
