package main

// Índice pregrabado de cada raíz: toda la estructura de carpetas y archivos en memoria y en
// %APPDATA%\organizador_carpetas\indices\<raíz>.json.gz. Sirve para que Claude ubique rutas
// al instante (herramienta "indice") en vez de recorrer el disco (Z: es lento) en cada consulta.
//
//   - Al abrir una raíz: se carga el índice guardado (instantáneo) y se rehace completo en segundo plano.
//   - En cada cambio de la app (crear, mover, renombrar, eliminar, depurar, agregar archivos — también
//     los que hace Claude): solo se vuelve a leer la parte que cambió. Gancho: olvidarTamanos.
//   - Recargar (F5): se rehace completo (por si algo cambió desde el Explorador).

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxNodosIndice = 400000 // tope por raíz (memoria); si se pasa, el índice queda "parcial"

// nodoIndice: una carpeta o archivo. En carpetas, Tam y Arch suman todo lo que tienen debajo.
type nodoIndice struct {
	Rel  string `json:"r"`
	Dir  bool   `json:"d,omitempty"`
	Tam  int64  `json:"t,omitempty"`
	Arch int    `json:"a,omitempty"`
	Mod  int64  `json:"m,omitempty"` // unix
}

type indiceRaiz struct {
	Ruta     string       `json:"ruta"`
	Generado time.Time    `json:"generado"`
	Parcial  bool         `json:"parcial"`
	Nodos    []nodoIndice `json:"nodos"` // ordenados por Rel
}

// trabajo de una raíz: cola de rutas por releer ("" = todo) y un solo trabajador a la vez.
type estadoIndice struct {
	mu     sync.Mutex
	ind    *indiceRaiz
	cola   []string
	activo bool
	leidos int // elementos leídos en el recorrido completo en curso (progreso)

	// Vigilancia en vivo (ReadDirectoryChangesW)
	vigilando bool     // los avisos de Windows están llegando: el índice está al día
	dudoso    bool     // se perdieron avisos o se cayó Z:: hasta re-escanear no se confía en el índice
	completo  bool     // el trabajo en curso es un recorrido completo
	detener   func()   // apaga la vigilancia
	pend      []string // cambios recibidos esperando la pausa de 400 ms
	tPend     *time.Timer
}

var indices = struct {
	mu sync.Mutex
	m  map[string]*estadoIndice // clave: ruta de la raíz en minúsculas
}{m: map[string]*estadoIndice{}}

// EstadoIndice: para mostrar en el panel.
type EstadoIndice struct {
	Raiz      string `json:"raiz"`
	Listo     bool   `json:"listo"`
	Cargando  bool   `json:"cargando"`
	Carpetas  int    `json:"carpetas"`
	Archivos  int    `json:"archivos"`
	Leidos    int    `json:"leidos"` // progreso del recorrido en curso
	Vigilando bool   `json:"vigilando"`
	Dudoso    bool   `json:"dudoso"`
	Generado  string `json:"generado"`
	Parcial   bool   `json:"parcial"`
}

func estadoIdx(ruta string) *estadoIndice {
	indices.mu.Lock()
	defer indices.mu.Unlock()
	k := strings.ToLower(ruta)
	e := indices.m[k]
	if e == nil {
		e = &estadoIndice{}
		indices.m[k] = e
	}
	return e
}

// EstadosIndice: cómo va el índice de cada raíz abierta.
func (a *App) EstadosIndice() []EstadoIndice {
	var l []EstadoIndice
	for _, ri := range a.lista() {
		l = append(l, a.estadoDe(ri))
	}
	return l
}

