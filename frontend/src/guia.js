// Pasos de la guía (componente Tour). sel: elemento que se ilumina (el primero que exista); null = centrado.

// Parte 1: pantalla inicial, antes de elegir carpeta. Termina cuando el usuario elige su carpeta raíz.
export function pasosInicio(hayRecientes) {
  return [
    {
      sel: null, titulo: 'Bienvenido al Organizador de carpetas',
      texto: `<p>Esta aplicación sirve para <b>ordenar carpetas</b> (de su equipo o de la red, como Z:) y <b>documentarlas</b>
        con una ficha técnica en Excel: para qué sirve cada carpeta, quién responde y cuánto tiempo se guardan sus archivos.</p>
        <p>La guía le muestra todo, desde elegir la carpeta hasta lo básico del día a día. Use <b>Siguiente</b> o las flechas ← →.</p>`,
    },
    {
      sel: '.que-es', titulo: '¿Qué es una carpeta raíz?',
      texto: `<p>Es la carpeta <b>“de arriba”</b> que la app organiza, por ejemplo <code>Z:\\MANTENIMIENTO\\PRIVADA</code>. Todo se hace dentro de ella.</p>
        <ul><li><b>Carpetas principales</b>: las que están directo en la raíz. Cada una lleva su <b>ficha técnica</b>.</li>
        <li><b>Subcarpetas</b>: lo que hay más adentro. Se explican dentro de la ficha de su carpeta principal.</li>
        <li>En la raíz quedan el <b>Excel de fichas</b> y la <b>papelera de la app</b> (lo eliminado no se borra, va ahí).</li></ul>`,
    },
    {
      sel: '.arbolito', titulo: 'Así se ve por dentro',
      texto: `<p>Un ejemplo de los tres niveles. Elija como raíz la carpeta que <b>contiene</b> las áreas que quiere documentar,
        no una subcarpeta suelta.</p><p>Puede abrir <b>varias raíces</b> a la vez (por ejemplo MANTENIMIENTO y RRHH): cada una con su Excel y su color.</p>`,
    },
    ...(hayRecientes ? [{
      sel: '.recientes', titulo: 'Carpetas usadas recientemente',
      texto: `<p>Las últimas carpetas que abrió. Un clic y se abren sin buscarlas.</p>
        <p>La <b>×</b> quita una de la lista y <b>Limpiar recientes</b> borra toda la lista. Las carpetas no se tocan.</p>`,
    }] : []),
    {
      sel: '#btn-elegir', titulo: 'Elija su carpeta raíz',
      texto: `<p>Al pulsar <b>Elegir carpeta</b> se abre el explorador de Windows: busque la carpeta (por ejemplo en Z:) y pulse
        <b>Seleccionar carpeta</b>.</p><p>Si la carpeta ya tiene el Excel de fichas, la app lo lee; si no, podrá crearlo. La guía continúa adentro.</p>`,
      espera: 'Haga clic en «Elegir carpeta»',
    },
  ]
}

