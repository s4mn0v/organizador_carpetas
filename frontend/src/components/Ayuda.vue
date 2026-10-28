<script setup>
import {nextTick, onMounted, ref} from 'vue'
import Icono from './Icono.vue'

const emit = defineEmits(['cerrar'])
const TEMAS = [
  {id: 'inicio', titulo: 'Cómo funciona'},
  {id: 'barra', titulo: 'La barra lateral'},
  {id: 'carpeta', titulo: 'Ver una carpeta'},
  {id: 'archivo', titulo: 'Ver un archivo'},
  {id: 'crud', titulo: 'Crear, renombrar, mover, eliminar'},
  {id: 'fichas', titulo: 'Fichas técnicas (Excel)'},
  {id: 'grafo', titulo: 'Vista gráfica'},
  {id: 'buscar', titulo: 'Buscar'},
  {id: 'duplicados', titulo: 'Archivos duplicados'},
  {id: 'vencidos', titulo: 'Retención y vencidos'},
  {id: 'problemas', titulo: 'Si algo sale mal'},
]
const tema = ref('inicio')
const cuerpo = ref(null)
const caja = ref(null)

async function ir(id) {
  tema.value = id
  await nextTick()
  if (cuerpo.value) cuerpo.value.scrollTop = 0
}
onMounted(() => caja.value?.focus())
</script>

