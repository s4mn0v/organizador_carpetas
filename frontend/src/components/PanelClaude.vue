<script setup>
// Panel lateral de Claude: chat con Claude Code (suscripción del usuario) usando las herramientas de la app.
// 1) el usuario elige QUÉ quiere hacer (herramienta) → 2) escribe → Claude solo recibe esa herramienta.
import {computed, nextTick, onBeforeUnmount, onMounted, ref} from 'vue'
import Icono from './Icono.vue'
import {marked} from 'marked'
import DOMPurify from 'dompurify'
import {
  AbrirConversacion, CancelarClaude, ConectarClaude, ConversacionClaude, DesconectarClaude, EliminarConversacion, EnviarAClaude,
  EstadoClaude, EstadosIndice, EstiloClaude, GuardarConversacion, GuardarEstiloClaude, HerramientasClaude, LimpiarLogClaude,
  ListarConversaciones, LogClaude, MedirIndexacion, NuevaConversacionClaude, UsoConversacion,
} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {mensajeError} from '../util'

const props = defineProps({
  contexto: {type: String, default: ''},        // ruta virtual seleccionada ("" = Inicio)
  contextoVisible: {type: String, default: ''}, // la misma ruta como se ve en Windows
})
const emit = defineEmits(['cerrar', 'desconectado'])

const conectado = ref(false)
const herramientas = ref([])
const elegida = ref(null)           // null = aún no elige; '' = Automático (todas)
const texto = ref('')
const usarContexto = ref(true)
const ocupado = ref(false)
const mensajes = ref([])            // {tipo: 'yo'|'claude'|'herr'|'res'|'error'|'info', ...}
const lista = ref(null)
const caja = ref(null)
let quitarEventos = []
let huboTexto = false
const indices = ref({})             // raíz -> EstadoIndice
const MAX_CONV = 5
const convId = ref('')              // conversación actual del historial ('' = ninguna)
const historial = ref([])           // ResumenConversacion[]
const hoja = ref(null)              // hoja superior derecha: 'historial' | 'estilo' | null
const aviso = ref('')               // mensaje en la hoja (p. ej. límite de conversaciones)
const borrando = ref('')            // id a confirmar para eliminar
const estilo = ref('')
const estiloGuardado = ref('')
let tIndice = null

/* ---------- Consumo y registro (popup) ---------- */
const verConsumo = ref(false)
const usoConv = ref(null)           // acumulado de la conversación actual
const usoUltimo = ref(null)         // última respuesta
const log = ref([])                 // líneas del registro
const term = ref(null)
const copiado = ref(false)
const fmt = (n) => (n || 0).toLocaleString('es-CO')
const total = (u) => u ? u.entrada + u.salida + u.cacheLectura + u.cacheCreacion : 0
async function cargarUso() { usoConv.value = convId.value ? await UsoConversacion(convId.value) : null }
async function abrirConsumo() {
  verConsumo.value = true
  log.value = (await LogClaude()) || []
  await cargarUso()
  bajarTerm()
}
function bajarTerm() { nextTick(() => term.value && (term.value.scrollTop = term.value.scrollHeight)) }
async function copiarLog() {
  try { await navigator.clipboard.writeText(log.value.join('\n')); copiado.value = true; setTimeout(() => (copiado.value = false), 1500) } catch { /* sin portapapeles */ }
}
async function limpiarLog() { await LimpiarLogClaude(); log.value = [] }
async function medir() { for (const id of Object.keys(indices.value)) await MedirIndexacion(id) }

marked.setOptions({gfm: true, breaks: true})
// Markdown de Claude -> HTML seguro (sin scripts ni atributos peligrosos).
const html = (t) => DOMPurify.sanitize(marked.parse(t || ''))

const detalleIndice = computed(() => {
  const l = Object.values(indices.value)
  const n = (k) => l.reduce((t, i) => t + (i[k] || 0), 0).toLocaleString('es-CO')
  const vivo = l.length && l.every((i) => i.vigilando)
  return `Estructura pregrabada: ${n('carpetas')} carpetas y ${n('archivos')} archivos. Claude la consulta sin recorrer el disco.` +
    (vivo ? ' Windows avisa cada cambio en Z: (también de otras personas): siempre al día.' : ' Sin aviso de cambios en vivo: la búsqueda y los tamaños leen el disco.')
})
// Sin índice completo no se envía nada a Claude (evita gastar tokens recorriendo el disco a ciegas).
const indiceListo = computed(() => {
  const l = Object.values(indices.value)
  return l.length > 0 && l.every((i) => i.listo && !i.dudoso)
})
const leidos = computed(() => Object.values(indices.value).reduce((n, i) => n + (i.leidos || 0), 0).toLocaleString('es-CO'))
const textoIndice = computed(() => {
  const l = Object.values(indices.value)
  if (!l.length) return ''
  if (l.some((i) => i.listo && i.dudoso)) return 're-escaneando…'
  if (!indiceListo.value) return 'creando índice…'
  return l.every((i) => i.vigilando) ? 'índice ✓ en vivo' : 'índice ✓ (sin vigilancia)'
})

