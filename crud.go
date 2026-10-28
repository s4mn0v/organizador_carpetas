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

// Operaciones sobre carpetas y archivos. Reciben y devuelven rutas virtuales ("<raíz>/<ruta>").
// Las fichas del Excel se mantienen al día: al renombrar o mover una carpeta, cambian la
// ruta de su ficha y de las de sus subcarpetas; al eliminarla, se borran (con copia en el Historial).

// registrarHistorial anota en el Excel de la raíz si existe (no detiene la operación si falla).
func (a *App) registrarHistorial(ri *raizInfo, accion, rel, detalle string) {
	b, err := a.baseEscritura(ri)
	if err != nil {
		return
	}
	if f, _, err := b.abrirEscritura(); err == nil {
		historial(f, accion, rutaAExcel(rel), detalle)
		guardarLibro(f)
		f.Close()
	}
	a.olvidarBase(ri)
}

// excelListo: si hay fichas afectadas, el Excel debe poder escribirse ANTES de tocar archivos.
func (a *App) excelListo(ri *raizInfo, rel string) (*baseDatos, error) {
	b, err := a.base(ri)
	if err != nil || len(b.idsBajo(rel))+b.subsBajo(rel) == 0 {
		return nil, nil // sin Excel o sin fichas/descripciones afectadas: nada que actualizar
	}
	return a.baseEscritura(ri)
}

// CrearCarpeta crea una subcarpeta dentro de padre (una raíz o una carpeta).
func (a *App) CrearCarpeta(padre, nombre string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	nombre = strings.TrimSpace(nombre)
	if err := nombreValido(nombre); err != nil {
		return "", err
	}
	if padre == "" {
		return "", errors.New("elija en cuál carpeta raíz se crea")
	}
	ri, relPadre, absPadre, err := a.absoluta(padre)
	if err != nil {
		return "", err
	}
	if !dirExiste(absPadre) {
		return "", errors.New("la carpeta donde se quiere crear ya no existe")
	}
	if relPadre == "" && (strings.EqualFold(nombre, carpetaPapelera) || strings.EqualFold(nombre, nombreBase)) {
		return "", fmt.Errorf("“%s” es un nombre reservado", nombre)
	}
	rel := unir(relPadre, nombre)
	abs := filepath.Join(absPadre, nombre)
	if _, err := os.Stat(abs); err == nil {
		return "", fmt.Errorf("ya existe “%s” en esa carpeta", nombre)
	}
	if err := os.Mkdir(abs, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta: %w", err)
	}
	a.olvidarTamanos()
	a.registrarHistorial(ri, "Crear carpeta", rel, "")
	return unir(ri.ID, rel), nil
}

// Renombrar cambia el nombre de una carpeta o archivo (en su misma ubicación).
func (a *App) Renombrar(v, nuevoNombre string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	nuevoNombre = strings.TrimSpace(nuevoNombre)
	if err := nombreValido(nuevoNombre); err != nil {
		return "", err
	}
	ri, rel, abs, err := a.absoluta(v)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", errors.New("la carpeta raíz no se puede renombrar desde aquí")
	}
	nueva := unir(padreDe(rel), nuevoNombre)
	if nueva == rel {
		return v, nil
	}
	return a.moverA(ri, rel, abs, ri, nueva, "Renombrar")
}

// Mover pasa una carpeta o archivo a otra carpeta (puede ser de otra raíz).
func (a *App) Mover(v, destino string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	ri, rel, abs, err := a.absoluta(v)
	if err != nil {
		return "", err
	}
	if destino == "" {
		return "", errors.New("elija la carpeta destino")
	}
	rd, relDest, absDest, err := a.absoluta(destino)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", errors.New("la carpeta raíz no se puede mover")
	}
	if !dirExiste(absDest) {
		return "", errors.New("la carpeta destino no existe")
	}
	if ri == rd && dentroDe(relDest, rel) {
		return "", errors.New("no se puede mover una carpeta dentro de sí misma")
	}
	if ri == rd && strings.EqualFold(padreDe(rel), relDest) {
		return v, nil
	}
	return a.moverA(ri, rel, abs, rd, unir(relDest, path.Base(rel)), "Mover")
}

