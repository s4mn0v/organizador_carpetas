# Organizador de carpetas

Aplicación de escritorio (Wails + Go + Vue) para organizar carpetas (locales o de red, p. ej.
`Z:\MANTENIMIENTO\PRIVADA` y `Z:\RRHH\PRIVADA`) y documentar cada carpeta y subcarpeta con una
**ficha técnica** en Excel. El `.exe` se ejecuta fuera de las carpetas; los cambios se hacen directo en ellas.

## Uso

- La app abre en una pantalla de bienvenida (sin barra lateral) que explica qué es una carpeta raíz. El explorador
  de Windows solo se abre al pulsar **Elegir carpeta**. Las carpetas recientes se pueden abrir con un clic, quitar
  una por una (×) o borrar con **Limpiar recientes**.
- Botón **GUÍA** (también en el Inicio y en Ayuda → *Ver guía paso a paso*): recorrido que ilumina cada parte de
  la pantalla, desde elegir la carpeta raíz hasta crear una carpeta (`frontend/src/guia.js`, componente `Tour.vue`).
- Al abrir, elija una **carpeta raíz**. Con **Agregar carpeta raíz** se abren más: todas se ven juntas bajo
  **Inicio**, cada una con **su propio Excel de fichas** y **su color** de carpetas (automático: amarillo, azul,
  verde…; se cambia con clic derecho → *Cambiar color* y se recuerda). No se puede abrir una raíz dentro de otra.
  Las últimas 8 usadas quedan en recientes.
- **Barra lateral**: árbol de carpetas **y archivos**, que se lee por demanda (rápido en red), buscador
  en todas las raíces (`Ctrl+F`) y pestañas **Carpetas**, **Gráfica**, **Duplicados** y **Vencidos**.
  - **Clic derecho**: menú con nueva carpeta, agregar archivos, renombrar, mover, eliminar, gráfico, duplicados,
    vencidos (en una raíz: color, abrir su Excel, quitarla de la lista).
  - **Arrastrar y soltar** una carpeta o archivo sobre otra carpeta: mover (pide confirmar).
- **Carpeta**: ruta navegable, cifras (subcarpetas, **archivos y tamaño de la carpeta con todas sus subcarpetas**,
  fecha, ficha, retención), ficha técnica y contenido. El total se calcula en segundo plano y el número va
  subiendo mientras suma. Queda guardado: al volver a una carpeta aparece al instante, y el cálculo sigue aunque
  se cambie de carpeta. Solo se recalcula si algo cambia en ella (crear, renombrar, mover, eliminar, agregar,
  depurar), en sus carpetas superiores o con **Recargar** (F5). Las subcarpetas ya calculadas no se recorren otra vez.
- **Archivo**: tipo, tamaño, fechas, ubicación; abrir, mostrar en el Explorador, renombrar, mover, eliminar.
- Atajos: `F1` ayuda, `F2` renombrar, `F5` recargar.

## Buscar

Mismo criterio en el buscador lateral, el de la vista gráfica y los filtros de Duplicados y Vencidos
(`busqueda.go` y `puntuar` en `frontend/src/util.js`):

- Coincide si el nombre tiene **al menos una** de las palabras, en **cualquier orden**, sin tildes ni mayúsculas.
- También palabras **parecidas**: que empiezan igual, que están dentro de otra o con 1–2 letras distintas.
- Orden: primero los que tienen más palabras, luego los más parecidos. Se devuelven los 300 mejores.

## Excel de fichas técnicas

`Fichas tecnicas carpetas.xlsx`, en **cada carpeta raíz**. Si no existe, la app lo crea (botón en la raíz).

- Las fichas van **solo en las carpetas principales** (las que están directo en la raíz).
  Las subcarpetas (todos los niveles) se **explican dentro** de la ficha de su carpeta principal.
- Hoja **Carpetas**: `ID`, `Ruta carpeta`, `Actualizado` y los campos de la sección General
  (nombre, descripción, proceso/área, responsable, tipo de contenido, **retención de archivos**).
  "Tipo de contenido" admite **varios valores** (tipo `multilista`): se guardan en la misma celda
  separados por `;` (p. ej. `Informes; Formatos`).