const grupos = computed(() => [
  {tipo: 'lectura', titulo: 'Consultar', items: herramientas.value.filter((h) => h.tipo === 'lectura')},
  {tipo: 'escritura', titulo: 'Organizar', items: herramientas.value.filter((h) => h.tipo === 'escritura')},
  {tipo: 'peligro', titulo: 'Con confirmación', items: herramientas.value.filter((h) => h.tipo === 'peligro')},
])
const herrElegida = computed(() => herramientas.value.find((h) => h.nombre === elegida.value) || null)
const pista = computed(() => elegida.value === '' ? 'Escriba lo que necesita; Claude decide qué herramientas usar' : herrElegida.value?.pista || '')
const etiquetaDe = (n) => herramientas.value.find((h) => h.nombre === n)?.etiqueta || n

function agregar(m) {
  mensajes.value.push({id: Math.random(), ...m})
  nextTick(() => lista.value && (lista.value.scrollTop = lista.value.scrollHeight))
}

function alEvento(e) {
  switch (e.tipo) {
    case 'texto': huboTexto = true; agregar({tipo: 'claude', texto: e.texto}); break
    case 'herramienta': agregar({tipo: 'herr', idHerr: e.id, nombre: e.herramienta, entrada: resumenEntrada(e.entrada)}); break
    case 'resultado': {
      // Se une por id; si no viene, a la última pendiente.
      const pend = mensajes.value.filter((m) => m.tipo === 'herr' && m.resultado === undefined)
      const m = pend.find((x) => e.id && x.idHerr === e.id) || pend[pend.length - 1]
      if (m) { m.resultado = e.texto || '(sin texto)'; m.error = e.esError }
      break
    }
    case 'fin':
      if (!huboTexto && e.texto) agregar({tipo: 'claude', texto: e.texto})
      if (e.uso) usoUltimo.value = e.uso
      cerrarPendientes()
      guardar().then(cargarUso)
      break
    case 'error':
      agregar({tipo: 'error', texto: e.texto || 'Error de Claude'})
      if (e.uso) usoUltimo.value = e.uso
      cerrarPendientes()
      guardar().then(cargarUso)
      break
  }
}

// Al terminar, ninguna herramienta queda "cargando" (p. ej. pasos internos de Claude Code sin resultado visible).
function cerrarPendientes() {
  for (const m of mensajes.value) if (m.tipo === 'herr' && m.resultado === undefined) m.resultado = '(terminado)'
  ocupado.value = false
}

// {"ruta":"r1/Informes","nombre":"2026"} -> 'r1/Informes · 2026'
function resumenEntrada(x) {
  if (!x || typeof x !== 'object') return ''
  return Object.values(x).map((v) => typeof v === 'object' ? JSON.stringify(v) : String(v)).join(' · ')
}

async function enviar() {
  const t = texto.value.trim()
  if (!t || ocupado.value || elegida.value === null || !indiceListo.value) return
  if (!convId.value) {
    try {
      convId.value = await NuevaConversacionClaude()
    } catch (e) {
      await abrirHoja('historial', mensajeError(e))
      return
    }
  }
  const ctx = usarContexto.value && props.contexto ? `${props.contexto} (${props.contextoVisible})` : ''
  agregar({tipo: 'yo', texto: t, herr: elegida.value ? etiquetaDe(elegida.value) : 'Automático'})
  texto.value = ''
  ocupado.value = true
  huboTexto = false
  try {
    await EnviarAClaude(t, elegida.value, ctx)
  } catch (e) {
    agregar({tipo: 'error', texto: mensajeError(e)})
    ocupado.value = false
  }
}

function elegir(n) {
  elegida.value = n
  nextTick(() => caja.value?.focus())
}

/* ---------- Historial (máx. 5) ---------- */
async function cargarHistorial() { historial.value = (await ListarConversaciones()) || [] }

