package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entrada es una carpeta o un archivo dentro de la raíz.
type Entrada struct {
	Nombre     string `json:"nombre"`
	Ruta       string `json:"ruta"` // relativa a la raíz, con "/"
	EsCarpeta  bool   `json:"esCarpeta"`
	Ext        string `json:"ext"` // ".pdf", ".xlsx"… (minúsculas)
	Tamano     int64  `json:"tamano"`
	Modificado string `json:"modificado"` // RFC3339
	Ficha      bool   `json:"ficha"`      // la carpeta tiene ficha en el Excel
}

// DetalleCarpeta: lo que se muestra al elegir una carpeta.
type DetalleCarpeta struct {
	Carpeta     Entrada     `json:"carpeta"`
	Entradas    []Entrada   `json:"entradas"`
	Subcarpetas int         `json:"subcarpetas"`
	Archivos    int         `json:"archivos"`
	TamanoTotal int64       `json:"tamanoTotal"` // solo archivos directos
	Registros   []*Registro `json:"registros"`
}

// DetalleArchivo: lo que se muestra al elegir un archivo.
type DetalleArchivo struct {
	Archivo Entrada `json:"archivo"`
	Creado  string  `json:"creado"`
}

// omitir: lo que no se muestra (temporales de Office, sistema, la papelera de la app y el Excel de fichas).
func omitir(nombre string, enRaiz bool) bool {
	l := strings.ToLower(nombre)
	if strings.HasPrefix(l, "~$") || strings.HasPrefix(l, ".") || l == "desktop.ini" || l == "thumbs.db" {
		return true
	}
	return enRaiz && (strings.EqualFold(nombre, carpetaPapelera) || strings.EqualFold(nombre, nombreBase))
}

func entradaDe(rel string, info os.FileInfo) Entrada {
	e := Entrada{
		Nombre: info.Name(), Ruta: rel, EsCarpeta: info.IsDir(),
		Modificado: info.ModTime().Format(time.RFC3339),
	}
	if !e.EsCarpeta {
		e.Ext = strings.ToLower(filepath.Ext(info.Name()))
		e.Tamano = info.Size()
	}
	return e
}

// leerDir lista una carpeta: primero carpetas, luego archivos, en orden alfabético.
func leerDir(raiz, rel string, conFicha map[string]bool) ([]Entrada, error) {
	abs, ok := rutaSegura(raiz, rel)
	if !ok {
		return nil, errors.New("ruta no válida")
	}
	var ents []os.DirEntry
	if err := conTiempo(20*time.Second, func() error {
		var err error
		ents, err = os.ReadDir(abs)
		return err
	}); err != nil {
		return nil, err
	}
	out := make([]Entrada, 0, len(ents))
	for _, d := range ents {
		if omitir(d.Name(), rel == "") {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		e := entradaDe(unir(rel, d.Name()), info)
		if e.EsCarpeta && conFicha != nil {
			e.Ficha = conFicha[strings.ToLower(e.Ruta)]
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EsCarpeta != out[j].EsCarpeta {
			return out[i].EsCarpeta
		}
		return strings.ToLower(sinTildes(out[i].Nombre)) < strings.ToLower(sinTildes(out[j].Nombre))
	})
	return out, nil
}

// Listar devuelve el contenido de una carpeta (para el árbol, que carga por demanda).
func (a *App) Listar(rel string) ([]Entrada, error) {
	raiz := a.raizActual()
	if raiz == "" {
		return nil, errors.New("elija la carpeta raíz")
	}
	return leerDir(raiz, rel, a.rutasConFicha())
}

// Carpeta devuelve el detalle de una carpeta con sus fichas.
func (a *App) Carpeta(rel string) (*DetalleCarpeta, error) {
	raiz := a.raizActual()
	abs, ok := rutaSegura(raiz, rel)
	if raiz == "" || !ok {
		return nil, errors.New("ruta no válida")
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("la carpeta ya no existe: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("no es una carpeta")
	}
	conFicha := a.rutasConFicha()
	ents, err := leerDir(raiz, rel, conFicha)
	if err != nil {
		return nil, err
	}
	rel, _ = limpiarRel(rel)
	d := &DetalleCarpeta{Carpeta: entradaDe(rel, info), Entradas: ents, Registros: []*Registro{}}
	if rel == "" {
		d.Carpeta.Nombre = filepath.Base(raiz)
	}
	d.Carpeta.Ficha = conFicha[strings.ToLower(rel)]
	for _, e := range ents {
		if e.EsCarpeta {
			d.Subcarpetas++
		} else {
			d.Archivos++
			d.TamanoTotal += e.Tamano
		}
	}
	if b, _ := a.base(); b != nil {
		for _, r := range b.registros {
			if strings.EqualFold(r.Ruta, rel) {
				d.Registros = append(d.Registros, r)
			}
		}
	}
	return d, nil
}

// Archivo devuelve los datos de un archivo.
func (a *App) Archivo(rel string) (*DetalleArchivo, error) {
	raiz := a.raizActual()
	abs, ok := rutaSegura(raiz, rel)
	if raiz == "" || !ok {
		return nil, errors.New("ruta no válida")
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("el archivo ya no existe: %w", err)
	}
	rel, _ = limpiarRel(rel)
	return &DetalleArchivo{Archivo: entradaDe(rel, info), Creado: fechaCreacion(info)}, nil
}

// ---------------- Búsqueda ----------------

// ResultadoBusqueda: coincidencias por nombre (carpetas y archivos).
type ResultadoBusqueda struct {
	Entradas []Entrada `json:"entradas"`
	Cortada  bool      `json:"cortada"` // se llegó al máximo o al tiempo límite
}

// Buscar recorre la raíz buscando el texto en los nombres (sin tildes ni mayúsculas).
func (a *App) Buscar(texto string) (*ResultadoBusqueda, error) {
	raiz := a.raizActual()
	q := strings.ToLower(sinTildes(strings.TrimSpace(texto)))
	res := &ResultadoBusqueda{Entradas: []Entrada{}}
	if raiz == "" || len([]rune(q)) < 2 {
		return res, nil
	}
	const maximo = 300
	limite := time.Now().Add(25 * time.Second)
	conFicha := a.rutasConFicha()
	errFin := errors.New("fin")
	err := filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if p == raiz {
			return nil
		}
		rel, _ := filepath.Rel(raiz, p)
		rel = filepath.ToSlash(rel)
		if omitir(d.Name(), !strings.Contains(rel, "/")) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(sinTildes(d.Name())), q) {
			if info, err := d.Info(); err == nil {
				e := entradaDe(rel, info)
				e.Ficha = e.EsCarpeta && conFicha[strings.ToLower(rel)]
				res.Entradas = append(res.Entradas, e)
			}
		}
		if len(res.Entradas) >= maximo || time.Now().After(limite) {
			res.Cortada = true
			return errFin
		}
		return nil
	})
	if err != nil && err != errFin {
		return nil, err
	}
	return res, nil
}

