<script setup>
import {computed, onBeforeUnmount, onMounted, reactive, ref} from 'vue'
import Icono from './Icono.vue'
import {CancelarDuplicados, Duplicados} from '../../wailsjs/go/main/App'
import {EventsOff, EventsOn} from '../../wailsjs/runtime/runtime'
import {fechaHora, filtrarPorTexto, mensajeError, raizDeRuta, relDe, rutaWin, tamano, tamanoLargo, tipoArchivo} from '../util'

// Archivos duplicados: mismo nombre y mismo contenido, en la carpeta elegida y todas sus subcarpetas.
const props = defineProps({
  alcance: {type: String, default: ''},          // carpeta donde se busca ('' = todas las raíces)
  raices: {type: Object, default: () => ({})},   // id -> {nombre, color}
})
const emit = defineEmits(['abrir', 'mostrar', 'ir', 'eliminar'])

const resultado = ref(null)
const buscadoEn = ref('')
const buscando = ref(false)
const error = ref('')
const progreso = ref(null)
const filtro = ref('')
const orden = ref('desperdicio')
const abiertos = reactive({})

function rutaVisible(r) {
  if (!r) return 'Todas las carpetas raíz'
  const n = props.raices[raizDeRuta(r)]?.nombre || ''
  return relDe(r) ? `${n}\\${rutaWin(relDe(r))}` : n
}

async function buscar() {
  buscando.value = true
  error.value = ''
  progreso.value = {fase: 'recorrer', hechos: 0, total: 0, revisados: 0}
  buscadoEn.value = props.alcance
  for (const k of Object.keys(abiertos)) delete abiertos[k]
  try {
    resultado.value = await Duplicados(props.alcance)
    if (resultado.value.grupos.length <= 3) resultado.value.grupos.forEach((g) => (abiertos[g.huella + g.nombre] = true))
  } catch (e) {
    error.value = mensajeError(e)
  } finally {
    buscando.value = false
    progreso.value = null
  }
}
const cancelar = () => CancelarDuplicados()

// Cuando se elimina una copia desde aquí, se quita de la lista.
function quitar(ruta) {
  const r = resultado.value
  if (!r) return
  for (const g of r.grupos) {
    const i = g.copias.findIndex((c) => c.ruta === ruta)
    if (i < 0) continue
    g.copias.splice(i, 1)
    r.desperdicio -= g.tamano
    g.desperdicio = g.tamano * Math.max(0, g.copias.length - 1)
  }
  r.grupos = r.grupos.filter((g) => g.copias.length > 1)
}
defineExpose({quitar})

const grupos = computed(() => {
  let l = resultado.value?.grupos || []
  if (filtro.value.trim()) l = filtrarPorTexto(l, filtro.value)
  else {
    l = [...l]
    const k = {desperdicio: (g) => -g.desperdicio, copias: (g) => -g.copias.length, tamano: (g) => -g.tamano, nombre: (g) => g.nombre.toLowerCase()}[orden.value]
    l.sort((a, b) => (k(a) < k(b) ? -1 : k(a) > k(b) ? 1 : 0))
  }
  return l
})
const totalCopias = computed(() => (resultado.value?.grupos || []).reduce((s, g) => s + g.copias.length, 0))
const pct = computed(() => {
  const p = progreso.value
  return p && p.fase === 'comparar' && p.total ? Math.round((p.hechos / p.total) * 100) : null
})
const alternar = (g) => (abiertos[g.huella + g.nombre] = !abiertos[g.huella + g.nombre])
const abierto = (g) => !!abiertos[g.huella + g.nombre]
// La copia más antigua (por fecha de modificación) se marca como "original".
const masAntigua = (g) => g.copias.reduce((a, c) => (c.modificado < a.modificado ? c : a), g.copias[0])

onMounted(() => EventsOn('duplicados:progreso', (p) => { if (buscando.value) progreso.value = p }))
onBeforeUnmount(() => EventsOff('duplicados:progreso'))
</script>