async function guardar() {
  if (!convId.value) return
  const titulo = mensajes.value.find((m) => m.tipo === 'yo')?.texto || ''
  try {
    await GuardarConversacion(convId.value, titulo, JSON.stringify(mensajes.value))
    await cargarHistorial()
  } catch { /* si falla, el chat sigue */ }
}

async function nueva() {
  try {
    convId.value = await NuevaConversacionClaude()
    mensajes.value = []
    elegida.value = null
    hoja.value = null
    usoConv.value = null
    usoUltimo.value = null
    await cargarHistorial()
  } catch (e) {
    await abrirHoja('historial', mensajeError(e))
  }
}

async function abrir(id) {
  try {
    const c = await AbrirConversacion(id)
    convId.value = c.id
    mensajes.value = JSON.parse(c.mensajes || '[]')
    usoConv.value = c.uso || null
    usoUltimo.value = null
    elegida.value = null
    hoja.value = null
    await cargarHistorial()
    nextTick(() => lista.value && (lista.value.scrollTop = lista.value.scrollHeight))
  } catch (e) { aviso.value = mensajeError(e) }
}

async function eliminar(id) {
  try {
    await EliminarConversacion(id)
    log.value = (await LogClaude()) || []
    if (id === convId.value) { convId.value = ''; mensajes.value = []; elegida.value = null }
    borrando.value = ''
    aviso.value = ''
    await cargarHistorial()
  } catch (e) { aviso.value = mensajeError(e) }
}

async function abrirHoja(h, msg = '') {
  aviso.value = msg
  if (hoja.value === h && !msg) { hoja.value = null; return }
  hoja.value = h
  borrando.value = ''
  if (h === 'historial') await cargarHistorial()
  if (h === 'estilo') { estilo.value = await EstiloClaude(); estiloGuardado.value = estilo.value }
}
function cerrarHoja() { hoja.value = null; aviso.value = '' }

/* ---------- Estilo de respuesta ("skill") ---------- */
async function guardarEstilo(texto) {
  try {
    estilo.value = await GuardarEstiloClaude(texto)
    estiloGuardado.value = estilo.value
    aviso.value = 'Guardado: se aplica desde el próximo mensaje.'
  } catch (e) { aviso.value = mensajeError(e) }
}

function fechaCorta(iso) {
  const d = new Date(iso)
  const min = Math.round((Date.now() - d) / 60000)
  if (min < 1) return 'ahora'
  if (min < 60) return `hace ${min} min`
  if (min < 1440) return `hace ${Math.round(min / 60)} h`
  return d.toLocaleDateString('es-CO', {day: 'numeric', month: 'short'})
}

async function desconectar() {
  if (ocupado.value) await CancelarClaude()
  await DesconectarClaude()
  conectado.value = false
  emit('desconectado')
}

function teclas(e) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); enviar() }
}

onMounted(async () => {
  quitarEventos = [
    EventsOn('claude:evento', alEvento),
    EventsOn('indice:estado', (i) => { indices.value = {...indices.value, [i.raiz]: i} }),
    EventsOn('claude:cambio', () => refrescarIndices()),
    EventsOn('claude:log', (l) => {
      log.value.push(l)
      if (log.value.length > 1000) log.value.splice(0, log.value.length - 1000)
      if (verConsumo.value) bajarTerm()
    }),
  ]
  try {
    await refrescarIndices()
    herramientas.value = await HerramientasClaude()
    const st = (await EstadoClaude()).activo ? await EstadoClaude() : await ConectarClaude()
    conectado.value = st.activo
    const conv = await ConversacionClaude()
    ocupado.value = conv.ocupado
    await cargarHistorial()
    if (conv.conv) await abrir(conv.conv) // vuelve a la conversación en curso
  } catch (e) {
    agregar({tipo: 'error', texto: mensajeError(e)})
  }
  // Mientras el índice no esté listo (o se abra otra raíz), se revisa cada 2 s.
  tIndice = setInterval(() => { if (!indiceListo.value) refrescarIndices() }, 2000)
})
async function refrescarIndices() {
  const l = (await EstadosIndice()) || []
  indices.value = Object.fromEntries(l.map((i) => [i.raiz, i]))
}
onBeforeUnmount(() => { quitarEventos.forEach((q) => q && q()); clearInterval(tIndice) })

defineExpose({conectar: async () => { conectado.value = (await ConectarClaude()).activo }})
</script>