// ---------------- Vista gráfica ----------------

// NodoGrafo es un punto del grafo.
type NodoGrafo struct {
	ID        string `json:"id"` // ruta relativa ("" = raíz del grafo)
	Nombre    string `json:"nombre"`
	EsCarpeta bool   `json:"esCarpeta"`
	Ext       string `json:"ext"`
	Nivel     int    `json:"nivel"`
	Hijos     int    `json:"hijos"` // entradas directas (para el tamaño del punto)
	Ficha     bool   `json:"ficha"`
	Tamano    int64  `json:"tamano"`
}

// EnlaceGrafo une una carpeta con lo que contiene.
type EnlaceGrafo struct {
	Origen  string `json:"origen"`
	Destino string `json:"destino"`
}

// Grafo es la vista gráfica de una carpeta.
type Grafo struct {
	Nodos   []NodoGrafo   `json:"nodos"`
	Enlaces []EnlaceGrafo `json:"enlaces"`
	Cortado bool          `json:"cortado"` // se llegó al máximo de nodos
	Maximo  int           `json:"maximo"`
}

// Grafo arma el grafo desde una carpeta hasta cierta profundidad (máx. 2500 nodos).
func (a *App) Grafo(rel string, profundidad int, conArchivos bool) (*Grafo, error) {
	raiz := a.raizActual()
	rel, ok := limpiarRel(rel)
	if raiz == "" || !ok {
		return nil, errors.New("ruta no válida")
	}
	if profundidad < 1 {
		profundidad = 1
	}
	if profundidad > 8 {
		profundidad = 8
	}
	const maximo = 2500
	conFicha := a.rutasConFicha()
	g := &Grafo{Nodos: []NodoGrafo{}, Enlaces: []EnlaceGrafo{}, Maximo: maximo}
	nombre := path.Base(rel)
	if rel == "" {
		nombre = filepath.Base(raiz)
	}
	g.Nodos = append(g.Nodos, NodoGrafo{ID: rel, Nombre: nombre, EsCarpeta: true, Ficha: conFicha[strings.ToLower(rel)]})
	idx := map[string]int{rel: 0}
	limite := time.Now().Add(40 * time.Second)

	// Recorrido por niveles: primero lo más cercano, así si se corta queda lo importante.
	cola := []string{rel}
	for nivel := 1; nivel <= profundidad && len(cola) > 0 && !g.Cortado; nivel++ {
		var siguiente []string
		for _, dir := range cola {
			ents, err := leerDir(raiz, dir, conFicha)
			if err != nil {
				continue
			}
			g.Nodos[idx[dir]].Hijos = len(ents)
			for _, e := range ents {
				if !e.EsCarpeta && !conArchivos {
					continue
				}
				if len(g.Nodos) >= maximo || time.Now().After(limite) {
					g.Cortado = true
					break
				}
				idx[e.Ruta] = len(g.Nodos)
				g.Nodos = append(g.Nodos, NodoGrafo{
					ID: e.Ruta, Nombre: e.Nombre, EsCarpeta: e.EsCarpeta, Ext: e.Ext,
					Nivel: nivel, Ficha: e.Ficha, Tamano: e.Tamano,
				})
				g.Enlaces = append(g.Enlaces, EnlaceGrafo{Origen: dir, Destino: e.Ruta})
				if e.EsCarpeta {
					siguiente = append(siguiente, e.Ruta)
				}
			}
			if g.Cortado {
				break
			}
		}
		cola = siguiente
	}
	return g, nil
}
