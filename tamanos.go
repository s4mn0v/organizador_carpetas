package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Tamaño total de una carpeta (con todas sus subcarpetas).
//
// - Cada resultado se guarda hasta que algo cambie en esa carpeta: crear, renombrar, mover, eliminar,
//   agregar archivos o depurar (se borran esa carpeta, sus carpetas superiores y las de adentro)
//   o hasta que el usuario pulse Recargar.
// - El cálculo sigue aunque el usuario cambie de carpeta: si vuelve, ve lo que lleva o el total.
// - Máximo 2 recorridos a la vez; el último que el usuario pidió va primero.
// - Mientras recorre avisa lo sumado (evento "tamano:progreso") y, si una subcarpeta ya está calculada,
//   suma su total sin volver a recorrerla.

// TamanoTotal: suma de todo lo que hay dentro de una carpeta, a cualquier profundidad.
type TamanoTotal struct {
	Ruta     string  `json:"ruta"`
	Tamano   int64   `json:"tamano"`
	Archivos int     `json:"archivos"`
	Carpetas int     `json:"carpetas"`
	Cargando bool    `json:"cargando"` // todavía se está sumando
	EnCola   bool    `json:"enCola"`   // esperando turno (hay otros cálculos en curso)
	Segundos float64 `json:"segundos"`
	Hora     string  `json:"hora"` // cuándo se terminó de calcular (RFC3339)
}

type trabajoTam struct {
	t         TamanoTotal
	listo     bool
	invalido  bool // algo cambió mientras se calculaba: el resultado no se guarda
	prioridad time.Time
	hecho     chan struct{}
}

const maxRecorridos = 2

// TamanoCarpeta devuelve el total de la carpeta v ("" = todas las raíces). Si ya se calculó, al instante.
func (a *App) TamanoCarpeta(v string) (*TamanoTotal, error) {
	v, ok := limpiarRel(v)
	if !ok {
		return nil, errors.New("ruta no válida")
	}
	if v != "" {
		if _, _, _, err := a.absoluta(v); err != nil {
			return nil, err
		}
	}
	a.tamMu.Lock()
	if a.tams == nil {
		a.tams = map[string]*trabajoTam{}
	}
	tr := a.tams[v]
	if tr != nil && tr.listo {
		t := tr.t
		a.tamMu.Unlock()
		return &t, nil
	}
	if tr == nil {
		tr = &trabajoTam{t: TamanoTotal{Ruta: v, Cargando: true, EnCola: true}, hecho: make(chan struct{})}
		a.tams[v] = tr
		a.tamCola = append(a.tamCola, tr)
	}
	tr.prioridad = time.Now() // lo último que pidió el usuario va primero
	actual := tr.t
	a.lanzarTamanos()
	a.tamMu.Unlock()

	a.emitir("tamano:progreso", actual) // lo que lleva (o "en cola") para mostrarlo ya
	<-tr.hecho
	a.tamMu.Lock()
	t := tr.t
	a.tamMu.Unlock()
	return &t, nil
}

// lanzarTamanos arranca los cálculos en cola mientras haya turno (con tamMu tomado).
func (a *App) lanzarTamanos() {
	for a.tamActivo < maxRecorridos && len(a.tamCola) > 0 {
		k := 0
		for i, tr := range a.tamCola {
			if tr.prioridad.After(a.tamCola[k].prioridad) {
				k = i
			}
		}
		tr := a.tamCola[k]
		a.tamCola = append(a.tamCola[:k], a.tamCola[k+1:]...)
		a.tamActivo++
		tr.t.EnCola = false
		go a.calcularTamano(tr)
	}
}