<template>
  <aside class="panel-claude" aria-label="Claude">
    <header class="pc-cab">
      <div class="pc-titulo">
        <span class="pc-punto" :class="{on: conectado}" :title="conectado ? 'Conectado' : 'Desconectado'"></span>
        <strong>Claude</strong>
        <small>{{ conectado ? 'conectado' : 'desconectado' }}</small>
        <small v-if="textoIndice" class="pc-indice" :title="detalleIndice">· {{ textoIndice }}</small>
      </div>
      <div class="pc-acc">
        <button class="pc-btn" :class="{on: verConsumo}" title="Consumo de tokens y registro" @click="abrirConsumo"><Icono nombre="terminal"/></button>
        <button class="pc-btn" :class="{on: hoja === 'historial'}" :title="`Historial (${historial.length}/${MAX_CONV})`" @click="abrirHoja('historial')"><Icono nombre="reloj"/></button>
        <button class="pc-btn" :class="{on: hoja === 'estilo'}" title="Skill: estilo de respuesta" @click="abrirHoja('estilo')"><Icono nombre="editar"/></button>
        <button class="pc-btn" title="Nueva conversación" :disabled="ocupado" @click="nueva"><Icono nombre="mas"/></button>
        <button class="pc-btn" title="Desconectar Claude (apaga el servidor de herramientas)" @click="desconectar"><Icono nombre="quitar"/></button>
        <button class="pc-btn" title="Cerrar panel (sigue conectado)" @click="emit('cerrar')"><Icono nombre="cerrar"/></button>
      </div>
    </header>

    <!-- Popup: consumo de tokens + registro tipo terminal -->
    <div v-if="verConsumo" class="pc-modal-fondo" @click.self="verConsumo = false" @keydown.esc="verConsumo = false">
      <div class="pc-modal" role="dialog" aria-label="Consumo de tokens y registro">
        <div class="pc-modal-cab">
          <strong>Consumo de tokens y registro</strong>
          <button class="pc-btn" title="Cerrar" @click="verConsumo = false"><Icono nombre="cerrar"/></button>
        </div>
        <div class="pc-uso">
          <div class="pc-uso-caja">
            <small>Conversación actual</small>
            <template v-if="usoConv && usoConv.mensajes">
              <b class="pc-uso-total">{{ fmt(total(usoConv)) }} <span>tokens</span></b>
              <dl>
                <dt>Entrada</dt><dd>{{ fmt(usoConv.entrada) }}</dd>
                <dt>Salida</dt><dd>{{ fmt(usoConv.salida) }}</dd>
                <dt>Caché leída</dt><dd>{{ fmt(usoConv.cacheLectura) }}</dd>
                <dt>Caché creada</dt><dd>{{ fmt(usoConv.cacheCreacion) }}</dd>
                <dt>Respuestas</dt><dd>{{ usoConv.mensajes }}</dd>
                <dt>Equivalente API</dt><dd>US$ {{ usoConv.costoUSD.toFixed(4) }}</dd>
              </dl>
            </template>
            <p v-else class="pc-nota">Sin consumo todavía.</p>
          </div>
          <div class="pc-uso-caja">
            <small>Última respuesta</small>
            <template v-if="usoUltimo">
              <b class="pc-uso-total">{{ fmt(total(usoUltimo)) }} <span>tokens</span></b>
              <dl>
                <dt>Entrada / salida</dt><dd>{{ fmt(usoUltimo.entrada) }} / {{ fmt(usoUltimo.salida) }}</dd>
                <dt>Contexto enviado</dt><dd>{{ fmt(usoUltimo.entrada + usoUltimo.cacheLectura + usoUltimo.cacheCreacion) }}</dd>
                <dt>Turnos</dt><dd>{{ usoUltimo.turnos }}</dd>
                <dt>Duración</dt><dd>{{ (usoUltimo.duracionMs / 1000).toFixed(1) }} s</dd>
                <dt>Equivalente API</dt><dd>US$ {{ usoUltimo.costoUSD.toFixed(4) }}</dd>
              </dl>
            </template>
            <p v-else class="pc-nota">Aparece al terminar el próximo mensaje.</p>
          </div>
        </div>
        <p class="pc-nota">Con su plan Pro/Max no se cobra por token: el valor en US$ es solo una referencia en precios de API. La caché leída es contexto reutilizado y gasta mucho menos de su límite de uso.</p>
        <div class="pc-term-cab">
          <small>Registro ({{ log.length }} líneas)</small>
          <span>
            <button class="pc-mini" title="Compara el recorrido anterior con el nuevo (puede tardar en Z:)" @click="medir">Medir indexación</button>
            <button class="pc-mini" @click="copiarLog">{{ copiado ? 'Copiado ✓' : 'Copiar' }}</button>
            <button class="pc-mini" @click="limpiarLog">Limpiar</button>
          </span>
        </div>
        <pre ref="term" class="pc-term">{{ log.length ? log.join('\n') : 'Sin actividad todavía.' }}</pre>
      </div>
    </div>

    <!-- Hoja superior derecha: historial o estilo -->
    <section v-if="hoja" class="pc-hoja">
      <div class="pc-hoja-cab">
        <strong v-if="hoja === 'historial'">Historial <small>{{ historial.length }}/{{ MAX_CONV }}</small></strong>
        <strong v-else>Skill: estilo de respuesta</strong>
        <button class="pc-btn" title="Cerrar" @click="cerrarHoja"><Icono nombre="cerrar"/></button>
      </div>
      <p v-if="aviso" class="pc-aviso">{{ aviso }}</p>

      <template v-if="hoja === 'historial'">
        <p v-if="!historial.length" class="pc-nota">Aún no hay conversaciones.</p>
        <ul class="pc-convs">
          <li v-for="c in historial" :key="c.id" :class="{actual: c.id === convId}">
            <button class="pc-conv" :disabled="ocupado" @click="abrir(c.id)">
              <b>{{ c.titulo }}</b><small>{{ fechaCorta(c.actualizada) }}{{ c.id === convId ? ' · actual' : '' }}</small>
            </button>
            <template v-if="borrando === c.id">
              <button class="pc-mini rojo" @click="eliminar(c.id)">Eliminar</button>
              <button class="pc-mini" @click="borrando = ''">No</button>
            </template>
            <button v-else class="pc-btn" title="Eliminar (también borra la sesión guardada de Claude)" @click="borrando = c.id"><Icono nombre="borrar"/></button>
          </li>
        </ul>
        <p class="pc-nota">Máximo {{ MAX_CONV }}. Al eliminar una se borra todo lo que quedó en este PC (mensajes, sesión de Claude Code y su registro). En los servidores de Anthropic se rige por su configuración de privacidad en claude.ai.</p>
      </template>

      <template v-else>
        <p class="pc-nota">Se agrega a las instrucciones de Claude en cada mensaje. Úselo para controlar el largo y el formato de las respuestas.</p>
        <textarea v-model="estilo" class="pc-estilo" rows="6"></textarea>
        <div class="pc-hoja-acc">
          <button class="pc-mini" @click="guardarEstilo('')">Restablecer</button>
          <button class="pc-mini primario" :disabled="estilo.trim() === estiloGuardado.trim()" @click="guardarEstilo(estilo)">Guardar</button>
        </div>
      </template>
    </section>

    <div ref="lista" class="pc-lista">
      <p v-if="!mensajes.length" class="pc-vacio">
        Elija abajo <b>qué quiere hacer</b> y escriba su pedido.<br>
        Eliminar, depurar y editar fichas siempre piden su confirmación.
      </p>
      <template v-for="m in mensajes" :key="m.id">
        <div v-if="m.tipo === 'yo'" class="pc-yo"><small>{{ m.herr }}</small>{{ m.texto }}</div>
        <div v-else-if="m.tipo === 'claude'" class="pc-claude pc-md" v-html="html(m.texto)"></div>
        <details v-else-if="m.tipo === 'herr'" class="pc-herr" :class="{error: m.error, cargando: m.resultado === undefined}">
          <summary><Icono nombre="sync"/><b>{{ etiquetaDe(m.nombre) }}</b><span>{{ m.entrada }}</span></summary>
          <pre v-if="m.resultado !== undefined">{{ m.resultado }}</pre>
        </details>
        <div v-else-if="m.tipo === 'error'" class="pc-error"><Icono nombre="alerta"/>{{ m.texto }}</div>
      </template>
      <div v-if="ocupado" class="pc-pensando"><span></span><span></span><span></span>Claude está trabajando…</div>
    </div>

    <footer class="pc-pie">
      <!-- Paso 1: qué quiere hacer -->
      <div class="pc-herramientas" :class="{compacto: elegida !== null}">
        <template v-if="elegida === null">
          <p class="pc-paso">1 · ¿Qué quiere hacer?</p>
          <div v-for="g in grupos" :key="g.tipo" class="pc-grupo">
            <small>{{ g.titulo }}</small>
            <div class="pc-chips">
              <button v-for="h in g.items" :key="h.nombre" class="pc-chip" :class="h.tipo" @click="elegir(h.nombre)">{{ h.etiqueta }}</button>
            </div>
          </div>
          <button class="pc-chip auto" @click="elegir('')">Automático (Claude elige)</button>
        </template>
        <div v-else class="pc-elegida">
          <span class="pc-chip on" :class="herrElegida?.tipo || 'auto'">{{ herrElegida?.etiqueta || 'Automático' }}</span>
          <button class="pc-cambiar" :disabled="ocupado" @click="elegida = null">Cambiar</button>
        </div>
      </div>

      <!-- Paso 2: el pedido -->
      <label v-if="contexto" class="pc-ctx" :title="contextoVisible">
        <input type="checkbox" v-model="usarContexto"/> Sobre: <code>{{ contextoVisible }}</code>
      </label>
      <div class="pc-entrada">
        <textarea ref="caja" v-model="texto" rows="2" :disabled="elegida === null || !indiceListo"
                  :placeholder="!indiceListo ? `Actualizando índice… ${leidos} elementos leídos. Claude se habilita al terminar.` : elegida === null ? 'Primero elija qué quiere hacer' : pista"
                  @keydown="teclas"></textarea>
        <button v-if="ocupado" class="btn pc-enviar" title="Detener" @click="CancelarClaude()"><Icono nombre="cerrar"/></button>
        <button v-else class="btn pc-enviar" :disabled="!texto.trim() || elegida === null || !indiceListo" title="Enviar (Enter)" @click="enviar"><Icono nombre="subir"/></button>
      </div>
    </footer>
  </aside>