func (a *App) moverA(ri *raizInfo, rel, abs string, rd *raizInfo, nueva, accion string) (string, error) {
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("ya no existe: %w", err)
	}
	absNueva, _ := rutaSegura(rd.Ruta, nueva)
	// En Windows "Informe" e "informe" son el mismo nombre: solo se permite si es el mismo elemento.
	if _, err := os.Stat(absNueva); err == nil && !strings.EqualFold(abs, absNueva) {
		return "", fmt.Errorf("ya existe “%s” en el destino", path.Base(nueva))
	}
	var b *baseDatos
	if info.IsDir() {
		if b, err = a.excelListo(ri, rel); err != nil {
			return "", err
		}
		// Entre raíces distintas las fichas no se pasan de un Excel a otro.
		if b != nil && ri != rd {
			return "", fmt.Errorf("“%s” tiene fichas o explicaciones en el Excel de %s: solo se puede mover dentro de esa raíz", path.Base(rel), ri.Nombre)
		}
	}
	if err := moverEntre(abs, absNueva); err != nil {
		return "", fmt.Errorf("no se pudo %s (¿hay un archivo abierto?): %w", strings.ToLower(accion), err)
	}
	a.olvidarTamanos()
	if b != nil {
		if _, err := b.cambiarRuta(rel, nueva); err != nil {
			moverEntre(absNueva, abs)
			a.olvidarBase(ri)
			return "", fmt.Errorf("no se pudo actualizar el Excel; se deshizo el cambio: %w", err)
		}
		a.olvidarBase(ri)
	} else if ri == rd {
		a.registrarHistorial(ri, accion, rel, "Nueva ruta: "+rutaAExcel(nueva))
	} else {
		a.registrarHistorial(ri, accion, rel, "Movido a otra raíz: "+filepath.Join(rd.Ruta, filepath.FromSlash(nueva)))
		a.registrarHistorial(rd, accion, nueva, "Traído de "+abs)
	}
	return unir(rd.ID, nueva), nil
}

// ResumenEliminar: lo que se pierde al eliminar (para el diálogo de confirmación).
type ResumenEliminar struct {
	Carpetas      int  `json:"carpetas"`
	Archivos      int  `json:"archivos"`
	Fichas        int  `json:"fichas"`
	Descripciones int  `json:"descripciones"` // explicaciones de subcarpetas que se borran
	Cortado       bool `json:"cortado"`       // la carpeta es muy grande y se contó solo una parte
}

// ContarEliminar cuenta lo que hay dentro antes de eliminar.
func (a *App) ContarEliminar(v string) (*ResumenEliminar, error) {
	ri, rel, abs, err := a.absoluta(v)
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
	if b, _ := a.base(ri); b != nil {
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			r.Fichas = len(b.idsBajo(rel))
			r.Descripciones = b.subsBajo(rel)
		}
	}
	return r, nil
}

// Eliminar mueve una carpeta o archivo a la papelera de su raíz (_papelera_organizador\<fecha>\…).
// No se usa la papelera de Windows porque en unidades de red (Z:) no existe y borraría para siempre.
func (a *App) Eliminar(v string) (string, error) {
	a.op.Lock()
	defer a.op.Unlock()
	ri, rel, abs, err := a.absoluta(v)
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
		if b, err = a.excelListo(ri, rel); err != nil {
			return "", err
		}
	}
	carpetaDia := unir(unir(carpetaPapelera, time.Now().Format("2006-01-02 150405")), padreDe(rel))
	destino := libre(ri.Ruta, carpetaDia, path.Base(rel))
	absDest, _ := rutaSegura(ri.Ruta, destino)
	if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
		return "", err
	}
	if err := moverEntre(abs, absDest); err != nil {
		return "", fmt.Errorf("no se pudo eliminar (¿hay un archivo abierto?): %w", err)
	}
	a.olvidarTamanos()
	ocultar(filepath.Join(ri.Ruta, carpetaPapelera))
	if b != nil {
		if err := b.eliminarBajo(rel); err != nil {
			a.olvidarBase(ri)
			return destino, fmt.Errorf("se eliminó, pero no se pudieron borrar sus fichas/descripciones del Excel: %w", err)
		}
		a.olvidarBase(ri)
	}
	a.registrarHistorial(ri, "Eliminar", rel, "Movido a "+rutaAExcel(destino))
	return destino, nil
}

// AgregarArchivos copia archivos elegidos (de cualquier parte) a una carpeta.
// Si ya existe uno con el mismo nombre, el copiado queda como "nombre (2)".
func (a *App) AgregarArchivos(destino string) ([]string, error) {
	ri, relDest, absDest, err := a.absoluta(destino)
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
	defer a.olvidarTamanos()
	var hechos []string
	for _, o := range archivos {
		rel := libre(ri.Ruta, relDest, filepath.Base(o))
		dst, _ := rutaSegura(ri.Ruta, rel)
		if strings.EqualFold(filepath.Clean(o), dst) {
			continue
		}
		if err := copiarConFecha(o, dst); err != nil {
			return hechos, fmt.Errorf("no se pudo copiar %s: %w", filepath.Base(o), err)
		}
		hechos = append(hechos, unir(ri.ID, rel))
	}
	if len(hechos) > 0 {
		a.registrarHistorial(ri, "Agregar archivos", relDest, fmt.Sprintf("%d archivo(s)", len(hechos)))
	}
	return hechos, nil
}