<template>
  <div class="velo" @click.self="emit('cerrar')">
    <div ref="caja" class="guia" role="dialog" aria-modal="true" aria-label="Cómo usar la aplicación" tabindex="-1"
         @keydown.esc="emit('cerrar')">
      <header>
        <div>
          <p class="sobre">Guía rápida</p>
          <h2>Cómo usar el organizador de carpetas</h2>
        </div>
        <button class="cerrar" aria-label="Cerrar" @click="emit('cerrar')"><Icono nombre="cerrar"/></button>
      </header>

      <div class="cuerpo">
        <nav aria-label="Temas">
          <button v-for="(t, i) in TEMAS" :key="t.id" :class="{activo: tema === t.id}" @click="ir(t.id)">
            <span class="n">{{ i + 1 }}</span>{{ t.titulo }}
          </button>
        </nav>

        <article ref="cuerpo">
          <!-- 1 -->
          <template v-if="tema === 'inicio'">
            <h3>Cómo funciona</h3>
            <p>La aplicación sirve para ordenar carpetas (locales o de red) y documentar sus subcarpetas. Trabaja con tres cosas:</p>
            <div class="tarjetas">
              <div><b>Las carpetas raíz</b><span>Las que usted elige, por ejemplo <code>Z:\MANTENIMIENTO\PRIVADA</code> y <code>Z:\RRHH\PRIVADA</code>. Puede abrir varias a la vez; cada una tiene su color.</span></div>
              <div><b>El Excel de fichas</b><span><code>Fichas tecnicas carpetas.xlsx</code>, uno en cada raíz: una ficha técnica por carpeta principal, con la explicación de sus subcarpetas.</span></div>
              <div><b>La papelera de la app</b><span><code>_papelera_organizador</code>, en cada raíz: lo que elimine o depure va ahí, no se borra.</span></div>
            </div>
            <h4>La idea en una frase</h4>
            <p class="destacado">Agregue las carpetas raíz → navegue → organícelas (crear, renombrar, mover) → documente cada carpeta principal con su <b>ficha técnica</b> y su <b>tiempo de retención</b> → limpie duplicados y vencidos.</p>
            <h4>Reglas que la app siempre cumple</h4>
            <ul>
              <li>Los cambios se hacen <b>directo en la carpeta</b>: lo que vea aquí es lo que hay en el disco o en Z.</li>
              <li><b>Nada se borra de golpe.</b> Eliminar mueve a <code>_papelera_organizador</code> (en Z la papelera de Windows no existe).</li>
              <li>Si renombra o mueve una carpeta, su ficha y las explicaciones de sus subcarpetas se actualizan solas en el Excel.</li>
              <li>Cada cambio queda en la hoja <b>Historial</b> del Excel.</li>
            </ul>
          </template>

          <!-- 2 -->
          <template v-else-if="tema === 'barra'">
            <h3>La barra lateral</h3>
            <ul>
              <li><b>Agregar carpeta raíz</b>: abre otra carpeta. Todas se ven juntas bajo <b>Inicio</b>, cada una con su color (por ejemplo amarillo para MANTENIMIENTO, azul para RRHH).</li>
              <li><b>Carpetas / Gráfica / Duplicados / Vencidos</b>: cambia de vista.</li>
              <li><b>Buscar</b>: busca por palabras en todas las carpetas raíz (<kbd>Ctrl</kbd> + <kbd>F</kbd>). Ver “Buscar”.</li>
              <li><b>Mostrar archivos</b>: muestra u oculta los archivos en el árbol.</li>
            </ul>
            <h4>El árbol</h4>
            <table>
              <tbody>
                <tr><td><span class="mas">›</span></td><td>Abre o cierra la carpeta sin seleccionarla.</td></tr>
                <tr><td>Clic en el nombre</td><td>Selecciona la carpeta o el archivo y la abre a la derecha.</td></tr>
                <tr><td><i class="p lleno"></i></td><td>La carpeta tiene ficha técnica en el Excel.</td></tr>
                <tr><td>Clic derecho</td><td>Menú con todas las acciones: nueva carpeta, renombrar, mover, eliminar, ver en el gráfico, duplicados y vencidos. En una raíz: cambiar color o quitarla de la lista.</td></tr>
                <tr><td>Arrastrar</td><td>Arrastre una carpeta o archivo y suéltelo sobre otra carpeta para moverlo (pide confirmar). Si se queda encima de una carpeta cerrada, se abre.</td></tr>
              </tbody>
            </table>
            <p class="nota">Cada carpeta se lee al abrirla, para que funcione rápido aunque la raíz esté en red. Si una carpeta tiene muchos archivos, aparece “Ver más”.</p>
            <h4>Botones de abajo</h4>
            <ul>
              <li><b>Nueva carpeta</b>: crea una carpeta dentro de la seleccionada.</li>
              <li><b>Recargar</b> (<kbd>F5</kbd>): vuelve a leer todo. Úselo si cambió algo desde el Explorador.</li>
              <li><b>Abrir Excel</b>, <b>Papelera</b> y <b>Ayuda</b> (<kbd>F1</kbd>).</li>
            </ul>
          </template>

          <!-- 3 -->
          <template v-else-if="tema === 'carpeta'">
            <h3>Ver una carpeta</h3>
            <ol>
              <li>Arriba, la <b>ruta</b>: cada parte es un botón para subir de nivel.</li>
              <li><b>Cifras</b>: subcarpetas, <b>archivos y tamaño de la carpeta completa</b> (con todas sus subcarpetas; en red puede tardar unos segundos en aparecer), fecha, si tiene ficha y su retención.</li>
              <li><b>Ficha técnica</b>: la información de la carpeta guardada en el Excel.</li>
              <li><b>Contenido</b>: las subcarpetas y archivos. Haga clic en un encabezado para ordenar; doble clic en una fila para entrar o abrir.</li>
            </ol>
            <p class="nota">En una carpeta raíz también verá si falta su Excel de fichas (botón para crearlo) y si hay fichas que apuntan a carpetas que ya no existen. En <b>Inicio</b> están todas las raíces abiertas con el total de archivos y tamaño.</p>
          </template>

          <!-- 4 -->
          <template v-else-if="tema === 'archivo'">
            <h3>Ver un archivo</h3>
            <p>Muestra el tipo, el tamaño, las fechas y la ubicación completa.</p>
            <ul>
              <li><b>Abrir</b>: lo abre con su programa (Excel, PDF, Power BI…).</li>
              <li><b>Mostrar en el Explorador</b>: abre la carpeta con el archivo seleccionado.</li>
              <li><b>Renombrar</b>, <b>Mover</b>, <b>Eliminar</b>.</li>
            </ul>
            <p class="nota">Al renombrar, revise que la extensión (<code>.pdf</code>, <code>.xlsx</code>…) siga igual.</p>
          </template>

          <!-- 5 -->
          <template v-else-if="tema === 'crud'">
            <h3>Crear, renombrar, mover y eliminar</h3>
            <table>
              <tbody>
                <tr><td><b>Nueva carpeta</b></td><td>Botón amarillo o “Nueva subcarpeta” en una carpeta.</td></tr>
                <tr><td><b>Agregar archivos</b></td><td>Copia archivos de cualquier parte a la carpeta. Si ya existe uno igual, el nuevo queda como “nombre (2)”. El original no se toca.</td></tr>
                <tr><td><b>Renombrar</b> (<kbd>F2</kbd>)</td><td>Cambia el nombre. En carpetas, sus fichas se actualizan.</td></tr>
                <tr><td><b>Mover</b></td><td>Elija la carpeta destino en el árbol que aparece, o arrástrela en la barra lateral. Una carpeta se mueve con todo su contenido y sus fichas. Se puede mover a otra raíz, salvo carpetas con fichas o explicaciones en el Excel de su raíz.</td></tr>
                <tr><td><b>Eliminar</b></td><td>Pasa a <code>_papelera_organizador\fecha\…</code>. Si la carpeta tiene contenido o fichas, pide escribir el nombre para confirmar. Sus fichas se borran del Excel (quedan en Historial).</td></tr>
              </tbody>
            </table>
            <p class="nota">Para recuperar algo eliminado: botón <b>Papelera</b> → busque la fecha → muévalo de vuelta con el Explorador.</p>
          </template>

          <!-- 6 -->
          <template v-else-if="tema === 'fichas'">
            <h3>Fichas técnicas (Excel)</h3>
            <p>Las fichas van en las <b>carpetas principales</b>: las que están directo en la raíz (por ejemplo <code>1. Ejecución</code>). Las subcarpetas no llevan ficha propia: se <b>explican dentro</b> de la ficha de su carpeta principal.</p>
            <ol class="pasos">
              <li>Si no existe el Excel, vaya a la raíz y pulse <b>Crear Excel de fichas</b>.</li>
              <li>Abra una carpeta principal → <b>Crear ficha técnica</b>.</li>
              <li>Llene la sección <b>General</b>.</li>
              <li>En <b>Retención de archivos</b> indique cuánto tiempo deben quedarse los archivos (ver “Retención y vencidos”).</li>
              <li>En <b>Subcarpetas</b> escriba para qué sirve cada una (todas las que tiene, en todos los niveles) y, si necesita otra regla, su propia retención. Las que deje vacías no se guardan.</li>
              <li><b>Guardar en Excel</b>.</li>
            </ol>
            <h4>Qué se ve en cada carpeta</h4>
            <ul>
              <li><b>Raíz</b>: cuáles carpetas principales tienen ficha y cuáles no.</li>
              <li><b>Carpeta principal</b>: su ficha completa, con la explicación de sus subcarpetas.</li>
              <li><b>Subcarpeta</b>: su explicación y un botón para ir a la ficha de su carpeta principal.</li>
            </ul>
            <h4>Hojas del Excel</h4>
            <ul>
              <li><b>Carpetas</b>: una fila por ficha (carpetas principales).</li>
              <li><b>Subcarpetas</b>: una fila por subcarpeta explicada (ruta, descripción y retención propia).</li>
              <li><b>Campos</b>: qué campos tiene la ficha y de qué tipo. Puede agregar campos aquí (y la columna con el mismo nombre en la hoja Carpetas).</li>
              <li><b>Historial</b>: cada cambio hecho desde la app.</li>
            </ul>
            <p class="nota">Si renombra o mueve una subcarpeta, su explicación la sigue. Si el Excel está abierto en Excel, la app no puede escribir y lo avisa.</p>
          </template>

          <!-- 7 -->
          <template v-else-if="tema === 'grafo'">
            <h3>Vista gráfica</h3>
            <p>Muestra la carpeta como una red: cada punto es una carpeta o un archivo, unido a la carpeta que lo contiene.</p>
            <table>
              <tbody>
                <tr><td><i class="p" style="background:#f2b33d;box-shadow:none"></i></td><td>Carpeta. Más grande = más contenido.</td></tr>
                <tr><td><i class="p" style="background:#f2b33d;box-shadow:0 0 0 2px #fff, 0 0 0 3px #10313b"></i></td><td>Carpeta con ficha técnica (borde blanco).</td></tr>
                <tr><td><i class="p" style="background:#2e8b57;box-shadow:none"></i> <i class="p" style="background:#c8402b;box-shadow:none"></i> …</td><td>Archivos, con color por tipo (ver leyenda abajo a la izquierda).</td></tr>
              </tbody>
            </table>
            <h4>Cómo moverse</h4>
            <ul>
              <li><b>Rueda</b>: acercar / alejar. <b>Arrastrar el fondo</b>: mover la vista. <b>Arrastrar un punto</b>: reacomodarlo.</li>
              <li><b>Clic</b>: lo selecciona (y lo abre en la barra lateral). <b>Doble clic</b> en carpeta: dibuja desde ella; en archivo: lo abre.</li>
              <li><b>Niveles</b>: cuántos niveles de subcarpetas dibujar.</li>
              <li><b>Resaltar y buscar</b>: ilumina lo que se parece al texto (misma regla que el buscador) y abre una lista; elija uno (o <kbd>↓</kbd> + <kbd>Enter</kbd>) y el gráfico hace zoom hasta él. Con “Buscar también en las subcarpetas no dibujadas” encuentra archivos más profundos: al elegirlos, se dibuja su carpeta y se enfocan.</li>
              <li>Los colores de las carpetas son los de su raíz.</li>
            </ul>
            <p class="nota">Para no saturar, se dibujan como máximo 2.500 elementos. Si la carpeta es muy grande, baje los niveles o dibuje desde una subcarpeta.</p>
          </template>

          <template v-else-if="tema === 'buscar'">
            <h3>Buscar</h3>
            <p>El buscador de la barra lateral, el de la vista gráfica y los filtros de Duplicados y Vencidos usan la misma regla:</p>
            <ul>
              <li>Muestra lo que tenga <b>al menos una</b> de las palabras escritas.</li>
              <li>El <b>orden no importa</b>: “informe mantenimiento” encuentra “Mantenimiento – Informe 2024”.</li>
              <li>No importan tildes ni mayúsculas, y encuentra palabras <b>parecidas</b>: que empiezan igual (“mant” → “mantenimiento”), que están dentro de otra, o con una o dos letras cambiadas (“mantenimeinto”).</li>
              <li>Primero aparecen los que tienen <b>todas</b> las palabras, luego los más parecidos.</li>
            </ul>
            <p class="nota">Busca en todas las carpetas raíz abiertas. Se muestran los 300 resultados más parecidos.</p>
          </template>

          <template v-else-if="tema === 'duplicados'">
            <h3>Archivos duplicados</h3>
            <p>Encuentra archivos con el <b>mismo nombre y el mismo contenido</b> en la carpeta elegida y en todas sus subcarpetas (o en todas las raíces, desde Inicio).</p>
            <ol class="pasos">
              <li>Elija la carpeta en la barra lateral (o clic derecho → <b>Buscar duplicados aquí</b>).</li>
              <li>Pestaña <b>Duplicados</b> → <b>Buscar duplicados</b>.</li>
              <li>Abra cada grupo: verá cada copia con su ubicación, tamaño, fecha de creación y de modificación. La más antigua se marca.</li>
              <li>Elimine las copias que sobran (van a la papelera de la app).</li>
            </ol>
            <p class="nota">Primero se comparan nombre y tamaño; solo esos archivos se leen completos para confirmar que son idénticos. En red, la primera búsqueda grande puede tardar varios minutos; se puede detener.</p>
          </template>

          <template v-else-if="tema === 'vencidos'">
            <h3>Retención y vencidos</h3>
            <p>La <b>retención</b> es cuánto tiempo deben quedarse los archivos de una carpeta. Un archivo está <b>vencido</b> cuando su fecha de modificación es más antigua que ese tiempo.</p>
            <h4>Dónde se define</h4>
            <ul>
              <li>En la ficha de la carpeta principal: campo <b>Retención de archivos</b> (por ejemplo “2 años”). Aplica a ella y a todas sus subcarpetas.</li>
              <li>En la sección <b>Subcarpetas</b> de la ficha, una subcarpeta puede tener su propia retención (por ejemplo “6 meses”), que también heredan sus subcarpetas.</li>
              <li><b>No depurar nunca</b>: esa carpeta (y lo que tiene dentro) no se depura aunque la superior tenga retención.</li>
            </ul>
            <h4>Depurar</h4>
            <ol class="pasos">
              <li>Elija la carpeta (o Inicio para todas) → pestaña <b>Vencidos</b> → <b>Revisar vencidos</b>.</li>
              <li>Revise la lista, agrupada por la carpeta que define la regla.</li>
              <li>Marque los archivos (o grupos completos) → <b>Mover a la papelera</b> → confirme.</li>
            </ol>
            <p class="nota">Nada se borra solo: siempre se revisa y se confirma. Lo depurado queda en <code>_papelera_organizador\&lt;fecha&gt; depuracion\…</code> con su ruta original, y cada archivo queda anotado en el Historial del Excel.</p>
          </template>

          <!-- 8 -->
          <template v-else-if="tema === 'problemas'">
            <h3>Si algo sale mal</h3>
            <table class="problemas">
              <thead><tr><th>Mensaje o síntoma</th><th>Qué hacer</th></tr></thead>
              <tbody>
                <tr><td>“El Excel está abierto…”</td><td>Cierre <code>Fichas tecnicas carpetas.xlsx</code> e intente de nuevo.</td></tr>
                <tr><td>“¿Hay un archivo abierto?”</td><td>Un archivo abierto (o una carpeta con un archivo abierto) no se puede mover, renombrar ni eliminar. Ciérrelo.</td></tr>
                <tr><td>“No respondió a tiempo”</td><td>La unidad de red está lenta o desconectada. Ábrala en el Explorador y pulse Recargar.</td></tr>
                <tr><td>Fichas que apuntan a carpetas que no existen</td><td>Alguien renombró o movió la carpeta por fuera de la app. Corrija la ruta en el Excel o elimine la ficha desde la raíz.</td></tr>
                <tr><td>Cambié algo desde el Explorador y no se ve</td><td>Pulse <b>Recargar</b> (<kbd>F5</kbd>).</td></tr>
                <tr><td>“Tiene fichas o explicaciones en el Excel de…”</td><td>Esa carpeta no se puede mover a otra raíz porque sus fichas están en el Excel de la suya. Muévala dentro de la misma raíz.</td></tr>
                <tr><td>“Esa carpeta está dentro de… que ya está abierta”</td><td>No se puede abrir una raíz dentro de otra. Use la que ya está abierta.</td></tr>
              </tbody>
            </table>
            <p class="nota">¿Se eliminó o depuró algo por error? Está en <code>_papelera_organizador</code>, dentro de su carpeta raíz, ordenado por fecha y con su ruta original.</p>
          </template>

          <footer class="navegar">
            <button v-if="TEMAS.findIndex((t) => t.id === tema) > 0" class="btn fantasma"
                    @click="ir(TEMAS[TEMAS.findIndex((t) => t.id === tema) - 1].id)">← {{ TEMAS[TEMAS.findIndex((t) => t.id === tema) - 1].titulo }}</button>
            <span></span>
            <button v-if="TEMAS.findIndex((t) => t.id === tema) < TEMAS.length - 1" class="btn"
                    @click="ir(TEMAS[TEMAS.findIndex((t) => t.id === tema) + 1].id)">{{ TEMAS[TEMAS.findIndex((t) => t.id === tema) + 1].titulo }} →</button>
          </footer>
        </article>
      </div>
    </div>
  </div>
