<script setup>
import {computed, onBeforeUnmount, onMounted, reactive, ref} from 'vue'
import Icono from './Icono.vue'
import {CancelarVencidos, Vencidos} from '../../wailsjs/go/main/App'
import {EventsOff, EventsOn} from '../../wailsjs/runtime/runtime'
import {fecha, fechaHora, filtrarPorTexto, mensajeError, raizDeRuta, relDe, rutaWin, tamano, tamanoLargo, tipoArchivo} from '../util'

// Depuración: archivos cuya fecha de modificación pasó el tiempo de retención de la ficha técnica.
// Nada se borra solo: el usuario revisa, marca y confirma; los archivos van a la papelera de la aplicación.
const props = defineProps({
  alcance: {type: String, default: ''},
  raices: {type: Object, default: () => ({})},
})
const emit = defineEmits(['abrir', 'mostrar', 'ir', 'depurar'])

const resultado = ref(null)
const buscadoEn = ref('')
const buscando = ref(false)
const error = ref('')
const progreso = ref(null)
const filtro = ref('')
const marcados = reactive({})   // ruta -> true
const cerrados = reactive({})   // origen -> true (grupo contraído)

function rutaVisible(r) {
  if (!r) return 'Todas las carpetas raíz'
  const n = props.raices[raizDeRuta(r)]?.nombre || ''
  return relDe(r) ? `${n}\\${rutaWin(relDe(r))}` : n
}
const dentroDe = (r, base) => r === base || r.startsWith(base + '/')

async function revisar() {
  buscando.value = true
  error.value = ''
  progreso.value = {revisados: 0, encontrados: 0}
  buscadoEn.value = props.alcance
  for (const k of Object.keys(marcados)) delete marcados[k]
  try {
    resultado.value = await Vencidos(props.alcance)
  } catch (e) {
    error.value = mensajeError(e)
  } finally {
    buscando.value = false
    progreso.value = null
  }
}
const cancelar = () => CancelarVencidos()

// Después de depurar, se quitan de la lista.
function quitar(rutas) {
  if (!resultado.value) return
  const s = new Set(rutas)
  resultado.value.archivos = resultado.value.archivos.filter((a) => !s.has(a.ruta))
  for (const r of rutas) delete marcados[r]
}
defineExpose({quitar})

const archivos = computed(() => {
  const l = resultado.value?.archivos || []
  return filtro.value.trim() ? filtrarPorTexto(l, filtro.value) : l
})
// Agrupados por la carpeta donde se definió la regla.
const grupos = computed(() => {
  const m = new Map()
  for (const a of archivos.value) {
    if (!m.has(a.origen)) m.set(a.origen, {origen: a.origen, retencion: a.retencion, archivos: [], tamano: 0})
    const g = m.get(a.origen)
    g.archivos.push(a)
    g.tamano += a.tamano
  }
  return [...m.values()]
})
const seleccion = computed(() => (resultado.value?.archivos || []).filter((a) => marcados[a.ruta]))
const tamSeleccion = computed(() => seleccion.value.reduce((s, a) => s + a.tamano, 0))
const totalTam = computed(() => (resultado.value?.archivos || []).reduce((s, a) => s + a.tamano, 0))
const todosGrupo = (g) => g.archivos.every((a) => marcados[a.ruta])
const algunoGrupo = (g) => g.archivos.some((a) => marcados[a.ruta])
function marcarGrupo(g, v) { for (const a of g.archivos) v ? (marcados[a.ruta] = true) : delete marcados[a.ruta] }
function marcarTodos(v) { for (const a of archivos.value) v ? (marcados[a.ruta] = true) : delete marcados[a.ruta] }
function alternar(a) { marcados[a.ruta] ? delete marcados[a.ruta] : (marcados[a.ruta] = true) }
function textoDias(d) {
  if (d < 1) return 'hoy'
  if (d < 60) return `hace ${d} días`
  if (d < 730) return `hace ${Math.round(d / 30.4)} meses`
  return `hace ${(d / 365).toFixed(1).replace('.', ',')} años`
}
const depurar = () => emit('depurar', seleccion.value.map((a) => a.ruta), tamSeleccion.value)

onMounted(() => EventsOn('vencidos:progreso', (p) => { if (buscando.value) progreso.value = p }))
onBeforeUnmount(() => EventsOff('vencidos:progreso'))
</script>