func (a *App) estadoDe(ri *raizInfo) EstadoIndice {
	e := estadoIdx(ri.Ruta)
	e.mu.Lock()
	defer e.mu.Unlock()
	st := EstadoIndice{Raiz: ri.ID, Cargando: e.activo, Leidos: e.leidos, Vigilando: e.vigilando, Dudoso: e.dudoso}
	if e.ind != nil {
		st.Listo, st.Parcial = true, e.ind.Parcial
		st.Generado = e.ind.Generado.Format(time.RFC3339)
		for _, n := range e.ind.Nodos {
			if n.Dir {
				st.Carpetas++
			} else {
				st.Archivos++
			}
		}
	}
	return st
}

// ---------------- Ganchos ----------------

// indiceCambio: algo cambió en estas rutas virtuales (mismo formato que olvidarTamanos).
// "" = se abrió una raíz: carga/arma los índices que falten.
func (a *App) indiceCambio(rutas ...string) {
	a.sincronizarVigilancia()
	for _, v := range rutas {
		v, _ = limpiarRel(v)
		if v == "" {
			for _, ri := range a.lista() {
				e := estadoIdx(ri.Ruta)
				e.mu.Lock()
				sinIndice := e.ind == nil
				rehacer := (e.ind == nil || e.dudoso) && !e.activo
				e.mu.Unlock()
				if sinIndice {
					a.cargarIndiceGuardado(ri) // sirve de punto de partida, pero queda "dudoso" hasta re-escanear
				}
				if rehacer {
					a.pedirIndice(ri, "")
				}
				a.sincronizarVigilancia()
			}
			continue
		}
		ri, rel, err := a.resolver(v)
		if err != nil || rel == "" {
			continue // raíz cerrada, o es la raíz misma (no cambia su estructura)
		}
		a.pedirIndice(ri, rel)
	}
}

// reindexarTodo: botón Recargar.
func (a *App) reindexarTodo() {
	for _, ri := range a.lista() {
		a.pedirIndice(ri, "")
	}
}

// pedirIndice encola una ruta para releer ("" = toda la raíz) y arranca el trabajador si hace falta.
func (a *App) pedirIndice(ri *raizInfo, rel string) {
	e := estadoIdx(ri.Ruta)
	e.mu.Lock()
	defer e.mu.Unlock()
	if rel == "" || e.ind == nil {
		e.cola = []string{""}
	} else if len(e.cola) == 0 || e.cola[0] != "" {
		e.cola = append(e.cola, rel)
	}
	if !e.activo {
		e.activo = true
		go a.trabajarIndice(ri, e)
	}
}

func (a *App) trabajarIndice(ri *raizInfo, e *estadoIndice) {
	a.emitir("indice:estado", a.estadoDe(ri))
	for {
		e.mu.Lock()
		cola := e.cola
		e.cola = nil
		if len(cola) == 0 {
			e.activo = false
			e.mu.Unlock()
			a.emitir("indice:estado", a.estadoDe(ri))
			return
		}
		base := e.ind
		e.completo = cola[0] == "" || base == nil
		e.mu.Unlock()

		var nuevo *indiceRaiz
		t0 := time.Now()
		if cola[0] == "" || base == nil {
			defer func() {}()
			ultimo := time.Now()
			nuevo = recorrerRaiz(ri.Ruta, func(n int) {
				e.mu.Lock()
				e.leidos = n
				e.mu.Unlock()
				if time.Since(ultimo) > 700*time.Millisecond { // progreso al panel sin saturarlo
					ultimo = time.Now()
					a.emitir("indice:estado", a.estadoDe(ri))
				}
			})
		} else {
			nuevo = actualizarIndice(base, ri.Ruta, cola)
		}
		e.mu.Lock()
		e.ind, e.leidos = nuevo, 0
		if e.completo {
			e.dudoso = false // recorrido completo: vuelve a ser confiable
			a.logClaude("índice %s: %s elementos en %.1f s (%d hilos)", ri.ID, miles(len(nuevo.Nodos)), time.Since(t0).Seconds(), hilosIndice)
		}
		e.completo = false
		e.mu.Unlock()
		_ = guardarIndice(nuevo)
	}
}

// ---------------- Recorrido ----------------

