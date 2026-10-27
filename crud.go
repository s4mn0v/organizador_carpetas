package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Operaciones sobre carpetas y archivos. Todas trabajan con rutas relativas a la raíz.
// Las fichas del Excel se mantienen al día: al renombrar o mover una carpeta, cambian la
// ruta de su ficha y de las de sus subcarpetas; al eliminarla, se borran (con copia en el Historial).

func (a *App) absoluta(rel string) (string, string, error) {
	raiz := a.raizActual()
	if raiz == "" {
		return "", "", errors.New("elija la carpeta raíz")
	}
	rel, ok := limpiarRel(rel)
	if !ok {
		return "", "", errors.New("ruta no válida")
	}
	abs, _ := rutaSegura(raiz, rel)
	return rel, abs, nil
}

// registrarHistorial anota en el Excel si existe (no detiene la operación si falla).
func (a *App) registrarHistorial(accion, rel, detalle string) {
	b, err := a.baseEscritura()
	if err != nil {
		return
	}
	if f, _, err := b.abrirEscritura(); err == nil {
		historial(f, accion, rutaAExcel(rel), detalle)
		guardarLibro(f)
		f.Close()
	}
	a.olvidarBase()
}

// excelListo: si hay fichas afectadas, el Excel debe poder escribirse ANTES de tocar archivos.
func (a *App) excelListo(rel string) (*baseDatos, error) {
	b, err := a.base()
	if err != nil || len(b.idsBajo(rel)) == 0 {
		return nil, nil // sin Excel o sin fichas afectadas: nada que actualizar
	}
	return a.baseEscritura()
}

// CrearCarpeta crea una subcarpeta dentro de padre ("" = raíz).
func (a *App) CrearCarpeta(padre, nombre string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	nombre = strings.TrimSpace(nombre)
	if err := nombreValido(nombre); err != nil {
		return "", err
	}
	padre, absPadre, err := a.absoluta(padre)
	if err != nil {
		return "", err
	}
	if !dirExiste(absPadre) {
		return "", errors.New("la carpeta donde se quiere crear ya no existe")
	}
	if padre == "" && (strings.EqualFold(nombre, carpetaPapelera) || strings.EqualFold(nombre, nombreBase)) {
		return "", fmt.Errorf("“%s” es un nombre reservado", nombre)
	}
	rel := unir(padre, nombre)
	abs := filepath.Join(absPadre, nombre)
	if _, err := os.Stat(abs); err == nil {
		return "", fmt.Errorf("ya existe “%s” en esa carpeta", nombre)
	}
	if err := os.Mkdir(abs, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta: %w", err)
	}
	a.registrarHistorial("Crear carpeta", rel, "")
	return rel, nil
}

// Renombrar cambia el nombre de una carpeta o archivo (en su misma ubicación).
func (a *App) Renombrar(rel, nuevoNombre string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	nuevoNombre = strings.TrimSpace(nuevoNombre)
	if err := nombreValido(nuevoNombre); err != nil {
		return "", err
	}
	rel, abs, err := a.absoluta(rel)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", errors.New("la carpeta raíz no se puede renombrar desde aquí")
	}
	nueva := unir(padreDe(rel), nuevoNombre)
	if nueva == rel {
		return rel, nil
	}
	return a.moverA(rel, abs, nueva, "Renombrar")
}

// Mover pasa una carpeta o archivo a otra carpeta (destino "" = raíz).
func (a *App) Mover(rel, destino string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	rel, abs, err := a.absoluta(rel)
	if err != nil {
		return "", err
	}
	destino, absDest, err := a.absoluta(destino)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", errors.New("la carpeta raíz no se puede mover")
	}
	if !dirExiste(absDest) {
		return "", errors.New("la carpeta destino no existe")
	}
	if dentroDe(destino, rel) {
		return "", errors.New("no se puede mover una carpeta dentro de sí misma")
	}
	if strings.EqualFold(padreDe(rel), destino) {
		return rel, nil
	}
	return a.moverA(rel, abs, unir(destino, path.Base(rel)), "Mover")
}

