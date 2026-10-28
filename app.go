package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App expone las funciones al frontend.
//
// Se pueden abrir varias carpetas raíz a la vez; cada una tiene su propio Excel de fichas.
// Hacia el frontend todas las rutas son "virtuales": "<id de la raíz>/<ruta dentro de la raíz>"
// (p. ej. "r1/Informes/2025"); "" es el Inicio, que reúne todas las raíces.
type App struct {
	ctx    context.Context
	mu     sync.Mutex // protege raices y la caché de cada Excel
	op     sync.Mutex // una operación de archivos a la vez
	raices []*raizInfo
	sigID  int

	tamMu     sync.Mutex             // protege tams, tamCola y tamActivos
	tams      map[string]*trabajoTam // tamaño de cada carpeta: calculado o calculándose
	tamCola   []*trabajoTam          // esperando turno
	tamActivo int                    // recorridos en curso (máx. 2 a la vez)

	cancelarDup  atomic.Bool
	cancelarVenc atomic.Bool
}

func NewApp() *App { return &App{} }

// startup: la app arranca sin carpetas; el usuario las elige cada vez.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ---------------- Excel de cada raíz ----------------

func (ri *raizInfo) rutaBase() string { return filepath.Join(ri.Ruta, nombreBase) }

// base devuelve el Excel de fichas de la raíz (con caché por fecha de modificación). nil si no existe.
func (a *App) base(ri *raizInfo) (*baseDatos, error) {
	ruta := ri.rutaBase()
	info, err := os.Stat(ruta)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if ri.cache != nil && ri.cacheFecha.Equal(info.ModTime()) {
		b := ri.cache
		a.mu.Unlock()
		return b, nil
	}
	a.mu.Unlock()
	a.migrarSiHaceFalta(ri)
	if i2, err := os.Stat(ruta); err == nil {
		info = i2
	}
	b, err := cargarBase(ruta)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	ri.cache, ri.cacheFecha = b, info.ModTime()
	a.mu.Unlock()
	return b, nil
}

// migrarSiHaceFalta quita una sola vez del Excel las secciones retiradas de la ficha.
// Si el Excel está abierto, lo intenta la próxima vez (mientras tanto esas secciones no se muestran).
func (a *App) migrarSiHaceFalta(ri *raizInfo) {
	a.mu.Lock()
	hecho := ri.migrado
	a.mu.Unlock()
	if hecho {
		return
	}
	nombres, respaldo, err := migrarBase(ri.rutaBase(), ri.Ruta)
	if err != nil {
		return
	}
	a.mu.Lock()
	ri.migrado = true
	if len(nombres) > 0 {
		ri.aviso = fmt.Sprintf("%s: se quitaron del Excel las secciones Gestión documental y Notas (%d columnas). Respaldo en %s", ri.Nombre, len(nombres), respaldo)
	}
	a.mu.Unlock()
}

// baseEscritura: el Excel listo para modificar (sin caché y comprobando que no esté abierto).
func (a *App) baseEscritura(ri *raizInfo) (*baseDatos, error) {
	a.olvidarBase(ri)
	b, err := a.base(ri)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s no tiene %s: créelo desde su carpeta raíz", ri.Nombre, nombreBase)
		}
		return nil, fmt.Errorf("no se pudo abrir %s de %s: %w", nombreBase, ri.Nombre, err)
	}
	if err := puedeEscribir(b.ruta); err != nil {
		return nil, err
	}
	return b, nil
}

func (a *App) olvidarBase(ri *raizInfo) {
	a.mu.Lock()
	ri.cache = nil
	a.mu.Unlock()
}

// rutasConFicha: carpetas principales (rutas dentro de la raíz, en minúsculas) que tienen ficha.
func (a *App) rutasConFicha(ri *raizInfo) map[string]bool {
	m := map[string]bool{}
	if b, _ := a.base(ri); b != nil {
		for _, r := range b.registros {
			if esPrincipal(r.Ruta) {
				m[strings.ToLower(r.Ruta)] = true
			}
		}
	}
	return m
}

// regV: copia de la ficha con la ruta virtual (para el frontend).
func regV(ri *raizInfo, r *Registro) *Registro {
	c := *r
	c.Ruta = unir(ri.ID, r.Ruta)
	return &c
}

// ---------------- Estado general ----------------

// EstadoRaiz: lo que el frontend necesita de cada carpeta raíz abierta.
type EstadoRaiz struct {
	Raiz
	Accesible   bool        `json:"accesible"`
	Error       string      `json:"error"`
	Base        string      `json:"base"`
	BaseExiste  bool        `json:"baseExiste"`
	BaseError   string      `json:"baseError"`
	Campos      []Campo     `json:"campos"`
	Registros   int         `json:"registros"`
	SinCarpeta  []*Registro `json:"sinCarpeta"`  // fichas cuya carpeta ya no existe
	NoPrincipal []*Registro `json:"noPrincipal"` // fichas en la raíz o en subcarpetas (solo van en carpetas principales)
	PapeleraHay bool        `json:"papeleraHay"`
}

// Estado: lo que necesita la pantalla de inicio.
type Estado struct {
	Raices       []*EstadoRaiz `json:"raices"`
	Recientes    []string      `json:"recientes"`
	Paleta       []string      `json:"paleta"`
	NombreExcel  string        `json:"nombreExcel"`
	NombrePapele string        `json:"nombrePapelera"`
	Avisos       []string      `json:"avisos"` // mensajes para mostrar una vez
}

