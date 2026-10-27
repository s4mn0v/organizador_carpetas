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
            <p>La aplicación sirve para ordenar una carpeta (local o de red) y documentar sus subcarpetas. Trabaja con tres cosas:</p>
            <div class="tarjetas">
              <div><b>La carpeta raíz</b><span>La que usted elige, por ejemplo <code>Z:\MANTENIMIENTO\PRIVADA\14. Dirección Mtto</code>. Todo se hace dentro de ella.</span></div>
              <div><b>El Excel de fichas</b><span><code>Fichas tecnicas carpetas.xlsx</code>, guardado en la raíz: una ficha técnica por carpeta principal, con la explicación de sus subcarpetas.</span></div>
              <div><b>La papelera de la app</b><span><code>_papelera_organizador</code>, en la raíz: lo que elimine va ahí, no se borra.</span></div>
            </div>
            <p class="destacado">Elija la raíz → navegue las carpetas → organícelas (crear, renombrar, mover) → documente cada carpeta principal con su <b>ficha técnica</b>.</p>
            <h4>Reglas</h4>
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
              <li>Arriba, la <b>ruta de la raíz</b>: haga clic para elegir otra carpeta.</li>
              <li><b>Carpetas / Vista gráfica</b>: cambia entre la vista normal y el gráfico.</li>
              <li><b>Buscar</b>: busca por nombre en todas las carpetas, sin importar tildes ni mayúsculas (<kbd>Ctrl</kbd> + <kbd>F</kbd>).</li>
              <li><b>Mostrar archivos</b>: muestra u oculta los archivos en el árbol.</li>
            </ul>
            <h4>El árbol</h4>
            <table>
              <tbody>
                <tr><td><span class="mas">›</span></td><td>Abre o cierra la carpeta sin seleccionarla.</td></tr>
                <tr><td>Clic en el nombre</td><td>Selecciona la carpeta o el archivo y la abre a la derecha.</td></tr>
                <tr><td><i class="p lleno"></i></td><td>La carpeta tiene ficha técnica en el Excel.</td></tr>
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
              <li><b>Cifras</b>: subcarpetas, archivos, tamaño, fecha y si tiene ficha.</li>
              <li><b>Ficha técnica</b>: la información de la carpeta guardada en el Excel.</li>
              <li><b>Contenido</b>: las subcarpetas y archivos. Haga clic en un encabezado para ordenar; doble clic en una fila para entrar o abrir.</li>
            </ol>
            <p class="nota">En la raíz también verá si falta el Excel de fichas (botón para crearlo) y si hay fichas que apuntan a carpetas que ya no existen.</p>
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
                <tr><td><b>Mover</b></td><td>Elija la carpeta destino en el árbol que aparece. Una carpeta se mueve con todo su contenido y sus fichas.</td></tr>
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
              <li>En <b>Subcarpetas</b> escriba para qué sirve cada una (todas las que tiene, en todos los niveles). Las que deje vacías no se guardan.</li>
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
              <li><b>Subcarpetas</b>: una fila por subcarpeta explicada (ruta y descripción).</li>
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
              <li><b>Niveles</b>: cuántos niveles de subcarpetas dibujar. <b>Resaltar</b>: ilumina los que coinciden con el texto.</li>
            </ul>
            <p class="nota">Para no saturar, se dibujan como máximo 2.500 elementos. Si la carpeta es muy grande, baje los niveles o dibuje desde una subcarpeta.</p>
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
              </tbody>
            </table>
            <p class="nota">¿Se eliminó algo por error? Está en <code>_papelera_organizador</code>, dentro de la raíz, ordenado por fecha y con su ruta original.</p>
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