func (a *App) calcularTamano(tr *trabajoTam) {
	v := tr.t.Ruta
	inicio := time.Now()
	t := TamanoTotal{Ruta: v, Cargando: true}
	ultimo := time.Time{}
	avisar := func(forzar bool) {
		if !forzar && time.Since(ultimo) < 150*time.Millisecond {
			return
		}
		ultimo = time.Now()
		t.Segundos = time.Since(inicio).Seconds()
		a.tamMu.Lock()
		tr.t = t
		a.tamMu.Unlock()
		a.emitir("tamano:progreso", t)
	}
	avisar(true)

	// Carpetas a recorrer: la pedida o todas las raíces.
	type dir struct {
		ri  *raizInfo
		rel string
	}
	var dirs []dir
	if v == "" {
		for _, ri := range a.lista() {
			dirs = append(dirs, dir{ri, ""})
		}
	} else if ri, rel, err := a.resolver(v); err == nil {
		dirs = append(dirs, dir{ri, rel})
	}
	// ya: total guardado de una carpeta (ruta virtual), si está calculado.
	ya := func(vr string) (TamanoTotal, bool) {
		a.tamMu.Lock()
		defer a.tamMu.Unlock()
		if g := a.tams[vr]; g != nil && g.listo && g != tr {
			return g.t, true
		}
		return TamanoTotal{}, false
	}
	sumar := func(g TamanoTotal) {
		t.Tamano += g.Tamano
		t.Archivos += g.Archivos
		t.Carpetas += g.Carpetas
	}
	n := 0
	for _, d := range dirs {
		if v == "" {
			if g, ok := ya(d.ri.ID); ok {
				sumar(g)
				continue
			}
		}
		absIni, _ := rutaSegura(d.ri.Ruta, d.rel)
		filepath.WalkDir(absIni, func(p string, e os.DirEntry, err error) error {
			if err != nil || p == absIni {
				return nil
			}
			if n++; n%200 == 0 {
				avisar(false)
			}
			if omitir(e.Name(), filepath.Dir(p) == d.ri.Ruta) {
				if e.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if e.IsDir() {
				t.Carpetas++
				rel, _ := filepath.Rel(d.ri.Ruta, p)
				if g, ok := ya(unir(d.ri.ID, filepath.ToSlash(rel))); ok {
					sumar(g) // ya se sabe cuánto tiene: no se recorre otra vez
					return filepath.SkipDir
				}
				return nil
			}
			t.Archivos++
			if info, err := e.Info(); err == nil {
				t.Tamano += info.Size()
			}
			return nil
		})
	}
	t.Cargando = false
	t.Segundos = time.Since(inicio).Seconds()
	t.Hora = time.Now().Format(time.RFC3339)

	a.tamMu.Lock()
	tr.t = t
	tr.listo = !tr.invalido
	if tr.invalido && a.tams[v] == tr {
		delete(a.tams, v)
	}
	a.tamActivo--
	a.lanzarTamanos()
	a.tamMu.Unlock()
	close(tr.hecho)
	a.emitir("tamano:progreso", t)
}

// olvidarTamanos: algo cambió en estas rutas. Se borra el total de cada una, de sus carpetas
// superiores (hasta el Inicio) y de las que tenía adentro; se vuelven a calcular al verlas.
func (a *App) olvidarTamanos(rutas ...string) {
	a.tamMu.Lock()
	defer a.tamMu.Unlock()
	for _, r := range rutas {
		r, _ = limpiarRel(r)
		if r == "" { // solo el total del Inicio (p. ej. se abrió otra raíz)
			if tr := a.tams[""]; tr != nil {
				tr.invalido = true
				delete(a.tams, "")
			}
			continue
		}
		for k, tr := range a.tams {
			if k == "" || dentroDe(r, k) || dentroDe(k, r) {
				tr.invalido = true
				if tr.listo || a.tams[k] == tr {
					delete(a.tams, k)
				}
			}
		}
	}
}

// OlvidarTamanos (botón Recargar): borra todos los totales guardados.
func (a *App) OlvidarTamanos() {
	a.tamMu.Lock()
	defer a.tamMu.Unlock()
	for k, tr := range a.tams {
		tr.invalido = true
		delete(a.tams, k)
	}
}
