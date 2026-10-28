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

// Entrada es una carpeta o un archivo (ruta virtual: "<raíz>/<ruta>").
type Entrada struct {
	Nombre     string `json:"nombre"`
	Ruta       string `json:"ruta"`
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
	EsInicio    bool        `json:"esInicio"` // "" = todas las raíces
	EsRaiz      bool        `json:"esRaiz"`

	// Fichas: solo las carpetas principales (directo en la raíz) tienen ficha.
	EsPrincipal    bool              `json:"esPrincipal"`
	Arbol          []SubcarpetaFicha `json:"arbol"`        // principal: todas sus subcarpetas, para describirlas
	ArbolCortado   bool              `json:"arbolCortado"` // demasiadas subcarpetas: se muestran las primeras
	Principal      string            `json:"principal"`    // subcarpeta: su carpeta principal
	PrincipalFicha bool              `json:"principalFicha"`
	Descripcion    string            `json:"descripcion"` // subcarpeta: su explicación en la ficha
	Retencion      string            `json:"retencion"`   // tiempo que deben quedarse los archivos aquí ("" = sin regla)
	RetencionDe    string            `json:"retencionDe"` // carpeta donde se definió esa regla
}

// SubcarpetaFicha es una subcarpeta dentro de la ficha de su carpeta principal.
type SubcarpetaFicha struct {
	Ruta        string `json:"ruta"`
	Nombre      string `json:"nombre"`
	Nivel       int    `json:"nivel"` // 1 = hija directa de la principal
	Descripcion string `json:"descripcion"`
	Retencion   string `json:"retencion"` // propia ("" = hereda)
	Heredada    string `json:"heredada"`  // la que aplica si no tiene propia
}

