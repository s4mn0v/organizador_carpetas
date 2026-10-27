package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App expone las funciones al frontend.
type App struct {
	ctx  context.Context
	mu   sync.Mutex // protege raiz y la caché del Excel
	op   sync.Mutex // una operación de archivos a la vez
	raiz string

	cache      *baseDatos // Excel leído (se relee si cambia en disco)
	cacheFecha time.Time
	cacheRuta  string
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if c := leerConfig(); c.Raiz != "" && dirExiste(c.Raiz) {
		a.raiz = c.Raiz
	}
}

func (a *App) raizActual() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.raiz
}

func (a *App) rutaBase() string { return filepath.Join(a.raizActual(), nombreBase) }

// base devuelve el Excel de fichas (con caché por fecha de modificación). nil si no existe.
func (a *App) base() (*baseDatos, error) {
	ruta := a.rutaBase()
	info, err := os.Stat(ruta)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if a.cache != nil && a.cacheRuta == ruta && a.cacheFecha.Equal(info.ModTime()) {
		b := a.cache
		a.mu.Unlock()
		return b, nil
	}
	a.mu.Unlock()
	b, err := cargarBase(ruta)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.cache, a.cacheRuta, a.cacheFecha = b, ruta, info.ModTime()
	a.mu.Unlock()
	return b, nil
}

// baseEscritura: el Excel listo para modificar (sin caché y comprobando que no esté abierto).
func (a *App) baseEscritura() (*baseDatos, error) {
	a.olvidarBase()
	b, err := a.base()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no existe %s en la raíz: créelo desde el Resumen", nombreBase)
		}
		return nil, fmt.Errorf("no se pudo abrir %s: %w", nombreBase, err)
	}
	if err := puedeEscribir(b.ruta); err != nil {
		return nil, err
	}
	return b, nil
}

func (a *App) olvidarBase() {
	a.mu.Lock()
	a.cache = nil
	a.mu.Unlock()
}

// rutasConFicha: carpetas principales (en minúsculas) que tienen ficha.
func (a *App) rutasConFicha() map[string]bool {
	m := map[string]bool{}
	if b, _ := a.base(); b != nil {
		for _, r := range b.registros {
			if esPrincipal(r.Ruta) {
				m[strings.ToLower(r.Ruta)] = true
			}
		}
	}
	return m
}

// ---------------- Estado general ----------------

// Estado: lo que necesita la pantalla de inicio.
type Estado struct {
	Raiz         string      `json:"raiz"`
	NombreRaiz   string      `json:"nombreRaiz"`
	Recientes    []string    `json:"recientes"`
	Base         string      `json:"base"`
	BaseExiste   bool        `json:"baseExiste"`
	BaseError    string      `json:"baseError"`
	Campos       []Campo     `json:"campos"`
	Registros    int         `json:"registros"`
	SinCarpeta   []*Registro `json:"sinCarpeta"`  // fichas cuya carpeta ya no existe
	NoPrincipal  []*Registro `json:"noPrincipal"` // fichas en la raíz o en subcarpetas (solo van en carpetas principales)
	PapeleraHay  bool        `json:"papeleraHay"`
	NombreExcel  string      `json:"nombreExcel"`
	NombrePapele string      `json:"nombrePapelera"`
}

func (a *App) Estado() (*Estado, error) {
	raiz := a.raizActual()
	st := &Estado{Raiz: raiz, Recientes: leerConfig().Recientes, SinCarpeta: []*Registro{}, NoPrincipal: []*Registro{}, Campos: []Campo{},
		NombreExcel: nombreBase, NombrePapele: carpetaPapelera}
	if raiz == "" {
		return st, nil
	}
	if err := conTiempo(10*time.Second, func() error {
		if !dirExiste(raiz) {
			return fmt.Errorf("no se encuentra la carpeta %s", raiz)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	st.NombreRaiz = filepath.Base(raiz)
	st.Base = a.rutaBase()
	st.PapeleraHay = dirExiste(filepath.Join(raiz, carpetaPapelera))
	b, err := a.base()
	switch {
	case err == nil:
		st.BaseExiste = true
		st.Campos = b.campos
		st.Registros = len(b.registros)
		for _, r := range b.registros {
			if abs, ok := rutaSegura(raiz, r.Ruta); !ok || !dirExiste(abs) {
				st.SinCarpeta = append(st.SinCarpeta, r)
			} else if !esPrincipal(r.Ruta) {
				st.NoPrincipal = append(st.NoPrincipal, r)
			}
		}
	case os.IsNotExist(err):
	default:
		st.BaseExiste = true
		st.BaseError = "No se pudo leer el Excel de fichas: " + err.Error()
	}
	return st, nil
}

// ElegirRaiz pide la carpeta raíz y la recuerda.
func (a *App) ElegirRaiz() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Carpeta raíz a organizar",
		DefaultDirectory: a.raizActual(),
	})
	if err != nil || dir == "" {
		return a.raizActual(), err
	}
	return dir, a.UsarRaiz(dir)
}