</template>

<style>
.panel-claude { grid-column: 3; grid-row: 1; display: flex; flex-direction: column; min-height: 0; height: 100vh; background: var(--superficie); border-left: 1px solid var(--linea); }
.pc-cab { display: flex; justify-content: space-between; align-items: center; padding: 12px 12px 10px 16px; border-bottom: 1px solid var(--linea-2); }
.pc-titulo { display: flex; align-items: center; gap: 6px; min-width: 0; overflow: hidden; }
.pc-titulo small { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pc-acc { flex: none; }
.pc-titulo small { color: var(--tinta-3); font-size: var(--t-xs); }
.pc-punto { width: 8px; height: 8px; border-radius: 50%; background: var(--tinta-3); }
.pc-punto.on { background: var(--verde); box-shadow: 0 0 0 3px var(--verde-claro); }
.pc-acc { display: flex; gap: 2px; }
.pc-btn { width: 30px; height: 30px; display: grid; place-items: center; border: 0; border-radius: var(--r-s); background: none; color: var(--tinta-2); }
.pc-btn:hover:not(:disabled) { background: var(--papel); color: var(--tinta); }
.pc-btn:disabled { opacity: .4; }
.pc-btn.on { background: var(--ambar-claro); color: var(--ambar-oscuro); }
.panel-claude { position: relative; }
.pc-hoja { position: absolute; top: 52px; right: 10px; z-index: 5; width: min(340px, calc(100% - 20px)); max-height: 70%; overflow-y: auto;
  padding: 10px 12px 12px; border: 1px solid var(--linea); border-radius: var(--r-m); background: var(--superficie); box-shadow: 0 10px 30px rgba(16, 49, 59, .18); }
.pc-hoja-cab { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.pc-hoja-cab small { color: var(--tinta-3); font-weight: 400; }
.pc-aviso { margin: 0 0 8px; padding: 6px 8px; border-radius: var(--r-s); background: var(--amarillo-claro); color: var(--ambar-oscuro); font-size: var(--t-xs); }
.pc-nota { margin: 6px 0; color: var(--tinta-3); font-size: 11px; }
.pc-convs { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
.pc-convs li { display: flex; align-items: center; gap: 4px; border-radius: var(--r-s); }
.pc-convs li.actual { background: var(--ambar-claro); }
.pc-conv { flex: 1; min-width: 0; display: grid; padding: 5px 8px; border: 0; background: none; text-align: left; border-radius: var(--r-s); }
.pc-conv:hover:not(:disabled) { background: var(--papel); }
.pc-conv b { font-size: var(--t-xs); font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pc-conv small { font-size: 11px; color: var(--tinta-3); }
.pc-mini { padding: 3px 9px; border: 1px solid var(--linea); border-radius: var(--r-s); background: var(--superficie); font-size: var(--t-xs); }
.pc-mini.rojo { border-color: var(--rojo); color: var(--rojo); }
.pc-mini.primario { border-color: var(--ambar); background: var(--ambar); color: #3d2a00; font-weight: 600; }
.pc-mini:disabled { opacity: .45; }
.pc-estilo { width: 100%; resize: vertical; padding: 7px 9px; border: 1px solid var(--linea); border-radius: var(--r-m); font-size: var(--t-xs); }
.pc-modal-fondo { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; background: rgba(16, 49, 59, .35); }
.pc-modal { width: min(760px, calc(100vw - 40px)); max-height: calc(100vh - 60px); display: flex; flex-direction: column; gap: 8px; padding: 14px 16px 16px;
  border-radius: var(--r-m); background: var(--superficie); box-shadow: 0 20px 60px rgba(0, 0, 0, .3); }
.pc-modal-cab { display: flex; justify-content: space-between; align-items: center; }
.pc-uso { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.pc-uso-caja { padding: 10px 12px; border: 1px solid var(--linea-2); border-radius: var(--r-m); background: #fafbfc; }
.pc-uso-caja > small { color: var(--tinta-3); font-size: 11px; text-transform: uppercase; letter-spacing: .04em; }
.pc-uso-total { display: block; margin: 2px 0 6px; font-size: var(--t-xl); font-weight: 600; }
.pc-uso-total span { font-size: var(--t-xs); font-weight: 400; color: var(--tinta-3); }
.pc-uso dl { display: grid; grid-template-columns: 1fr auto; gap: 2px 10px; margin: 0; font-size: var(--t-xs); }
.pc-uso dt { color: var(--tinta-2); }
.pc-uso dd { margin: 0; text-align: right; font-variant-numeric: tabular-nums; }
.pc-term-cab { display: flex; justify-content: space-between; align-items: center; }
.pc-term-cab small { color: var(--tinta-3); font-size: 11px; text-transform: uppercase; letter-spacing: .04em; }
.pc-term-cab span { display: flex; gap: 6px; }
.pc-term { flex: 1; min-height: 180px; max-height: 42vh; margin: 0; padding: 10px 12px; overflow: auto; border-radius: var(--r-m);
  background: #0d2730; color: #c9dbe0; font: 12px/1.55 Consolas, 'Cascadia Mono', monospace; white-space: pre-wrap; word-break: break-word; }
.pc-hoja-acc { display: flex; justify-content: flex-end; gap: 6px; margin-top: 6px; }
.pc-btn svg { width: 16px; height: 16px; }

.pc-lista { flex: 1; overflow-y: auto; padding: 14px 14px 6px; display: flex; flex-direction: column; gap: 8px; }
.pc-vacio { margin: auto 8px; color: var(--tinta-3); font-size: var(--t-s); text-align: center; }
.pc-yo { align-self: flex-end; max-width: 88%; padding: 8px 11px; border-radius: 10px 10px 2px 10px; background: var(--riel); color: #fff; white-space: pre-wrap; font-size: var(--t-s); }
.pc-yo small { display: block; color: var(--ambar); font-size: 11px; font-weight: 600; }
.pc-claude { max-width: 100%; padding: 2px 2px; font-size: var(--t-s); color: var(--tinta); overflow-wrap: anywhere; }
.pc-indice { color: var(--tinta-3); font-size: 11px; white-space: nowrap; cursor: help; }
/* Markdown de Claude */
.pc-md > :first-child { margin-top: 0; }
.pc-md > :last-child { margin-bottom: 0; }
.pc-md p { margin: 0 0 8px; }
.pc-md h1, .pc-md h2, .pc-md h3, .pc-md h4 { margin: 12px 0 6px; font-size: var(--t-m); font-weight: 600; line-height: 1.3; }
.pc-md h1 { font-size: var(--t-l); }
.pc-md ul, .pc-md ol { margin: 0 0 8px; padding-left: 20px; }
.pc-md li { margin: 2px 0; }
.pc-md li > p { margin: 0; }
.pc-md code { padding: 1px 4px; border-radius: 3px; background: var(--papel); font-size: 12px; font-family: Consolas, monospace; overflow-wrap: anywhere; }
.pc-md pre { margin: 0 0 8px; padding: 8px; border-radius: var(--r-s); background: var(--papel); overflow-x: auto; }
.pc-md pre code { padding: 0; background: none; }
.pc-md blockquote { margin: 0 0 8px; padding-left: 10px; border-left: 3px solid var(--linea); color: var(--tinta-2); }
.pc-md table { width: 100%; margin: 0 0 8px; border-collapse: collapse; font-size: 12px; display: block; overflow-x: auto; }
.pc-md th, .pc-md td { padding: 3px 6px; border: 1px solid var(--linea-2); text-align: left; }
.pc-md th { background: var(--papel); font-weight: 600; }
.pc-md hr { border: 0; border-top: 1px solid var(--linea-2); margin: 10px 0; }
.pc-md a { color: var(--foco); }
.pc-herr { font-size: var(--t-xs); color: var(--tinta-2); border: 1px solid var(--linea-2); border-radius: var(--r-s); background: var(--papel); }
.pc-herr summary { display: flex; align-items: center; gap: 6px; padding: 4px 8px; cursor: pointer; list-style: none; }
.pc-herr summary::-webkit-details-marker { display: none; }
.pc-herr summary svg { width: 12px; height: 12px; flex: none; }
.pc-herr.cargando summary svg { animation: pc-girar 1s linear infinite; }
.pc-herr summary span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--tinta-3); }
.pc-herr.error { border-color: var(--rojo); background: var(--rojo-claro); }
.pc-herr pre { margin: 0; padding: 6px 8px; max-height: 180px; overflow: auto; border-top: 1px solid var(--linea-2); white-space: pre-wrap; word-break: break-all; font-size: 11px; }
.pc-error { display: flex; gap: 6px; align-items: flex-start; padding: 8px 10px; border-radius: var(--r-s); background: var(--rojo-claro); color: var(--rojo); font-size: var(--t-s); white-space: pre-wrap; }
.pc-error svg { width: 16px; height: 16px; flex: none; margin-top: 2px; }
.pc-pensando { display: flex; align-items: center; gap: 4px; color: var(--tinta-3); font-size: var(--t-xs); }
.pc-pensando span { width: 5px; height: 5px; border-radius: 50%; background: var(--ambar); animation: pc-salto 1s infinite; }
.pc-pensando span:nth-child(2) { animation-delay: .15s; }
.pc-pensando span:nth-child(3) { animation-delay: .3s; margin-right: 6px; }

.pc-pie { padding: 10px 12px 12px; border-top: 1px solid var(--linea-2); display: grid; gap: 8px; }
.pc-paso { margin: 0 0 2px; font-size: var(--t-xs); font-weight: 600; color: var(--tinta-2); }
.pc-grupo { margin-bottom: 6px; }
.pc-grupo > small { display: block; margin-bottom: 3px; color: var(--tinta-3); font-size: 11px; text-transform: uppercase; letter-spacing: .04em; }
.pc-chips { display: flex; flex-wrap: wrap; gap: 4px; }
.pc-chip { padding: 3px 9px; border: 1px solid var(--linea); border-radius: 999px; background: var(--superficie); font-size: var(--t-xs); }
.pc-chip:hover { border-color: var(--tinta-3); }
.pc-chip.lectura { border-left: 3px solid var(--verde); }
.pc-chip.escritura { border-left: 3px solid var(--amarillo); }
.pc-chip.peligro { border-left: 3px solid var(--rojo); }
.pc-chip.auto { border-style: dashed; }
.pc-chip.on { background: var(--ambar-claro); font-weight: 600; cursor: default; }
.pc-elegida { display: flex; align-items: center; gap: 8px; }
.pc-cambiar { border: 0; background: none; color: var(--foco); font-size: var(--t-xs); text-decoration: underline; }
.pc-ctx { display: flex; align-items: center; gap: 6px; font-size: var(--t-xs); color: var(--tinta-2); min-width: 0; }
.pc-ctx code { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pc-entrada { display: flex; gap: 6px; align-items: flex-end; }
.pc-entrada textarea { flex: 1; resize: none; padding: 7px 9px; border: 1px solid var(--linea); border-radius: var(--r-m); font-size: var(--t-s); background: var(--superficie); }
.pc-entrada textarea:disabled { background: var(--papel); }
.pc-enviar { width: 36px; height: 36px; padding: 0; justify-content: center; background: var(--ambar); border-color: var(--ambar); color: #3d2a00; }
.pc-enviar:disabled { opacity: .45; }
@keyframes pc-girar { to { transform: rotate(360deg); } }
@keyframes pc-salto { 0%, 100% { opacity: .3; transform: translateY(0); } 50% { opacity: 1; transform: translateY(-3px); } }
</style>
