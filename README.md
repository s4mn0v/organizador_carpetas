# Organizador de carpetas

Aplicación de escritorio (Wails + Go + Vue) para organizar una carpeta (local o de red, p. ej.
`Z:\MANTENIMIENTO\PRIVADA\14. Dirección Mtto`) y documentar cada carpeta y subcarpeta con una
**ficha técnica** en Excel. El `.exe` se ejecuta fuera de la carpeta; los cambios se hacen directo en ella.

## Uso

- Al abrir, elija la **carpeta raíz**. La app la recuerda (y las últimas 6 usadas).
- **Barra lateral**: árbol de carpetas **y archivos**, que se lee por demanda (rápido en red), buscador
  en todas las carpetas (`Ctrl+F`) y cambio entre **Carpetas** y **Vista gráfica**.
- **Carpeta**: ruta navegable, cifras (subcarpetas, archivos, tamaño, fecha, ficha), ficha técnica y contenido.
- **Archivo**: tipo, tamaño, fechas, ubicación; abrir, mostrar en el Explorador, renombrar, mover, eliminar.
- Atajos: `F1` ayuda, `F2` renombrar, `F5` recargar.

## Excel de fichas técnicas

`Fichas tecnicas carpetas.xlsx`, en la **raíz elegida**. Si no existe, la app lo crea (botón en la raíz).

- Las fichas van **solo en las carpetas principales** (las que están directo en la raíz).
  Las subcarpetas (todos los niveles) se **explican dentro** de la ficha de su carpeta principal.
- Hoja **Carpetas**: `ID`, `Ruta carpeta`, `Actualizado` y los campos de la sección General
  (nombre, descripción, proceso/área, responsable, tipo de contenido).
- Hoja **Subcarpetas**: `Ruta subcarpeta`, `Descripción`, `Actualizado`.
- Hoja **Campos**: cómo se muestra cada campo (grupo; tipo: texto, largo, lista, numero, marca; opciones).
- Hoja **Historial**: cada cambio hecho desde la app.
- Si el Excel está abierto en Excel, la app no escribe y lo avisa.
- Los Excel creados con la primera versión traían las secciones "Gestión documental" y "Notas":
  la app las oculta y las quita del Excel sola (filas de Campos y columnas de Carpetas), guardando antes
  un respaldo en `_papelera_organizador\respaldos Excel\`.

## Operaciones

| Acción | Carpetas / archivos | Excel |
|---|---|---|
| Nueva carpeta | Crea la carpeta | Anota en Historial |
| Agregar archivos | Copia archivos de cualquier parte (si el nombre existe: `nombre (2)`) | Anota en Historial |
| Renombrar / Mover | Renombra o mueve (una carpeta, con todo su contenido) | Actualiza la ruta de su ficha y de las explicaciones de sus subcarpetas |
| Eliminar | Mueve a `_papelera_organizador\<fecha>\<ruta>` en la raíz | Borra su ficha y las explicaciones de lo que contenía (copiadas en Historial) |

No se usa la papelera de Windows porque en unidades de red no existe y borraría para siempre.
`_papelera_organizador` y el Excel no se muestran en el árbol.

## Vista gráfica

Red de carpetas y archivos (estilo Obsidian) desde la carpeta elegida: niveles configurables (1–6),
con o sin archivos, resaltar por nombre, zoom con la rueda, arrastre, clic para seleccionar y doble clic para
entrar en una carpeta o abrir un archivo. Máximo 2.500 elementos por vista.