// UsarRaiz cambia a una raíz ya conocida (lista de recientes).
func (a *App) UsarRaiz(dir string) error {
	if !dirExiste(dir) {
		return fmt.Errorf("no se encuentra la carpeta %s", dir)
	}
	a.mu.Lock()
	a.raiz = filepath.Clean(dir)
	a.cache = nil
	a.mu.Unlock()
	return recordarRaiz(filepath.Clean(dir))
}

// ---------------- Excel de fichas ----------------

// CrearBase crea el Excel de fichas en la raíz.
func (a *App) CrearBase() error {
	if a.raizActual() == "" {
		return errors.New("elija la carpeta raíz")
	}
	if _, err := os.Stat(a.rutaBase()); err == nil {
		return errors.New("el Excel de fichas ya existe")
	}
	a.olvidarBase()
	return crearPlantilla(a.rutaBase())
}

// AbrirBase abre el Excel de fichas.
func (a *App) AbrirBase() error { return abrirConSistema(a.rutaBase()) }

// esPrincipal: carpeta que está directo en la raíz (las únicas que llevan ficha técnica).
func esPrincipal(rel string) bool { return rel != "" && !strings.Contains(rel, "/") }

// principalDe: la carpeta principal a la que pertenece rel ("" si rel es la raíz).
func principalDe(rel string) string {
	if i := strings.Index(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return rel
}

// GuardarRegistro crea o actualiza la ficha de una carpeta principal junto con
// las descripciones de sus subcarpetas.
func (a *App) GuardarRegistro(r Registro, subs []DescripcionSub) (*Registro, error) {
	a.op.Lock()
	defer a.op.Unlock()
	r.Ruta, _ = limpiarRel(r.Ruta)
	if !esPrincipal(r.Ruta) && r.ID == 0 {
		return nil, errors.New("las fichas técnicas solo se crean en las carpetas principales (las que están en la raíz)")
	}
	b, err := a.baseEscritura()
	if err != nil {
		return nil, err
	}
	defer a.olvidarBase()
	if subs == nil {
		subs = []DescripcionSub{}
	}
	return b.guardar(&r, subs)
}

// EliminarRegistro borra una ficha (queda copia en el Historial).
func (a *App) EliminarRegistro(id int) error {
	a.op.Lock()
	defer a.op.Unlock()
	b, err := a.baseEscritura()
	if err != nil {
		return err
	}
	defer a.olvidarBase()
	return b.eliminar([]int{id}, "Eliminar ficha")
}

// ---------------- Abrir con Windows ----------------

func (a *App) AbrirArchivo(rel string) error {
	abs, ok := rutaSegura(a.raizActual(), rel)
	if !ok || a.raizActual() == "" {
		return errors.New("ruta no válida")
	}
	return abrirConSistema(abs)
}

func (a *App) MostrarEnCarpeta(rel string) error {
	abs, ok := rutaSegura(a.raizActual(), rel)
	if !ok || a.raizActual() == "" {
		return errors.New("ruta no válida")
	}
	return mostrarEnExplorador(abs)
}

// AbrirPapelera abre la papelera de la app (lo eliminado se guarda ahí).
func (a *App) AbrirPapelera() error {
	p := filepath.Join(a.raizActual(), carpetaPapelera)
	if !dirExiste(p) {
		return errors.New("la papelera está vacía")
	}
	return abrirConSistema(p)
}