</template>

<style scoped>
.velo { position: fixed; inset: 0; z-index: 55; display: grid; place-items: center; background: rgba(16, 49, 59, .5); }
.guia {
  width: min(1040px, calc(100vw - 48px)); height: min(720px, calc(100vh - 48px));
  display: flex; flex-direction: column; background: var(--papel); border-radius: var(--r-m);
  box-shadow: 0 24px 64px rgba(16, 49, 59, .35); overflow: hidden; outline: none;
}
header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 18px 24px 16px; background: var(--riel); color: #fff; }
.sobre { margin: 0; font-size: var(--t-xs); text-transform: uppercase; letter-spacing: .08em; color: var(--ambar); font-weight: 600; }
h2 { margin: 2px 0 0; font-size: var(--t-xl); font-weight: 600; }
.cerrar { display: grid; place-items: center; width: 32px; height: 32px; padding: 0; border: 0; border-radius: var(--r-s); background: rgba(255, 255, 255, .08); color: #fff; }
.cerrar:hover { background: rgba(255, 255, 255, .16); }
.cerrar svg { width: 16px; height: 16px; }
.cuerpo { flex: 1; display: grid; grid-template-columns: 240px 1fr; min-height: 0; }
nav { display: grid; align-content: start; gap: 2px; padding: 12px 10px; border-right: 1px solid var(--linea); background: var(--superficie); overflow-y: auto; }
nav button {
  display: flex; align-items: center; gap: 10px; padding: 7px 10px; border: 0; border-radius: var(--r-s);
  background: none; text-align: left; font-size: var(--t-s); color: var(--tinta-2);
}
nav button:hover { background: var(--linea-2); color: var(--tinta); }
nav button.activo { background: var(--ambar-claro); color: var(--tinta); font-weight: 600; box-shadow: inset 3px 0 0 var(--ambar); }
.n { flex: none; width: 20px; height: 20px; display: grid; place-items: center; border-radius: 50%; background: var(--linea-2); font-size: 11px; font-weight: 600; color: var(--tinta-2); }
nav button.activo .n { background: var(--ambar); color: #3d2a00; }
article { overflow-y: auto; padding: 22px 32px 18px; font-size: var(--t-m); line-height: 1.6; color: var(--tinta); }
h3 { margin: 0 0 10px; font-size: var(--t-xl); font-weight: 600; }
h4 { margin: 22px 0 6px; font-size: var(--t-m); font-weight: 600; }
p { margin: 0 0 10px; max-width: 72ch; }
ul, ol { margin: 0 0 10px; padding-left: 22px; max-width: 74ch; }
li { margin-bottom: 4px; }
ol.pasos { list-style: none; padding-left: 0; counter-reset: paso; }
ol.pasos li { counter-increment: paso; position: relative; padding-left: 34px; margin-bottom: 8px; }
ol.pasos li::before {
  content: counter(paso); position: absolute; left: 0; top: 1px; width: 22px; height: 22px; display: grid; place-items: center;
  border-radius: 50%; background: var(--riel); color: #fff; font-size: 12px; font-weight: 600;
}
code { font-size: .9em; padding: 1px 5px; border-radius: 3px; background: var(--linea-2); }
kbd { font: inherit; font-size: var(--t-xs); padding: 1px 6px; border: 1px solid var(--linea); border-bottom-width: 2px; border-radius: 4px; background: #fff; }
.destacado { padding: 12px 16px; border-left: 4px solid var(--ambar); background: var(--superficie); border-radius: 0 var(--r-s) var(--r-s) 0; }
.nota { padding: 10px 14px; background: var(--ambar-claro); border: 1px solid #f0d9a0; border-radius: var(--r-s); font-size: var(--t-s); color: var(--tinta-2); }
.tarjetas { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin: 6px 0 4px; }
.tarjetas div { display: grid; gap: 4px; padding: 12px 14px; background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-m); font-size: var(--t-s); }
.tarjetas span { color: var(--tinta-2); }
table { width: 100%; max-width: 760px; border-collapse: collapse; margin: 4px 0 12px; font-size: var(--t-s); background: var(--superficie); border: 1px solid var(--linea); border-radius: var(--r-s); }
th { text-align: left; padding: 7px 12px; background: #f6f8f9; color: var(--tinta-2); font-weight: 600; border-bottom: 1px solid var(--linea); }
td { padding: 7px 12px; border-bottom: 1px solid var(--linea-2); vertical-align: top; }
tr:last-child td { border-bottom: 0; }
.estados td:first-child, .problemas td:first-child { width: 34%; }
.est { font-weight: 600; white-space: nowrap; }
.sim { display: inline-block; width: 18px; font-weight: 700; color: var(--c); }
.p { display: inline-block; width: 9px; height: 9px; border-radius: 50%; box-shadow: inset 0 0 0 1.5px var(--tinta-3); }
.p.lleno { background: var(--ambar); box-shadow: none; }
.mas { display: inline-grid; place-items: center; width: 20px; height: 20px; border-radius: var(--r-s); background: var(--ambar); color: #3d2a00; font-weight: 700; }
.ic { width: 14px; height: 14px; vertical-align: -2px; color: var(--tinta-2); }
.navegar { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 28px; padding-top: 14px; border-top: 1px solid var(--linea); }
@media (max-width: 760px) {
  .cuerpo { grid-template-columns: 1fr; }
  nav { display: flex; overflow-x: auto; border-right: 0; border-bottom: 1px solid var(--linea); }
  nav button { white-space: nowrap; }
  .tarjetas { grid-template-columns: 1fr; }
}
</style>