func (a *App) moverA(rel, abs, nueva, accion string) (string, error) {
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("ya no existe: %w", err)
	}
	absNueva, _ := rutaSegura(a.raizActual(), nueva)
	// En Windows "Informe" e "informe" son el mismo nombre: solo se permite si es el mismo elemento.
	if _, err := os.Stat(absNueva); err == nil && !strings.EqualFold(abs, absNueva) {
		return "", fmt.Errorf("ya existe “%s” en el destino", path.Base(nueva))
	}
	var b *baseDatos
	if info.IsDir() {
		if b, err = a.excelListo(rel); err != nil {
			return "", err
		}
	}
	if err := moverEntre(abs, absNueva); err != nil {
		return "", fmt.Errorf("no se pudo %s (¿hay un archivo abierto?): %w", strings.ToLower(accion), err)
	}
	if b != nil {
		if _, err := b.cambiarRuta(rel, nueva); err != nil {
			moverEntre(absNueva, abs)
			a.olvidarBase()
			return "", fmt.Errorf("no se pudo actualizar el Excel; se deshizo el cambio: %w", err)
		}
		a.olvidarBase()
	} else {
		a.registrarHistorial(accion, rel, "Nueva ruta: "+rutaAExcel(nueva))
	}
	return nueva, nil
}

// ResumenEliminar: lo que se pierde al eliminar (para el diálogo de confirmación).
type ResumenEliminar struct {
	Carpetas int  `json:"carpetas"`
	Archivos int  `json:"archivos"`
	Fichas   int  `json:"fichas"`
	Cortado  bool `json:"cortado"` // la carpeta es muy grande y se contó solo una parte
}

// ContarEliminar cuenta lo que hay dentro antes de eliminar.
func (a *App) ContarEliminar(rel string) (*ResumenEliminar, error) {
	rel, abs, err := a.absoluta(rel)
	if err != nil {
		return nil, err
	}
	r := &ResumenEliminar{}
	limite := time.Now().Add(8 * time.Second)
	fin := errors.New("fin")
	filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == abs {
			return nil
		}
		if d.IsDir() {
			r.Carpetas++
		} else {
			r.Archivos++
		}
		if time.Now().After(limite) {
			r.Cortado = true
			return fin
		}
		return nil
	})
	if b, _ := a.base(); b != nil {
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			r.Fichas = len(b.idsBajo(rel))
		}
	}
	return r, nil
}

// Eliminar mueve una carpeta o archivo a la papelera de la app (_papelera_organizador\<fecha>\…).
// No se usa la papelera de Windows porque en unidades de red (Z:) no existe y borraría para siempre.
func (a *App) Eliminar(rel string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	rel, abs, err := a.absoluta(rel)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", errors.New("la carpeta raíz no se puede eliminar")
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("ya no existe: %w", err)
	}
	var b *baseDatos
	if info.IsDir() {
		if b, err = a.excelListo(rel); err != nil {
			return "", err
		}
	}
	carpetaDia := unir(unir(carpetaPapelera, time.Now().Format("2006-01-02 150405")), padreDe(rel))
	destino := libre(a.raizActual(), carpetaDia, path.Base(rel))
	absDest, _ := rutaSegura(a.raizActual(), destino)
	if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
		return "", err
	}
	if err := moverEntre(abs, absDest); err != nil {
		return "", fmt.Errorf("no se pudo eliminar (¿hay un archivo abierto?): %w", err)
	}
	ocultar(filepath.Join(a.raizActual(), carpetaPapelera))
	if b != nil {
		ids := b.idsBajo(rel)
		if err := b.eliminar(ids, "Eliminar carpeta"); err != nil {
			a.olvidarBase()
			return destino, fmt.Errorf("se eliminó, pero no se pudieron borrar sus %d fichas del Excel: %w", len(ids), err)
		}
		a.olvidarBase()
	}
	a.registrarHistorial("Eliminar", rel, "Movido a "+rutaAExcel(destino))
	return destino, nil
}

// AgregarArchivos copia archivos elegidos (de cualquier parte) a una carpeta.
// Si ya existe uno con el mismo nombre, el copiado queda como "nombre (2)".
func (a *App) AgregarArchivos(destino string) ([]string, error) {
	destino, absDest, err := a.absoluta(destino)
	if err != nil {
		return nil, err
	}
	archivos, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Archivos para agregar a " + filepath.Base(absDest),
	})
	if err != nil || len(archivos) == 0 {
		return nil, err
	}
	a.op.Lock()
	defer a.op.Unlock()
	var hechos []string
	for _, o := range archivos {
		rel := libre(a.raizActual(), destino, filepath.Base(o))
		dst, _ := rutaSegura(a.raizActual(), rel)
		if strings.EqualFold(filepath.Clean(o), dst) {
			continue
		}
		if err := copiarConFecha(o, dst); err != nil {
			return hechos, fmt.Errorf("no se pudo copiar %s: %w", filepath.Base(o), err)
		}
		hechos = append(hechos, rel)
	}
	if len(hechos) > 0 {
		a.registrarHistorial("Agregar archivos", destino, fmt.Sprintf("%d archivo(s)", len(hechos)))
	}
	return hechos, nil
}