- Hoja **Subcarpetas**: `Ruta subcarpeta`, `Descripción`, `Actualizado`, `Retención`.
- Los Excel creados antes de la retención reciben la columna `Retención de archivos` (y su fila en Campos)
  la primera vez que se guarda una ficha.
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
| Renombrar / Mover | Renombra o mueve (una carpeta, con todo su contenido). Se puede mover a otra raíz si no tiene fichas ni explicaciones en el Excel de la suya | Actualiza la ruta de su ficha y de las explicaciones de sus subcarpetas |
| Eliminar | Mueve a `_papelera_organizador\<fecha>\<ruta>` en su raíz | Borra su ficha y las explicaciones de lo que contenía (copiadas en Historial) |
| Depurar vencidos | Mueve los archivos marcados a `_papelera_organizador\<fecha> depuracion\<ruta>` | Una fila por archivo en Historial |

No se usa la papelera de Windows porque en unidades de red no existe y borraría para siempre.
`_papelera_organizador` y el Excel no se muestran en el árbol.

## Vista gráfica

Red de carpetas y archivos (estilo Obsidian) desde la carpeta elegida o desde Inicio (todas las raíces, cada una
con su color): niveles configurables (1–6), con o sin archivos, zoom con la rueda, arrastre, clic para seleccionar
y doble clic para entrar en una carpeta o abrir un archivo. Máximo 2.500 elementos por vista.

**Resaltar y buscar**: ilumina lo que se parece al texto y despliega la lista de coincidencias; al elegir una
(clic o `↓` + `Enter`) el gráfico hace zoom hasta ese punto. *Buscar también en las subcarpetas no dibujadas*
busca a cualquier profundidad: al elegir un resultado se dibuja desde su carpeta y se enfoca.

## Duplicados

Archivos con el **mismo nombre y el mismo contenido** en la carpeta elegida y todas sus subcarpetas (o en todas
las raíces desde Inicio). Primero se agrupan por nombre + tamaño y solo esos se leen para comparar su huella
SHA-256. Por cada grupo: copias, espacio de más y, por copia, ubicación, tamaño, fecha de creación y de
modificación (la más antigua se marca). Cada copia se puede abrir, mostrar en el Explorador o eliminar (papelera de la app).

## Retención y vencidos

- **Retención de archivos** (ficha de la carpeta principal): tiempo que deben quedarse los archivos
  (`6 meses`, `2 años`, `30 días`, `8 semanas`) o `No depurar`.
- Cada subcarpeta puede tener la suya (sección Subcarpetas de la ficha). Sin valor, **hereda** la de la carpeta
  que la contiene; `No depurar` corta la herencia.
- **Vencido** = fecha de modificación más antigua que la retención.
- Pestaña **Vencidos**: revisa la carpeta elegida (o todas), agrupa por la carpeta que define la regla y permite
  marcar y **mover a la papelera** (con confirmación; con 20 o más archivos pide escribir `DEPURAR`).
  Nada se borra de forma automática.

## Ubicaciones recientemente utilizados

Para limpiar el "historial" de ubicaciones utilizadas recientemente ver el archivo generado automaticamente en:

**C:\Users\nombre_usuario\AppData\Roaming\organizador_carpetas\config.json**

Ahí también quedan los colores elegidos para cada raíz (`colores`).

## Firmar el ejecutable con certificado propio

Windows (Smart App Control / SmartScreen) bloquea los `.exe` sin firma con el mensaje
*"Bloqueamos … porque no pudimos comprobar su publicador"*. El `.exe` se firma con un certificado propio y
se entrega junto con un script que hace que el equipo del usuario confíe en esa firma.