<template>
  <section class="vista-dup">
    <header class="cab">
      <div>
        <h1>Archivos duplicados</h1>
        <p class="sub">Mismo <b>nombre</b> y mismo <b>contenido</b> (se compara byte a byte con una huella SHA-256), en la carpeta elegida y todas sus subcarpetas.</p>
      </div>
    </header>

    <div class="barra-busq">
      <div class="donde">
        <span class="muted">Buscar en</span>
        <strong :style="{'--cf': raices[raizDeRuta(alcance)]?.color || 'var(--ambar)'}"><Icono :nombre="alcance ? 'carpeta' : 'casa'" class="ic-alc"/>{{ rutaVisible(alcance) }}</strong>
        <small class="muted">Para cambiarla, elija otra carpeta en la barra lateral.</small>
      </div>
      <button v-if="!buscando" class="btn primario" @click="buscar"><Icono nombre="buscar"/>Buscar duplicados</button>
      <button v-else class="btn" @click="cancelar"><Icono nombre="cerrar"/>Detener</button>
    </div>

    <div v-if="buscando" class="progreso">
      <div class="barra"><span :style="{width: (pct ?? 8) + '%'}" :class="{indef: pct === null}"></span></div>
      <p v-if="!progreso || progreso.fase === 'recorrer'">Recorriendo carpetas… {{ progreso?.revisados || 0 }} archivos</p>
      <p v-else>Comparando contenido de los posibles duplicados… {{ progreso.hechos }} de {{ progreso.total }} ({{ pct }}%)</p>
      <small class="muted">En unidades de red puede tardar varios minutos la primera vez.</small>
    </div>
    <div v-else-if="error" class="aviso error"><Icono nombre="alerta"/><div>{{ error }}</div></div>

    <template v-if="resultado && !buscando">
      <div v-if="resultado.cancelado" class="aviso"><Icono nombre="alerta"/><div>Búsqueda detenida: los resultados están incompletos.</div></div>
      <dl class="cifras">
        <div><dt>Archivos revisados</dt><dd>{{ resultado.revisados.toLocaleString('es-CO') }}</dd></div>
        <div><dt>Archivos repetidos</dt><dd>{{ resultado.grupos.length }}<small>{{ totalCopias }} copias en total</small></dd></div>
        <div><dt>Espacio que ocupan las copias de más</dt><dd>{{ tamanoLargo(resultado.desperdicio) }}</dd></div>
        <div><dt>Buscado en</dt><dd class="pequeno">{{ rutaVisible(buscadoEn) }}<small>{{ Math.round(resultado.segundos) }} s<template v-if="resultado.errores"> · {{ resultado.errores }} sin poder leer</template></small></dd></div>
      </dl>

      <div v-if="!resultado.grupos.length" class="sin-ficha"><p>No hay archivos duplicados (mismo nombre y mismo contenido) en esta carpeta.</p></div>
      <template v-else>
        <div class="herr">
          <label class="filtro"><Icono nombre="buscar"/><input v-model="filtro" placeholder="Filtrar por nombre"/></label>
          <label class="ord">Ordenar
            <select v-model="orden" :disabled="!!filtro.trim()">
              <option value="desperdicio">Más espacio desperdiciado</option>
              <option value="copias">Más copias</option>
              <option value="tamano">Más pesados</option>
              <option value="nombre">Nombre</option>
            </select>
          </label>
          <span class="muted">{{ grupos.length }} de {{ resultado.grupos.length }}</span>
        </div>

        <ul class="grupos">
          <li v-for="g in grupos" :key="g.huella + g.nombre" class="grupo" :class="{abierto: abierto(g)}">
            <button class="cab-grupo" :aria-expanded="abierto(g)" @click="alternar(g)">
              <Icono nombre="chevron" class="chev" :class="{girado: abierto(g)}"/>
              <span class="ic arch" :style="{'--c': tipoArchivo(g.ext).color}"><Icono nombre="archivo"/></span>
              <span class="nom">{{ g.nombre }}</span>
              <span class="dato">{{ g.copias.length }} copias</span>
              <span class="dato">{{ tamano(g.tamano) }} c/u</span>
              <span class="dato fuerte" title="Espacio que se liberaría dejando una sola copia">{{ tamano(g.desperdicio) }} de más</span>
            </button>
            <div v-if="abierto(g)" class="tabla-caja">
              <table class="tabla">
                <thead><tr><th>Ubicación</th><th class="der">Tamaño</th><th>Creado</th><th>Modificado</th><th></th></tr></thead>
                <tbody>
                  <tr v-for="c in g.copias" :key="c.ruta">
                    <td><div class="ubic">
                      <button class="enlace-carpeta" :style="{'--cf': raices[raizDeRuta(c.ruta)]?.color || 'var(--ambar)'}" :title="c.ubicacion" @click="emit('ir', c.carpeta)">
                        <Icono nombre="carpeta" class="ic-c"/>{{ rutaVisible(c.carpeta) }}
                      </button>
                      <span v-if="c === masAntigua(g)" class="etq" title="La copia con la fecha de modificación más antigua">más antigua</span>
                    </div></td>
                    <td class="der">{{ tamano(c.tamano) }}</td>
                    <td class="fecha">{{ fechaHora(c.creado) }}</td>
                    <td class="fecha">{{ fechaHora(c.modificado) }}</td>
                    <td class="acc">
                      <button class="btn fantasma ic" title="Abrir" @click="emit('abrir', c.ruta)"><Icono nombre="abrir"/></button>
                      <button class="btn fantasma ic" title="Mostrar en el Explorador" @click="emit('mostrar', c.ruta)"><Icono nombre="carpeta-abierta"/></button>
                      <button class="btn fantasma ic rojo" title="Eliminar esta copia (va a la papelera de la aplicación)" @click="emit('eliminar', c.ruta)"><Icono nombre="borrar"/></button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </li>
        </ul>
      </template>
    </template>

    <div v-if="!resultado && !buscando && !error" class="sin-ficha">
      <p>Elija una carpeta en la barra lateral (o el Inicio para revisar todas las raíces) y presione <b>Buscar duplicados</b>.</p>
    </div>
  </section>