// DetalleArchivo: lo que se muestra al elegir un archivo.
type DetalleArchivo struct {
	Archivo     Entrada `json:"archivo"`
	Creado      string  `json:"creado"`
	Ubicacion   string  `json:"ubicacion"` // ruta completa en Windows
	Retencion   string  `json:"retencion"`
	RetencionDe string  `json:"retencionDe"`
	Vence       string  `json:"vence"` // fecha en que se pasa del tiempo de retención (RFC3339)
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

// leerDir lista una carpeta de una raíz: primero carpetas, luego archivos, en orden alfabético.
// Las rutas que devuelve son relativas a la raíz.
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

// virtuales pasa las rutas de las entradas a rutas virtuales de la raíz.
func virtuales(ri *raizInfo, ents []Entrada) []Entrada {
	for i := range ents {
		ents[i].Ruta = unir(ri.ID, ents[i].Ruta)
	}
	return ents
}

// entradasRaices: las raíces abiertas como carpetas (el contenido del Inicio).
func (a *App) entradasRaices() []Entrada {
	out := []Entrada{}
	for _, ri := range a.lista() {
		e := Entrada{Nombre: ri.Nombre, Ruta: ri.ID, EsCarpeta: true}
		if info, err := os.Stat(ri.Ruta); err == nil {
			e.Modificado = info.ModTime().Format(time.RFC3339)
		}
		out = append(out, e)
	}
	return out
}

// Listar devuelve el contenido de una carpeta (para el árbol, que carga por demanda). "" = las raíces.
func (a *App) Listar(v string) ([]Entrada, error) {
	if v == "" {
		return a.entradasRaices(), nil
	}
	ri, rel, err := a.resolver(v)
	if err != nil {
		return nil, err
	}
	ents, err := leerDir(ri.Ruta, rel, a.rutasConFicha(ri))
	if err != nil {
		return nil, err
	}
	return virtuales(ri, ents), nil
}

// Carpeta devuelve el detalle de una carpeta con sus fichas.
func (a *App) Carpeta(v string) (*DetalleCarpeta, error) {
	if v == "" {
		ents := a.entradasRaices()
		return &DetalleCarpeta{Carpeta: Entrada{Nombre: "Inicio", EsCarpeta: true}, Entradas: ents,
			Subcarpetas: len(ents), Registros: []*Registro{}, Arbol: []SubcarpetaFicha{}, EsInicio: true}, nil
	}
	ri, rel, abs, err := a.absoluta(v)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("la carpeta ya no existe: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("no es una carpeta")
	}
	conFicha := a.rutasConFicha(ri)
	ents, err := leerDir(ri.Ruta, rel, conFicha)
	if err != nil {
		return nil, err
	}
	d := &DetalleCarpeta{Carpeta: entradaDe(unir(ri.ID, rel), info), Registros: []*Registro{}, EsRaiz: rel == ""}
	if rel == "" {
		d.Carpeta.Nombre = ri.Nombre
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
	d.Entradas = virtuales(ri, ents)
	d.Arbol = []SubcarpetaFicha{}
	b, _ := a.base(ri)
	rg := reglasDe(b)
	if rel != "" {
		d.Retencion, d.RetencionDe = rg.efectiva(rel)
		if d.RetencionDe != "" {
			d.RetencionDe = unir(ri.ID, d.RetencionDe)
		}
	}
	switch {
	case esPrincipal(rel):
		d.EsPrincipal = true
		if b != nil {
			for _, r := range b.registros {
				if strings.EqualFold(r.Ruta, rel) {
					d.Registros = append(d.Registros, regV(ri, r))
				}
			}
		}
		d.Arbol, d.ArbolCortado = arbolSubcarpetas(ri, rel, b, rg)
	case rel != "":
		d.Principal = unir(ri.ID, principalDe(rel))
		d.PrincipalFicha = conFicha[strings.ToLower(principalDe(rel))]
		if b != nil {
			d.Descripcion = b.descripcionDe(rel)
		}
	}
	return d, nil
}

// arbolSubcarpetas lista todas las subcarpetas de una carpeta principal (en orden de árbol),
// con su descripción y retención. Máximo 600 o 20 segundos.
func arbolSubcarpetas(ri *raizInfo, rel string, b *baseDatos, rg reglas) ([]SubcarpetaFicha, bool) {
	out := []SubcarpetaFicha{}
	limite := time.Now().Add(20 * time.Second)
	cortado := false
	var recorrer func(dir string, nivel int)
	recorrer = func(dir string, nivel int) {
		if cortado || nivel > 10 {
			return
		}
		ents, err := leerDir(ri.Ruta, dir, nil)
		if err != nil {
			return
		}
		for _, e := range ents {
			if !e.EsCarpeta {
				continue
			}
			if len(out) >= 600 || time.Now().After(limite) {
				cortado = true
				return
			}
			s := SubcarpetaFicha{Ruta: unir(ri.ID, e.Ruta), Nombre: e.Nombre, Nivel: nivel}
			if b != nil {
				s.Descripcion = b.descripcionDe(e.Ruta)
				s.Retencion = rg.subs[strings.ToLower(e.Ruta)]
			}
			s.Heredada, _ = rg.efectiva(dir)
			out = append(out, s)
			recorrer(e.Ruta, nivel+1)
		}
	}
	recorrer(rel, 1)
	return out, cortado
}

// Archivo devuelve los datos de un archivo.
func (a *App) Archivo(v string) (*DetalleArchivo, error) {
	ri, rel, abs, err := a.absoluta(v)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("el archivo ya no existe: %w", err)
	}
	d := &DetalleArchivo{Archivo: entradaDe(unir(ri.ID, rel), info), Creado: fechaCreacion(info), Ubicacion: abs}
	if b, _ := a.base(ri); b != nil {
		rg := reglasDe(b)
		d.Retencion, d.RetencionDe = rg.efectiva(padreDe(rel))
		if d.RetencionDe != "" {
			d.RetencionDe = unir(ri.ID, d.RetencionDe)
		}
		if vence, ok := venceEl(d.Retencion, info.ModTime()); ok {
			d.Vence = vence.Format(time.RFC3339)
		}
	}
	return d, nil
}

// ---------------- Vista gráfica ----------------

// NodoGrafo es un punto del grafo.
type NodoGrafo struct {
	ID        string `json:"id"` // ruta virtual ("" = Inicio)
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

// Grafo arma el grafo desde una carpeta ("" = todas las raíces) hasta cierta profundidad (máx. 2500 nodos).
func (a *App) Grafo(v string, profundidad int, conArchivos bool) (*Grafo, error) {
	v, ok := limpiarRel(v)
	if !ok {
		return nil, errors.New("ruta no válida")
	}
	if profundidad < 1 {
		profundidad = 1
	}
	if profundidad > 8 {
		profundidad = 8
	}
	const maximo = 2500
	g := &Grafo{Nodos: []NodoGrafo{}, Enlaces: []EnlaceGrafo{}, Maximo: maximo}
	fichas := map[string]map[string]bool{} // id raíz -> rutas con ficha
	conFicha := func(ri *raizInfo) map[string]bool {
		if m, ok := fichas[ri.ID]; ok {
			return m
		}
		fichas[ri.ID] = a.rutasConFicha(ri)
		return fichas[ri.ID]
	}
	// hijos de un nodo (ruta virtual)
	leer := func(dir string) ([]Entrada, error) {
		if dir == "" {
			return a.entradasRaices(), nil
		}
		ri, rel, err := a.resolver(dir)
		if err != nil {
			return nil, err
		}
		ents, err := leerDir(ri.Ruta, rel, conFicha(ri))
		if err != nil {
			return nil, err
		}
		return virtuales(ri, ents), nil
	}

	nombre := "Inicio"
	ficha := false
	if v != "" {
		ri, rel, err := a.resolver(v)
		if err != nil {
			return nil, err
		}
		nombre = path.Base(rel)
		if rel == "" {
			nombre = ri.Nombre
		}
		ficha = conFicha(ri)[strings.ToLower(rel)]
	}
	g.Nodos = append(g.Nodos, NodoGrafo{ID: v, Nombre: nombre, EsCarpeta: true, Ficha: ficha})
	idx := map[string]int{v: 0}
	limite := time.Now().Add(40 * time.Second)

	// Recorrido por niveles: primero lo más cercano, así si se corta queda lo importante.
	cola := []string{v}
	for nivel := 1; nivel <= profundidad && len(cola) > 0 && !g.Cortado; nivel++ {
		var siguiente []string
		for _, dir := range cola {
			ents, err := leer(dir)
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