// recorrerRaiz arma el índice completo de una raíz.
func recorrerRaiz(raiz string, progreso func(int)) *indiceRaiz {
	ind := &indiceRaiz{Ruta: raiz, Generado: time.Now()}
	ind.Nodos, ind.Parcial = recorrer(raiz, "", maxNodosIndice, progreso)
	sumarCarpetas(ind)
	return ind
}

// hilosIndice: carpetas que se leen a la vez. Suficiente para esconder la latencia de Z:
// sin saturar el servidor de archivos.
const hilosIndice = 8

var errNoVigilancia = errors.New("vigilancia no disponible en este sistema")

// entradaDir: lo que devuelve leerDirRapido (API nativa en Windows).
type entradaDir struct {
	Nombre string
	Dir    bool
	Tam    int64
	Mod    int64
}

// recorrer lee la carpeta rel (relativa a raiz) y todo lo de adentro, varias carpetas a la vez.
// Incluye rel misma si no es "". Si rel es un archivo, devuelve solo ese archivo.
func recorrer(raiz, rel string, tope int, progreso func(int)) ([]nodoIndice, bool) {
	inicio, ok := rutaSegura(raiz, rel)
	if !ok {
		return nil, false
	}
	var out []nodoIndice
	if rel != "" {
		info, err := os.Stat(inicio)
		if err != nil {
			return nil, false // ya no existe
		}
		if omitir(info.Name(), !strings.Contains(rel, "/")) {
			return nil, false
		}
		n := nodoIndice{Rel: rel, Dir: info.IsDir(), Mod: info.ModTime().Unix()}
		if !n.Dir {
			n.Tam = info.Size()
			return []nodoIndice{n}, false
		}
		out = append(out, n)
	}

	// Cola de carpetas pendientes + 8 trabajadores. pendientes = en cola + en lectura.
	var (
		mu         sync.Mutex
		cond       = sync.NewCond(&mu)
		cola       = []string{rel}
		pendientes = 1
		parcial    bool
	)
	trabajador := func() {
		for {
			mu.Lock()
			for len(cola) == 0 && pendientes > 0 {
				cond.Wait()
			}
			if len(cola) == 0 {
				mu.Unlock()
				return
			}
			dir := cola[len(cola)-1]
			cola = cola[:len(cola)-1]
			cortar := parcial
			mu.Unlock()

			var ents []entradaDir
			if !cortar {
				abs, _ := rutaSegura(raiz, dir)
				ents, _ = leerDirRapido(abs) // sin permiso o desaparecida: se salta
			}

			mu.Lock()
			for _, e := range ents {
				if omitir(e.Nombre, dir == "") {
					continue
				}
				if len(out) >= tope {
					parcial = true
					break
				}
				r := unir(dir, e.Nombre)
				out = append(out, nodoIndice{Rel: r, Dir: e.Dir, Tam: e.Tam, Mod: e.Mod})
				if e.Dir {
					cola = append(cola, r)
					pendientes++
				}
				if progreso != nil && len(out)%500 == 0 {
					progreso(len(out))
				}
			}
			pendientes--
			cond.Broadcast()
			mu.Unlock()
		}
	}
	var wg sync.WaitGroup
	for k := 0; k < hilosIndice; k++ {
		wg.Add(1)
		go func() { defer wg.Done(); trabajador() }()
	}
	wg.Wait()
	return out, parcial
}