</template>

<style scoped>
.vista-dup { max-width: 1180px; margin: 0 auto; padding: 24px 40px 64px; }
.cab h1 { margin-bottom: 4px; }
.sub { margin: 0 0 16px; color: var(--tinta-2); max-width: 80ch; }
.barra-busq { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 16px; margin-bottom: 16px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); flex-wrap: wrap; }
.donde { display: grid; gap: 2px; min-width: 0; }
.donde strong { display: flex; align-items: center; gap: 6px; overflow-wrap: anywhere; }
.ic-alc { width: 18px; height: 18px; color: var(--cf); flex: none; }
.progreso { display: grid; gap: 6px; padding: 14px 16px; margin-bottom: 16px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); }
.progreso p { margin: 0; font-size: var(--t-s); }
.barra { height: 6px; border-radius: 99px; background: var(--linea-2); overflow: hidden; }
.barra span { display: block; height: 100%; background: var(--foco); transition: width .3s; }
.barra span.indef { width: 30% !important; animation: ir 1.4s ease-in-out infinite; }
@keyframes ir { from { transform: translateX(-100%); } to { transform: translateX(350%); } }
.herr { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; flex-wrap: wrap; }
.ord { display: flex; align-items: center; gap: 6px; font-size: var(--t-s); color: var(--tinta-2); }
.ord select { padding: 4px 6px; border: 1px solid var(--linea); border-radius: var(--r-s); background: #fff; }
.grupos { list-style: none; margin: 0; padding: 0; display: grid; gap: 6px; }
.grupo { background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); overflow: hidden; }
.grupo.abierto { border-color: var(--tinta-3); }
.cab-grupo { display: flex; align-items: center; gap: 10px; width: 100%; padding: 9px 12px; border: 0; background: none; text-align: left; font-size: var(--t-s); }
.cab-grupo:hover { background: #f8fafb; }
.chev { width: 13px; height: 13px; color: var(--tinta-3); transition: transform .12s; flex: none; }
.chev.girado { transform: rotate(90deg); }
.nom { flex: 1; min-width: 0; font-weight: 500; overflow-wrap: anywhere; }
.dato { color: var(--tinta-2); white-space: nowrap; font-size: var(--t-xs); }
.dato.fuerte { color: var(--rojo); font-weight: 600; }
.ic { width: 16px; height: 16px; flex: none; }
.ic.arch { display: inline-grid; place-items: center; color: var(--c); }
.ic.arch svg { width: 15px; height: 15px; }
.grupo .tabla-caja { border: 0; border-top: 1px solid var(--linea-2); border-radius: 0; }
.ubic { display: flex; align-items: center; gap: 8px; }
.enlace-carpeta { display: flex; align-items: center; gap: 6px; padding: 4px 0; border: 0; background: none; text-align: left; font-size: var(--t-s); color: var(--tinta); overflow-wrap: anywhere; }
.enlace-carpeta:hover { text-decoration: underline; }
.ic-c { width: 15px; height: 15px; color: var(--cf); flex: none; }
.etq { font-size: 11px; padding: 0 6px; border-radius: 99px; background: var(--verde-claro); color: var(--verde); white-space: nowrap; }
.fecha { white-space: nowrap; color: var(--tinta-2); }
</style>