<template>
  <section class="vista-venc">
    <header>
      <h1>Archivos vencidos</h1>
      <p class="sub">Archivos cuya <b>fecha de modificación</b> es más antigua que el tiempo de retención de su carpeta
        (campo <i>Retención de archivos</i> de la ficha técnica, o el de la subcarpeta). Revise, marque y muévalos a la papelera de la aplicación: se pueden recuperar.</p>
    </header>

    <div class="barra-busq">
      <div class="donde">
        <span class="muted">Revisar en</span>
        <strong :style="{'--cf': raices[raizDeRuta(alcance)]?.color || 'var(--ambar)'}"><Icono :nombre="alcance ? 'carpeta' : 'casa'" class="ic-alc"/>{{ rutaVisible(alcance) }}</strong>
        <small class="muted">Para cambiarla, elija otra carpeta en la barra lateral.</small>
      </div>
      <button v-if="!buscando" class="btn primario" @click="revisar"><Icono nombre="reloj"/>Revisar vencidos</button>
      <button v-else class="btn" @click="cancelar"><Icono nombre="cerrar"/>Detener</button>
    </div>

    <div v-if="buscando" class="progreso">
      <div class="barra"><span class="indef"></span></div>
      <p>Revisando… {{ progreso?.revisados || 0 }} archivos, {{ progreso?.encontrados || 0 }} vencidos</p>
    </div>
    <div v-else-if="error" class="aviso error"><Icono nombre="alerta"/><div>{{ error }}</div></div>

    <template v-if="resultado && !buscando">
      <div v-if="resultado.cancelado" class="aviso"><Icono nombre="alerta"/><div>Revisión detenida: la lista está incompleta.</div></div>
      <div v-if="resultado.cortado" class="aviso"><Icono nombre="alerta"/><div>Hay demasiados vencidos: se muestran los primeros {{ resultado.archivos.length }}. Depúrelos y vuelva a revisar.</div></div>
      <div v-if="resultado.sinExcel.length" class="aviso"><Icono nombre="excel"/><div>Sin Excel de fichas (no se revisaron): {{ resultado.sinExcel.join(', ') }}</div></div>

      <div v-if="!resultado.reglas" class="sin-ficha">
        <p>Ninguna carpeta de aquí tiene <b>tiempo de retención</b>. Defínalo en la ficha técnica de la carpeta principal
          (campo <i>Retención de archivos</i>) o en sus subcarpetas, y vuelva a revisar.</p>
      </div>
      <template v-else>
        <dl class="cifras">
          <div><dt>Archivos revisados</dt><dd>{{ resultado.revisados.toLocaleString('es-CO') }}</dd></div>
          <div><dt>Vencidos</dt><dd>{{ resultado.archivos.length.toLocaleString('es-CO') }}<small>{{ tamanoLargo(totalTam) }}</small></dd></div>
          <div><dt>Carpetas con retención</dt><dd>{{ resultado.reglas }}</dd></div>
          <div><dt>Revisado en</dt><dd class="pequeno">{{ rutaVisible(buscadoEn) }}<small>{{ Math.round(resultado.segundos) }} s</small></dd></div>
        </dl>

        <div v-if="!resultado.archivos.length" class="sin-ficha"><p>No hay archivos vencidos. Todo está dentro de su tiempo de retención.</p></div>
        <template v-else>
          <div class="herr">
            <label class="filtro"><Icono nombre="buscar"/><input v-model="filtro" placeholder="Filtrar por nombre"/></label>
            <button class="btn fantasma" @click="marcarTodos(true)">Marcar {{ filtro.trim() ? 'los filtrados' : 'todos' }}</button>
            <button class="btn fantasma" :disabled="!seleccion.length" @click="marcarTodos(false)">Desmarcar</button>
          </div>

          <section v-for="g in grupos" :key="g.origen" class="grupo">
            <header class="cab-grupo">
              <input type="checkbox" :checked="todosGrupo(g)" :indeterminate.prop="!todosGrupo(g) && algunoGrupo(g)"
                     :aria-label="'Marcar todos los de ' + rutaVisible(g.origen)" @change="marcarGrupo(g, $event.target.checked)"/>
              <button class="enlace-carpeta" :style="{'--cf': raices[raizDeRuta(g.origen)]?.color || 'var(--ambar)'}" @click="emit('ir', g.origen)">
                <Icono nombre="carpeta" class="ic-c"/>{{ rutaVisible(g.origen) }}
              </button>
              <span class="regla">Conservar {{ g.retencion }}</span>
              <span class="dato">{{ g.archivos.length }} archivo(s) · {{ tamanoLargo(g.tamano) }}</span>
              <button class="btn fantasma ic" :aria-expanded="!cerrados[g.origen]" @click="cerrados[g.origen] = !cerrados[g.origen]">
                <Icono nombre="chevron" class="chev" :class="{girado: !cerrados[g.origen]}"/>
              </button>
            </header>
            <div v-if="!cerrados[g.origen]" class="tabla-caja">
              <table class="tabla">
                <thead><tr><th></th><th>Archivo</th><th class="der">Tamaño</th><th>Modificado</th><th>Venció</th><th></th></tr></thead>
                <tbody>
                  <tr v-for="a in g.archivos.slice(0, 500)" :key="a.ruta" :class="{marcado: marcados[a.ruta]}" @click="alternar(a)">
                    <td class="chk"><input type="checkbox" :checked="!!marcados[a.ruta]" @click.stop @change="alternar(a)"/></td>
                    <td><div class="nom">
                      <span class="ic arch" :style="{'--c': tipoArchivo(a.ext).color}"><Icono nombre="archivo"/></span>
                      <span class="txt"><b>{{ a.nombre }}</b><small v-if="a.carpeta !== g.origen">{{ rutaWin(a.carpeta.slice(g.origen.length + 1)) }}</small></span>
                    </div></td>
                    <td class="der">{{ tamano(a.tamano) }}</td>
                    <td class="fecha" :title="fechaHora(a.modificado)">{{ fecha(a.modificado) }}</td>
                    <td class="fecha" :title="'Venció el ' + fecha(a.vencio)">{{ textoDias(a.dias) }}</td>
                    <td class="acc" @click.stop>
                      <button class="btn fantasma ic" title="Abrir" @click="emit('abrir', a.ruta)"><Icono nombre="abrir"/></button>
                      <button class="btn fantasma ic" title="Mostrar en el Explorador" @click="emit('mostrar', a.ruta)"><Icono nombre="carpeta-abierta"/></button>
                    </td>
                  </tr>
                  <tr v-if="g.archivos.length > 500"><td colspan="6" class="muted">… y {{ g.archivos.length - 500 }} más (use “Marcar” en el encabezado del grupo para incluirlos).</td></tr>
                </tbody>
              </table>
            </div>
          </section>

          <div class="pie-depurar">
            <span><b>{{ seleccion.length }}</b> marcado(s) · {{ tamanoLargo(tamSeleccion) }}</span>
            <button class="btn peligro" :disabled="!seleccion.length" @click="depurar"><Icono nombre="papelera"/>Mover a la papelera</button>
          </div>
        </template>
      </template>
    </template>

    <div v-if="!resultado && !buscando && !error" class="sin-ficha">
      <p>Elija una carpeta en la barra lateral (o el Inicio para revisar todas las raíces) y presione <b>Revisar vencidos</b>.</p>
    </div>
  </section>