// actualizarIndice: copia del índice con las rutas indicadas vueltas a leer (solo esa parte del disco).
func actualizarIndice(base *indiceRaiz, raiz string, rels []string) *indiceRaiz {
	rels = sinAnidadas(rels)
	quitar := func(r string) bool {
		for _, x := range rels {
			if strings.EqualFold(r, x) || dentroDe(r, x) {
				return true
			}
		}
		return false
	}
	nuevo := &indiceRaiz{Ruta: raiz, Generado: base.Generado, Parcial: base.Parcial}
	for _, n := range base.Nodos {
		if !quitar(n.Rel) {
			n.Tam, n.Arch = tamArchivo(n), 0 // los totales de carpetas se recalculan
			nuevo.Nodos = append(nuevo.Nodos, n)
		}
	}
	for _, r := range rels {
		ns, parcial := recorrer(raiz, r, maxNodosIndice-len(nuevo.Nodos), nil)
		nuevo.Nodos = append(nuevo.Nodos, ns...)
		nuevo.Parcial = nuevo.Parcial || parcial
	}
	sumarCarpetas(nuevo)
	return nuevo
}

func tamArchivo(n nodoIndice) int64 {
	if n.Dir {
		return 0
	}
	return n.Tam
}

// sinAnidadas: quita repetidas y las que están dentro de otra de la lista.
func sinAnidadas(rels []string) []string {
	sort.Slice(rels, func(i, j int) bool { return len(rels[i]) < len(rels[j]) })
	var out []string
	for _, r := range rels {
		dentro := false
		for _, o := range out {
			if strings.EqualFold(r, o) || dentroDe(r, o) {
				dentro = true
				break
			}
		}
		if !dentro {
			out = append(out, r)
		}
	}
	return out
}

// sumarCarpetas ordena los nodos y calcula, para cada carpeta, archivos y tamaño de todo lo de adentro.
func sumarCarpetas(ind *indiceRaiz) {
	sort.Slice(ind.Nodos, func(i, j int) bool { return ind.Nodos[i].Rel < ind.Nodos[j].Rel })
	pos := make(map[string]int, len(ind.Nodos))
	for i, n := range ind.Nodos {
		if n.Dir {
			pos[n.Rel] = i
			ind.Nodos[i].Tam, ind.Nodos[i].Arch = 0, 0
		}
	}
	for _, n := range ind.Nodos {
		if n.Dir {
			continue
		}
		for p := path.Dir(n.Rel); p != "." && p != ""; p = path.Dir(p) {
			if i, ok := pos[p]; ok {
				ind.Nodos[i].Tam += n.Tam
				ind.Nodos[i].Arch++
			}
		}
	}
}

// ---------------- Disco ----------------

func rutaIndice(raiz string) string {
	h := sha1.Sum([]byte(strings.ToLower(raiz)))
	return filepath.Join(filepath.Dir(rutaConfig()), "indices", hex.EncodeToString(h[:6])+".json.gz")
}