// Parte 2: dentro de la app, con una carpeta raíz abierta.
export const pasosApp = [
  {
    sel: null, titulo: '¡Carpeta abierta!',
    texto: `<p>Ahora conozca las partes de la aplicación. A la izquierda está la <b>barra lateral</b> para moverse;
      a la derecha, lo que haya elegido.</p>`,
  },
  {
    sel: '.lista', titulo: 'El árbol de carpetas',
    texto: `<ul><li><b>Inicio</b> reúne todas las carpetas raíz abiertas. Cada raíz tiene su <b>color</b>.</li>
      <li>La flecha <b>›</b> abre o cierra una carpeta; un clic en el nombre la muestra a la derecha.</li>
      <li>El <b>punto de color</b> indica que la carpeta tiene ficha técnica.</li>
      <li>Las carpetas se leen al abrirlas, por eso funciona rápido aunque estén en red.</li></ul>`,
  },
  {
    sel: '.arbol .fila', titulo: 'Clic derecho y arrastrar',
    texto: `<p><b>Clic derecho</b> sobre una carpeta o archivo: menú con nueva carpeta, renombrar, mover, eliminar, ver en el gráfico,
      duplicados y vencidos. En una raíz también puede cambiar su color o quitarla de la lista.</p>
      <p><b>Arrastre</b> una carpeta o archivo y suéltelo sobre otra carpeta para moverlo (pide confirmar).</p>`,
  },
  {
    sel: '.agregar-raiz', titulo: 'Abrir otra carpeta raíz',
    texto: `<p>Agrega otra carpeta raíz sin cerrar la actual. Todas se ven bajo <b>Inicio</b>, cada una con su Excel de fichas y su color.</p>`,
  },
  {
    sel: '.buscador', titulo: 'Buscar',
    texto: `<p>Escriba una o varias palabras (<kbd>Ctrl</kbd> + <kbd>F</kbd>). Encuentra lo que tenga <b>al menos una</b>,
      en cualquier orden, sin importar tildes, y también palabras parecidas (“mantenimeinto” → “mantenimiento”).</p>`,
  },
  {
    sel: '.vistas', titulo: 'Cuatro vistas',
    texto: `<ul><li><b>Carpetas</b>: la vista normal.</li><li><b>Gráfica</b>: las carpetas como una red, con zoom y búsqueda.</li>
      <li><b>Duplicados</b>: archivos con el mismo nombre y contenido.</li>
      <li><b>Vencidos</b>: archivos que pasaron su tiempo de retención, para depurarlos.</li></ul>`,
  },
  {
    sel: '.detalle .cabecera', titulo: 'La carpeta elegida',
    texto: `<p>Arriba, la <b>ruta</b>: cada parte es un botón para subir de nivel. A la derecha, los botones para
      <b>crear subcarpetas</b>, <b>agregar archivos</b>, verla en el gráfico, renombrar, mover o eliminar.</p>`,
  },
  {
    sel: '.cifras', titulo: 'Cifras de la carpeta',
    texto: `<p>Subcarpetas, archivos y <b>tamaño total</b> contando todas sus subcarpetas. El número va subiendo mientras se suma
      y queda guardado: al volver aparece al instante. Se recalcula solo si algo cambia o con <b>Recargar</b>.</p>`,
  },
  {
    sel: ['.principales', '.aviso', '.detalle .bloque'], titulo: 'Fichas técnicas',
    texto: `<p>Si la raíz no tiene Excel de fichas, aquí aparece el botón para <b>crearlo</b>. Luego, en cada carpeta principal:</p>
      <ul><li><b>Crear ficha técnica</b> → llene la sección General.</li>
      <li><b>Retención de archivos</b>: cuánto tiempo deben quedarse los archivos (por ejemplo “2 años”).</li>
      <li>En <b>Subcarpetas</b> explique para qué sirve cada una.</li><li><b>Guardar en Excel</b>.</li></ul>`,
  },
  {
    sel: ['.tabla-caja', '.detalle .bloque:last-child'], titulo: 'Contenido',
    texto: `<p>Lo que hay dentro de la carpeta. Clic en un encabezado para ordenar; <b>doble clic</b> en una fila para entrar o abrir el archivo.
      Al pasar el mouse aparecen los botones de abrir, renombrar, mover y eliminar. También funciona el clic derecho.</p>`,
  },
  {
    sel: '.btn.nueva', titulo: 'Lo más básico: crear una carpeta',
    texto: `<p>Crea una carpeta dentro de la que tenga seleccionada. Escriba el nombre y pulse <b>Crear carpeta</b>.</p>
      <p>Para renombrar algo seleccionado use <kbd>F2</kbd>.</p>`,
  },
  {
    sel: '.botones', titulo: 'Botones de abajo',
    texto: `<ul><li><b>Recargar</b> (<kbd>F5</kbd>): vuelve a leer todo, por si cambió algo desde el Explorador.</li>
      <li><b>Abrir Excel</b>: el Excel de fichas de la raíz elegida.</li>
      <li><b>Papelera</b>: lo eliminado desde la app, por fecha y con su ruta original.</li>
      <li><b>Ayuda</b> (<kbd>F1</kbd>): la guía escrita, y desde ahí puede repetir esta guía.</li></ul>`,
  },
  {
    sel: null, titulo: '¡Listo!',
    texto: `<p>Ya conoce lo básico. Reglas para estar tranquilo:</p>
      <ul><li><b>Nada se borra de golpe</b>: eliminar o depurar mueve a la papelera de la app.</li>
      <li>Si renombra o mueve una carpeta, su ficha se actualiza sola en el Excel.</li>
      <li>Cada cambio queda en la hoja <b>Historial</b> del Excel.</li></ul>`,
  },
]
