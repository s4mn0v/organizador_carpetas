package main

// Subir carpetas completas (de cualquier parte del PC o de la red) dentro de una carpeta de la raíz,
// y abrir carpetas en el Explorador.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ElegirCarpetaOrigen abre el selector de Windows para elegir una carpeta a subir ("" si cancela).
func (a *App) ElegirCarpetaOrigen() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Carpeta que quiere subir"})
}

// ResultadoSubir: lo que se copió.
type ResultadoSubir struct {
	Carpetas  []string `json:"carpetas"` // rutas virtuales de las carpetas creadas
	Archivos  int      `json:"archivos"`
	Tamano    int64    `json:"tamano"`
	Errores   []string `json:"errores"`
	Cancelado bool     `json:"cancelado"`
}

// ProgresoSubir: evento "subir:progreso".
type ProgresoSubir struct {
	Carpeta  string `json:"carpeta"` // la que se está copiando
	Archivos int    `json:"archivos"`
	Tamano   int64  `json:"tamano"`
	Actual   string `json:"actual"` // archivo en curso
}

var cancelarSubida atomic.Bool

// CancelarSubida detiene la copia en curso (lo ya copiado se queda).
func (a *App) CancelarSubida() { cancelarSubida.Store(true) }

// SubirCarpetas copia cada carpeta de origenes (rutas de Windows) dentro de destino (ruta virtual),
// con todo su contenido. Si ya existe una con el mismo nombre, queda como "nombre (2)".
func (a *App) SubirCarpetas(destino string, origenes []string) (*ResultadoSubir, error) {
	ri, relDest, absDest, err := a.absoluta(destino)
	if err != nil {
		return nil, err
	}
	if !dirExiste(absDest) {
		return nil, errors.New("la carpeta destino ya no existe")
	}
	if len(origenes) == 0 {
		return nil, errors.New("elija al menos una carpeta")
	}
	a.op.Lock()
	defer a.op.Unlock()
	cancelarSubida.Store(false)

	res := &ResultadoSubir{Carpetas: []string{}, Errores: []string{}}
	defer func() { a.olvidarTamanos(destino) }() // índice y tamaños se ponen al día
	prog := ProgresoSubir{}
	ultimo := time.Time{}
	avisar := func(forzar bool) {
		if forzar || time.Since(ultimo) > 200*time.Millisecond {
			ultimo = time.Now()
			prog.Archivos, prog.Tamano = res.Archivos, res.Tamano
			a.emitir("subir:progreso", prog)
		}
	}

	for _, o := range origenes {
		o = filepath.Clean(o)
		info, err := os.Stat(o)
		if err != nil || !info.IsDir() {
			res.Errores = append(res.Errores, o+": no es una carpeta o ya no existe")
			continue
		}
		// No se puede subir una carpeta dentro de sí misma (ni el destino dentro de lo que se sube).
		if strings.EqualFold(o, absDest) || dentroDeDisco(absDest, o) {
			res.Errores = append(res.Errores, filepath.Base(o)+": el destino está dentro de esa carpeta")
			continue
		}
		if relDest == "" && (strings.EqualFold(info.Name(), carpetaPapelera) || strings.EqualFold(info.Name(), nombreBase)) {
			res.Errores = append(res.Errores, info.Name()+": es un nombre reservado")
			continue
		}
		rel := libre(ri.Ruta, relDest, info.Name())
		absNueva, _ := rutaSegura(ri.Ruta, rel)
		prog.Carpeta = filepath.Base(absNueva)
		archivosAntes := res.Archivos

		err = filepath.WalkDir(o, func(p string, d os.DirEntry, err error) error {
			if cancelarSubida.Load() {
				return filepath.SkipAll
			}
			if err != nil {
				res.Errores = append(res.Errores, p+": "+err.Error())
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			r, _ := filepath.Rel(o, p)
			dst := filepath.Join(absNueva, r)
			if d.IsDir() {
				if err := os.MkdirAll(dst, 0o755); err != nil {
					res.Errores = append(res.Errores, r+": "+err.Error())
					return filepath.SkipDir
				}
				return nil
			}
			if omitir(d.Name(), false) { // temporales de Office, desktop.ini, Thumbs.db
				return nil
			}
			prog.Actual = r
			if err := copiarConFecha(p, dst); err != nil {
				res.Errores = append(res.Errores, r+": "+err.Error())
				return nil
			}
			if fi, err := d.Info(); err == nil {
				res.Tamano += fi.Size()
			}
			res.Archivos++
			avisar(false)
			return nil
		})
		res.Carpetas = append(res.Carpetas, unir(ri.ID, rel))
		a.registrarHistorial(ri, "Subir carpeta", rel, fmt.Sprintf("Desde %s (%d archivo(s))", o, res.Archivos-archivosAntes))
		if cancelarSubida.Load() {
			res.Cancelado = true
			break
		}
	}
	avisar(true)
	if len(res.Errores) > 50 {
		n := len(res.Errores)
		res.Errores = append(res.Errores[:50], fmt.Sprintf("… y %d errores más", n-50))
	}
	return res, nil
}

// AbrirEnExplorador abre la carpeta misma en el Explorador (si es un archivo, muestra su carpeta con él seleccionado).
func (a *App) AbrirEnExplorador(v string) error {
	_, _, abs, err := a.absoluta(v)
	if err != nil {
		return err
	}
	if dirExiste(abs) {
		return abrirCarpeta(abs)
	}
	return mostrarEnExplorador(abs)
}