func guardarIndice(ind *indiceRaiz) error {
	p := rutaIndice(ind.Ruta)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(f)
	err = json.NewEncoder(zw).Encode(ind)
	if e := zw.Close(); err == nil {
		err = e
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}

// cargarIndiceGuardado: usa el último índice del disco mientras se arma el nuevo.
func (a *App) cargarIndiceGuardado(ri *raizInfo) {
	f, err := os.Open(rutaIndice(ri.Ruta))
	if err != nil {
		return
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return
	}
	var ind indiceRaiz
	if json.NewDecoder(zr).Decode(&ind) != nil || !strings.EqualFold(ind.Ruta, ri.Ruta) {
		return
	}
	e := estadoIdx(ri.Ruta)
	e.mu.Lock()
	if e.ind == nil {
		e.ind, e.dudoso = &ind, true // del disco: pudo cambiar mientras la app estaba cerrada
	}
	e.mu.Unlock()
}

// ---------------- Consulta (herramienta MCP "indice") ----------------

// consultarIndice devuelve texto compacto (pocos tokens):
//   - sin filtro: árbol desde v hasta cierta profundidad, con archivos y tamaño de cada carpeta.
//   - con filtro: las rutas cuyo nombre coincide (misma regla que Buscar), mejores primero.
func (a *App) consultarIndice(v string, profundidad int, conArchivos bool, filtro string) (string, error) {
	v, ok := limpiarRel(v)
	if !ok {
		return "", fmt.Errorf("ruta no válida")
	}
	var raices []*raizInfo
	relBase := ""
	if v == "" {
		raices = a.lista()
	} else {
		ri, rel, err := a.resolver(v)
		if err != nil {
			return "", err
		}
		raices, relBase = []*raizInfo{ri}, rel
	}
	if len(raices) == 0 {
		return "No hay carpetas raíz abiertas en la app.", nil
	}
	const maxLineas = 400
	var b strings.Builder
	lineas, resto := 0, 0
	for _, ri := range raices {
		e := estadoIdx(ri.Ruta)
		e.mu.Lock()
		ind, activo := e.ind, e.activo
		e.mu.Unlock()
		fmt.Fprintf(&b, "## %s = %s", ri.ID, ri.Ruta)
		if ind == nil {
			b.WriteString(" — índice en construcción, intente en unos segundos (o use 'buscar')\n")
			continue
		}
		e.mu.Lock()
		dudoso := e.dudoso
		e.mu.Unlock()
		if dudoso {
			b.WriteString(" — índice re-escaneándose (puede no estar al día): espere a que termine\n")
			continue
		}
		fmt.Fprintf(&b, " (índice %s%s%s)\n", haceTexto(ind.Generado),
			map[bool]string{true: ", actualizándose", false: ""}[activo],
			map[bool]string{true: ", PARCIAL: raíz muy grande", false: ""}[ind.Parcial])

		if filtro != "" {
			n, r := listarCoincidencias(&b, ri, ind, relBase, filtro, maxLineas-lineas)
			lineas += n
			resto += r
			continue
		}
		prefijo := ""
		nivelBase := 0
		if relBase != "" {
			prefijo = strings.ToLower(relBase) + "/"
			nivelBase = strings.Count(relBase, "/") + 1
		}
		for _, n := range ind.Nodos {
			if prefijo != "" && !strings.HasPrefix(strings.ToLower(n.Rel), prefijo) {
				continue
			}
			nivel := strings.Count(n.Rel, "/") - nivelBase // 0 = hijo directo de v
			if nivel >= profundidad || (!n.Dir && !conArchivos) {
				continue
			}
			if lineas >= maxLineas {
				resto++
				continue
			}
			nombre := path.Base(n.Rel)
			if n.Dir {
				fmt.Fprintf(&b, "%s%s/ [%d arch, %s]\n", strings.Repeat("  ", nivel), nombre, n.Arch, tamanoTexto(n.Tam))
			} else {
				fmt.Fprintf(&b, "%s%s (%s, %s)\n", strings.Repeat("  ", nivel), nombre, tamanoTexto(n.Tam), time.Unix(n.Mod, 0).Format("2006-01-02"))
			}
			lineas++
		}
	}
	if resto > 0 {
		fmt.Fprintf(&b, "… %d más sin mostrar: pida una ruta más específica, menos profundidad o use filtro.\n", resto)
	}
	return b.String(), nil
}

// listarCoincidencias: rutas virtuales que coinciden con el filtro (máx. 60).
func listarCoincidencias(b *strings.Builder, ri *raizInfo, ind *indiceRaiz, relBase, filtro string, cupo int) (int, int) {
	qs := consulta(filtro)
	if len(qs) == 0 {
		return 0, 0
	}
	type hallado struct {
		n  nodoIndice
		k  int
		pt float64
	}
	var hs []hallado
	for _, n := range ind.Nodos {
		if relBase != "" && !dentroDe(n.Rel, relBase) {
			continue
		}
		if k, pt := puntuar(qs, path.Base(n.Rel)); k > 0 {
			hs = append(hs, hallado{n, k, pt})
		}
	}
	sort.SliceStable(hs, func(i, j int) bool {
		if hs[i].k != hs[j].k {
			return hs[i].k > hs[j].k
		}
		return hs[i].pt > hs[j].pt
	})
	tope := min(60, cupo)
	escritas := 0
	for _, h := range hs {
		if escritas >= tope {
			break
		}
		v := unir(ri.ID, h.n.Rel)
		if h.n.Dir {
			fmt.Fprintf(b, "%s/ [%d arch, %s]\n", v, h.n.Arch, tamanoTexto(h.n.Tam))
		} else {
			fmt.Fprintf(b, "%s (%s, %s)\n", v, tamanoTexto(h.n.Tam), time.Unix(h.n.Mod, 0).Format("2006-01-02"))
		}
		escritas++
	}
	if len(hs) == 0 {
		b.WriteString("(sin coincidencias)\n")
	}
	return escritas, len(hs) - escritas
}

func tamanoTexto(n int64) string {
	const u = 1024
	if n < u {
		return fmt.Sprintf("%d B", n)
	}
	f, i := float64(n), 0
	for f >= u && i < 4 {
		f /= u
		i++
	}
	return fmt.Sprintf("%.1f %s", f, []string{"B", "KB", "MB", "GB", "TB"}[i])
}

func haceTexto(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "recién actualizado"
	case d < time.Hour:
		return fmt.Sprintf("de hace %d min", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("de hace %d h", int(d.Hours()))
	default:
		return "del " + t.Format("2006-01-02")
	}
}

// ---------------- Vigilancia en vivo ----------------

// sincronizarVigilancia: vigila las raíces abiertas que tienen índice y apaga las que se cerraron.
func (a *App) sincronizarVigilancia() {
	abiertas := map[string]*raizInfo{}
	for _, ri := range a.lista() {
		abiertas[strings.ToLower(ri.Ruta)] = ri
	}
	indices.mu.Lock()
	todas := make(map[string]*estadoIndice, len(indices.m))
	for k, e := range indices.m {
		todas[k] = e
	}
	indices.mu.Unlock()
	for k, e := range todas {
		ri := abiertas[k]
		e.mu.Lock()
		switch {
		case ri == nil && e.detener != nil: // raíz cerrada: sin avisos, el índice deja de ser confiable
			e.detener()
			e.detener, e.vigilando, e.dudoso = nil, false, true
		case ri != nil && e.detener == nil:
			det, err := vigilar(ri.Ruta, func(rels []string) { a.alCambiarDisco(ri, e, rels) },
				func(motivo string) { a.alFallarVigilancia(ri, e, motivo) })
			if err == nil {
				e.detener, e.vigilando = det, true
			} else {
				e.detener = func() {} // no reintentar en cada cambio
				a.logClaude("índice %s: sin vigilancia en vivo (%v); la búsqueda y los tamaños leerán el disco", ri.ID, err)
			}
		}
		e.mu.Unlock()
	}
}

// detenerVigilancia: al cerrar la app.
func detenerVigilancia() {
	indices.mu.Lock()
	defer indices.mu.Unlock()
	for _, e := range indices.m {
		e.mu.Lock()
		if e.detener != nil {
			e.detener()
		}
		e.detener, e.vigilando = nil, false
		e.mu.Unlock()
	}
}

// alCambiarDisco: llegan avisos de Windows. Se juntan 400 ms (una copia de 500 archivos = una sola actualización).
func (a *App) alCambiarDisco(ri *raizInfo, e *estadoIndice, rels []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range rels {
		r, ok := limpiarRel(r)
		if !ok || r == "" {
			continue
		}
		if p := strings.SplitN(r, "/", 2)[0]; strings.EqualFold(p, carpetaPapelera) || strings.EqualFold(r, nombreBase) {
			continue
		}
		e.pend = append(e.pend, r)
	}
	if len(e.pend) == 0 {
		return
	}
	if e.tPend != nil {
		e.tPend.Stop()
	}
	e.tPend = time.AfterFunc(400*time.Millisecond, func() { a.aplicarCambiosDisco(ri, e) })
}

func (a *App) aplicarCambiosDisco(ri *raizInfo, e *estadoIndice) {
	e.mu.Lock()
	pend := e.pend
	e.pend, e.tPend = nil, nil
	ind := e.ind
	e.mu.Unlock()
	var conocidas map[string]bool // carpetas que ya están en el índice
	if ind != nil {
		conocidas = make(map[string]bool)
		for _, n := range ind.Nodos {
			if n.Dir {
				conocidas[n.Rel] = true
			}
		}
	}
	// Una carpeta que ya está en el índice y sigue existiendo solo "se modificó" porque cambió algo
	// adentro: esos cambios llegan aparte. Se descarta ANTES de agrupar (si no, taparía a sus hijas).
	var utiles []string
	vistos := map[string]bool{}
	for _, r := range pend {
		if vistos[r] {
			continue
		}
		vistos[r] = true
		if conocidas[r] {
			if abs, ok := rutaSegura(ri.Ruta, r); ok && dirExiste(abs) {
				continue
			}
		}
		utiles = append(utiles, r)
	}
	pend = sinAnidadas(utiles)
	n := 0
	for _, r := range pend {
		a.pedirIndice(ri, r)
		n++
	}
	if n > 0 {
		a.olvidarSoloTamanos(ri, pend)
		a.emitir("indice:cambio", ri.ID)
	}
}

// alFallarVigilancia: se perdieron avisos o se cayó la conexión: no se confía en el índice hasta re-escanear.
func (a *App) alFallarVigilancia(ri *raizInfo, e *estadoIndice, motivo string) {
	e.mu.Lock()
	e.dudoso = true
	if strings.HasPrefix(motivo, "conexión") {
		e.vigilando = false
	} else if motivo == "reconectado" {
		e.vigilando = true
	}
	e.mu.Unlock()
	a.logClaude("índice %s: %s → se vuelve a recorrer", ri.ID, motivo)
	if motivo != "conexión" && !strings.HasPrefix(motivo, "conexión") {
		a.pedirIndice(ri, "")
	}
	a.emitir("indice:estado", a.estadoDe(ri))
}

// ---------------- Índice al día: búsqueda y tamaños de la app ----------------

// fresco devuelve el índice solo si está al día con el disco (vigilado, sin avisos perdidos y sin
// cambios pendientes). Si hay una actualización corta en curso, espera hasta 3 s.
func fresco(ri *raizInfo) *indiceRaiz {
	e := estadoIdx(ri.Ruta)
	limite := time.Now().Add(3 * time.Second)
	for {
		e.mu.Lock()
		ind, ok := e.ind, e.ind != nil && e.vigilando && !e.dudoso && !e.completo
		ocupado := e.activo || len(e.pend) > 0
		e.mu.Unlock()
		if !ok {
			return nil
		}
		if !ocupado {
			return ind
		}
		if time.Now().After(limite) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// buscarEnIndice: misma regla que Buscar, pero en memoria. ok=false si algún índice no está al día.
func (a *App) buscarEnIndice(qs []string, ris []*raizInfo, rels []string) (*ResultadoBusqueda, bool) {
	type hallado struct {
		e       Entrada
		n       int
		puntaje float64
	}
	var hs []hallado
	for i, ri := range ris {
		ind := fresco(ri)
		if ind == nil {
			return nil, false
		}
		conFicha := a.rutasConFicha(ri)
		for _, nd := range ind.Nodos {
			if rels[i] != "" && !dentroDe(nd.Rel, rels[i]) {
				continue
			}
			nombre := path.Base(nd.Rel)
			k, pt := puntuar(qs, nombre)
			if k == 0 {
				continue
			}
			e := Entrada{Nombre: nombre, Ruta: unir(ri.ID, nd.Rel), EsCarpeta: nd.Dir, Modificado: time.Unix(nd.Mod, 0).Format(time.RFC3339)}
			if !nd.Dir {
				e.Ext = strings.ToLower(filepath.Ext(nombre))
				e.Tamano = nd.Tam
			} else {
				e.Ficha = conFicha[strings.ToLower(nd.Rel)]
			}
			hs = append(hs, hallado{e, k, pt})
		}
	}
	sort.SliceStable(hs, func(i, j int) bool {
		x, y := hs[i], hs[j]
		if x.n != y.n {
			return x.n > y.n
		}
		if x.puntaje != y.puntaje {
			return x.puntaje > y.puntaje
		}
		return len(x.e.Nombre) < len(y.e.Nombre)
	})
	res := &ResultadoBusqueda{Entradas: []Entrada{}, Total: len(hs)}
	for i := 0; i < len(hs) && i < 300; i++ {
		res.Entradas = append(res.Entradas, hs[i].e)
	}
	return res, true
}

// tamanoDesdeIndice: total de una carpeta (o de todo) al instante. ok=false si algún índice no está al día.
func (a *App) tamanoDesdeIndice(v string) (TamanoTotal, bool) {
	t := TamanoTotal{Ruta: v, Hora: time.Now().Format(time.RFC3339)}
	var ris []*raizInfo
	rel := ""
	if v == "" {
		ris = a.lista()
	} else {
		ri, r, err := a.resolver(v)
		if err != nil {
			return t, false
		}
		ris, rel = []*raizInfo{ri}, r
	}
	if len(ris) == 0 {
		return t, false
	}
	for _, ri := range ris {
		ind := fresco(ri)
		if ind == nil {
			return t, false
		}
		prefijo := ""
		if rel != "" {
			prefijo = rel + "/"
		}
		// Los nodos están ordenados: los de adentro de rel son un bloque seguido.
		desde := sort.Search(len(ind.Nodos), func(i int) bool { return ind.Nodos[i].Rel >= prefijo })
		for i := desde; i < len(ind.Nodos) && strings.HasPrefix(ind.Nodos[i].Rel, prefijo); i++ {
			if n := ind.Nodos[i]; n.Dir {
				t.Carpetas++
			} else {
				t.Archivos++
				t.Tamano += n.Tam
			}
		}
	}
	return t, true
}

// olvidarSoloTamanos: cambios que llegaron del disco (no de la app): se borran los totales guardados.
func (a *App) olvidarSoloTamanos(ri *raizInfo, rels []string) {
	vs := make([]string, 0, len(rels))
	for _, r := range rels {
		vs = append(vs, unir(ri.ID, r))
	}
	a.tamMu.Lock()
	defer a.tamMu.Unlock()
	for _, r := range vs {
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

// ---------------- Medición ----------------

// MedirIndexacion compara (en el registro) el recorrido de antes (WalkDir, 1 hilo) con el nuevo
// (API nativa + 8 hilos) sobre una raíz. Corre en segundo plano; puede tardar en Z:.
func (a *App) MedirIndexacion(id string) error {
	ri, err := a.raizPorID(id)
	if err != nil {
		return err
	}
	go func() {
		a.logClaude("medición %s: empezando (primero el método anterior)…", ri.ID)
		t0 := time.Now()
		n := 0
		_ = filepath.WalkDir(ri.Ruta, func(p string, d os.DirEntry, err error) error {
			if err == nil && p != ri.Ruta {
				n++
				if _, e := d.Info(); e != nil {
					return nil
				}
			}
			return nil
		})
		antes := time.Since(t0)
		a.logClaude("medición %s: anterior (WalkDir, 1 hilo): %s elementos en %.1f s", ri.ID, miles(n), antes.Seconds())
		t1 := time.Now()
		ns, _ := recorrer(ri.Ruta, "", maxNodosIndice, nil)
		ahora := time.Since(t1)
		a.logClaude("medición %s: nuevo (nativo + %d hilos): %s elementos en %.1f s → %.1f× más rápido",
			ri.ID, hilosIndice, miles(len(ns)), ahora.Seconds(), antes.Seconds()/max(ahora.Seconds(), 0.001))
	}()
	return nil
}