**Al usuario se le entregan solo 2 archivos** (salen en `build\bin\` al ejecutar `firmar.ps1`):

| Archivo | Qué es |
|---|---|
| `organizador_carpetas.exe` | La app, ya firmada |
| `instalar_certificado.ps1` | Instala la parte **pública** del certificado (la lleva adentro). Una vez por equipo |

> Un certificado propio solo es de confianza en los equipos donde se ejecutó `instalar_certificado.ps1`.
> Para que el `.exe` abra en cualquier equipo sin instalar nada hay que firmar con un certificado de una
> entidad reconocida por Microsoft (comercial o Azure Trusted Signing).

### Para el desarrollador

**1. Crear el certificado (una sola vez)**

PowerShell normal, en la carpeta del proyecto:

```powershell
$c = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=S4M, O=SAO6" `
     -FriendlyName "S4M firma de codigo" -CertStoreLocation Cert:\CurrentUser\My `
     -KeyAlgorithm RSA -KeyLength 3072 -HashAlgorithm SHA256 `
     -NotAfter (Get-Date).AddYears(5)

# Respaldo con clave privada, protegido a su usuario de Windows (sin contraseña)
Export-PfxCertificate -Cert $c -FilePath ".\S4M-Certificate.pfx" -ProtectTo "$env:USERNAME" -Force
```

- El **Subject** es el nombre que Windows muestra como publicador: use un nombre propio, nunca el de otra empresa.
- `S4M-Certificate.pfx` tiene la **clave privada**: no va al repositorio (`.gitignore` excluye `*.pfx`, `*.p12`,
  `*.cer`) ni se entrega a nadie.
- Ver los certificados: `Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | Format-List Subject, Thumbprint, NotAfter`
- Borrar uno que no sirve: `Remove-Item Cert:\CurrentUser\My\<Thumbprint>`

**2. Compilar, firmar y generar el paquete (cada versión)**

```powershell
powershell -ExecutionPolicy Bypass -File .\firmar.ps1               # compila, firma y genera el paquete
powershell -ExecutionPolicy Bypass -File .\firmar.ps1 -SinCompilar  # solo firma el .exe existente
```

El script compila (`wails build -webview2 embed`), toma el certificado de `Cert:\CurrentUser\My`
(Subject con `S4M`; si no está, usa el `.pfx`), firma el `.exe` (SHA256 + sello de tiempo) y genera
`build\bin\instalar_certificado.ps1`.

> Al recompilar, el `.exe` sale **sin firma**: siempre se entrega lo que deja `firmar.ps1`.

### Para el usuario (una vez por equipo)

1. Copie `organizador_carpetas.exe` e `instalar_certificado.ps1` a la **misma carpeta**.
2. Clic derecho en `instalar_certificado.ps1` → **Ejecutar con PowerShell**.
3. Acepte el permiso de administrador. Si no tiene, se instala solo para su usuario y Windows muestra una
   **Advertencia de seguridad** ("¿Desea instalar este certificado?"): responda **Sí**.
4. Debe aparecer *"Listo: el equipo confía en organizador_carpetas.exe"*. Presione Enter.
5. Abra `organizador_carpetas.exe`.

Las versiones nuevas del `.exe` no necesitan repetir el paso: basta con reemplazar el `.exe`.
Solo se repite si se crea un certificado nuevo.

Si al hacer clic derecho no aparece *Ejecutar con PowerShell* o lo bloquea, en PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File "C:\ruta\instalar_certificado.ps1"
```

Comprobar a mano: clic derecho al `.exe` → **Propiedades → Firmas digitales**, o
`Get-AuthenticodeSignature .\organizador_carpetas.exe` (`Valid` = el equipo confía).

> **Smart App Control** puede seguir bloqueando aunque el certificado esté instalado, porque solo acepta
> entidades reconocidas por Microsoft. Si pasa: desactivarlo en *Seguridad de Windows → Control de
> aplicaciones y del explorador → Smart App Control* (TI) o firmar con certificado comercial.

**Quitar el certificado de un equipo**

```powershell
Get-ChildItem Cert:\LocalMachine\Root, Cert:\LocalMachine\TrustedPublisher, Cert:\CurrentUser\Root, Cert:\CurrentUser\TrustedPublisher |
    Where-Object Subject -like "*S4M*" | Remove-Item
```

### Renovar (antes de que venza)

Repita el paso 1 y el 2, y entregue el `instalar_certificado.ps1` nuevo para que cada usuario lo ejecute.
Los `.exe` ya firmados siguen siendo válidos por el sello de tiempo.