</template>

<style scoped>
.vista-venc { max-width: 1180px; margin: 0 auto; padding: 24px 40px 90px; }
.sub { margin: 4px 0 16px; color: var(--tinta-2); max-width: 86ch; }
.barra-busq { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 16px; margin-bottom: 16px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); flex-wrap: wrap; }
.donde { display: grid; gap: 2px; min-width: 0; }
.donde strong { display: flex; align-items: center; gap: 6px; overflow-wrap: anywhere; }
.ic-alc { width: 18px; height: 18px; color: var(--cf); flex: none; }
.progreso { display: grid; gap: 6px; padding: 14px 16px; margin-bottom: 16px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); }
.progreso p { margin: 0; font-size: var(--t-s); }
.barra { height: 6px; border-radius: 99px; background: var(--linea-2); overflow: hidden; }
.barra span { display: block; height: 100%; width: 30%; background: var(--foco); animation: ir 1.4s ease-in-out infinite; }
@keyframes ir { from { transform: translateX(-100%); } to { transform: translateX(350%); } }
.herr { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
.grupo { margin-bottom: 10px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); overflow: hidden; }
.cab-grupo { display: flex; align-items: center; gap: 10px; padding: 8px 12px; background: #f6f8f9; flex-wrap: wrap; }
.cab-grupo input { accent-color: var(--rojo); width: 16px; height: 16px; }
.enlace-carpeta { display: flex; align-items: center; gap: 6px; border: 0; background: none; padding: 0; font-weight: 600; font-size: var(--t-s); text-align: left; overflow-wrap: anywhere; }
.enlace-carpeta:hover { text-decoration: underline; }
.ic-c { width: 16px; height: 16px; color: var(--cf); flex: none; }
.regla { font-size: var(--t-xs); padding: 1px 8px; border-radius: 99px; background: #e6f0f7; border: 1px solid #bcd5e6; }
.dato { flex: 1; text-align: right; font-size: var(--t-xs); color: var(--tinta-2); }
.chev { width: 13px; height: 13px; transition: transform .12s; }
.chev.girado { transform: rotate(90deg); }
.grupo .tabla-caja { border: 0; border-top: 1px solid var(--linea-2); border-radius: 0; max-height: 520px; }
.tabla tbody tr { cursor: pointer; }
.tabla tr.marcado { background: var(--rojo-claro); }
.chk { width: 1%; }
.chk input { accent-color: var(--rojo); }
.nom { display: flex; align-items: center; gap: 8px; }
.nom .txt { display: grid; min-width: 0; }
.nom b { font-weight: 500; overflow-wrap: anywhere; }
.nom small { color: var(--tinta-3); font-size: var(--t-xs); overflow-wrap: anywhere; }
.ic { width: 16px; height: 16px; flex: none; }
.ic.arch { display: inline-grid; place-items: center; color: var(--c); }
.ic.arch svg { width: 15px; height: 15px; }
.fecha { white-space: nowrap; color: var(--tinta-2); }
.pie-depurar { position: sticky; bottom: 0; display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 12px; padding: 12px 16px;
  background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); box-shadow: 0 -6px 20px rgba(16, 49, 59, .08); }
</style>