func (a *App) Estado() (*Estado, error) {
	st := &Estado{Raices: []*EstadoRaiz{}, Recientes: leerConfig().Recientes, Paleta: paleta,
		NombreExcel: nombreBase, NombrePapele: carpetaPapelera, Avisos: []string{}}
	if st.Recientes == nil {
		st.Recientes = []string{}
	}
	for _, ri := range a.lista() {
		er := a.estadoRaiz(ri)
		st.Raices = append(st.Raices, er)
		a.mu.Lock()
		if ri.aviso != "" {
			st.Avisos = append(st.Avisos, ri.aviso)
			ri.aviso = ""
		}
		a.mu.Unlock()
	}
	return st, nil
}

func (a *App) estadoRaiz(ri *raizInfo) *EstadoRaiz {
	er := &EstadoRaiz{Raiz: ri.Raiz, SinCarpeta: []*Registro{}, NoPrincipal: []*Registro{}, Campos: []Campo{}}
	if err := conTiempo(10*time.Second, func() error {
		if !dirExiste(ri.Ruta) {
			return fmt.Errorf("no se encuentra la carpeta %s", ri.Ruta)
		}
		return nil
	}); err != nil {
		er.Error = err.Error()
		return er
	}
	er.Accesible = true
	er.Base = ri.rutaBase()
	er.PapeleraHay = dirExiste(filepath.Join(ri.Ruta, carpetaPapelera))
	b, err := a.base(ri)
	switch {
	case err == nil:
		er.BaseExiste = true
		er.Campos = b.campos
		er.Registros = len(b.registros)
		for _, r := range b.registros {
			if abs, ok := rutaSegura(ri.Ruta, r.Ruta); !ok || !dirExiste(abs) {
				er.SinCarpeta = append(er.SinCarpeta, regV(ri, r))
			} else if !esPrincipal(r.Ruta) {
				er.NoPrincipal = append(er.NoPrincipal, regV(ri, r))
			}
		}
	case os.IsNotExist(err):
	default:
		er.BaseExiste = true
		er.BaseError = "No se pudo leer el Excel de fichas: " + err.Error()
	}
	return er
}

// ---------------- Excel de fichas ----------------

// CrearBase crea el Excel de fichas en una raíz.
func (a *App) CrearBase(id string) error {
	ri, err := a.raizPorID(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(ri.rutaBase()); err == nil {
		return errors.New("el Excel de fichas ya existe")
	}
	a.olvidarBase(ri)
	return crearPlantilla(ri.rutaBase())
}

// AbrirBase abre el Excel de fichas de una raíz.
func (a *App) AbrirBase(id string) error {
	ri, err := a.raizPorID(id)
	if err != nil {
		return err
	}
	return abrirConSistema(ri.rutaBase())
}

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
// las descripciones (y retenciones) de sus subcarpetas. Rutas virtuales.
func (a *App) GuardarRegistro(r Registro, subs []DescripcionSub) (*Registro, error) {
	a.op.Lock()
	defer a.op.Unlock()
	ri, rel, err := a.resolver(r.Ruta)
	if err != nil {
		return nil, err
	}
	r.Ruta = rel
	if !esPrincipal(r.Ruta) && r.ID == 0 {
		return nil, errors.New("las fichas técnicas solo se crean en las carpetas principales (las que están en la raíz)")
	}
	for i := range subs {
		_, sr, err := a.resolver(subs[i].Ruta)
		if err != nil {
			return nil, err
		}
		subs[i].Ruta = sr
		subs[i].Retencion = normalizarRetencion(subs[i].Retencion)
	}
	if r.Valores != nil {
		if v, ok := r.Valores[campoRetencion]; ok {
			r.Valores[campoRetencion] = normalizarRetencion(v)
		}
	}
	b, err := a.baseEscritura(ri)
	if err != nil {
		return nil, err
	}
	defer a.olvidarBase(ri)
	if subs == nil {
		subs = []DescripcionSub{}
	}
	g, err := b.guardar(&r, subs)
	if err != nil {
		return nil, err
	}
	return regV(ri, g), nil
}

// EliminarRegistro borra una ficha de una raíz (queda copia en el Historial).
func (a *App) EliminarRegistro(id string, ficha int) error {
	a.op.Lock()
	defer a.op.Unlock()
	ri, err := a.raizPorID(id)
	if err != nil {
		return err
	}
	b, err := a.baseEscritura(ri)
	if err != nil {
		return err
	}
	defer a.olvidarBase(ri)
	return b.eliminar([]int{ficha}, "Eliminar ficha")
}

// ---------------- Abrir con Windows ----------------

func (a *App) AbrirArchivo(v string) error {
	_, _, abs, err := a.absoluta(v)
	if err != nil {
		return err
	}
	return abrirConSistema(abs)
}

func (a *App) MostrarEnCarpeta(v string) error {
	_, _, abs, err := a.absoluta(v)
	if err != nil {
		return err
	}
	return mostrarEnExplorador(abs)
}

// AbrirPapelera abre la papelera de una raíz (lo eliminado se guarda ahí).
func (a *App) AbrirPapelera(id string) error {
	ri, err := a.raizPorID(id)
	if err != nil {
		return err
	}
	p := filepath.Join(ri.Ruta, carpetaPapelera)
	if !dirExiste(p) {
		return errors.New("la papelera está vacía")
	}
	return abrirConSistema(p)
}

// emitir manda un evento al frontend (progreso de tareas largas).
func (a *App) emitir(nombre string, datos interface{}) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, nombre, datos)
	}
}
