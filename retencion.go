package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Retención: cuánto tiempo deben quedarse los archivos en una carpeta. Se define en la ficha de la
// carpeta principal (campo "Retención de archivos") y, si hace falta, por subcarpeta (hoja Subcarpetas).
// Una subcarpeta sin valor hereda la de la carpeta que la contiene; "No depurar" corta la herencia.
// Un archivo está vencido cuando su fecha de modificación es más antigua que ese tiempo.

const (
	campoRetencion = "Retención de archivos"
	sinDepurar     = "No depurar"
)

var reRetencion = regexp.MustCompile(`^(\d+)\s*([a-z]+)$`)

// leerRetencion: cantidad y unidad ("d", "s", "m", "a"). ok=false si no es un tiempo válido.
func leerRetencion(v string) (int, string, bool) {
	s := strings.ToLower(sinTildes(strings.TrimSpace(v)))
	m := reRetencion.FindStringSubmatch(s)
	if m == nil {
		return 0, "", false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 || n > 1200 {
		return 0, "", false
	}
	switch m[2] {
	case "d", "dia", "dias":
		return n, "d", true
	case "s", "sem", "semana", "semanas":
		return n, "s", true
	case "m", "mes", "meses":
		return n, "m", true
	case "a", "ano", "anos", "año", "años":
		return n, "a", true
	}
	return 0, "", false
}

// normalizarRetencion deja el texto como se guarda en el Excel: "6 meses", "1 año", "No depurar" o "".
func normalizarRetencion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if normal(v) == normal(sinDepurar) {
		return sinDepurar
	}
	n, u, ok := leerRetencion(v)
	if !ok {
		return v // texto libre: se guarda tal cual (no se usa para depurar)
	}
	nombres := map[string][2]string{"d": {"día", "días"}, "s": {"semana", "semanas"}, "m": {"mes", "meses"}, "a": {"año", "años"}}
	if n == 1 {
		return "1 " + nombres[u][0]
	}
	return fmt.Sprintf("%d %s", n, nombres[u][1])
}

// venceEl: fecha en que un archivo modificado en mod se pasa del tiempo de retención.
func venceEl(retencion string, mod time.Time) (time.Time, bool) {
	n, u, ok := leerRetencion(retencion)
	if !ok {
		return time.Time{}, false
	}
	switch u {
	case "d":
		return mod.AddDate(0, 0, n), true
	case "s":
		return mod.AddDate(0, 0, 7*n), true
	case "m":
		return mod.AddDate(0, n, 0), true
	default:
		return mod.AddDate(n, 0, 0), true
	}
}

// reglas de retención de una raíz (rutas en minúsculas dentro de la raíz).
type reglas struct {
	fichas map[string]string // carpeta principal -> retención
	subs   map[string]string // subcarpeta -> retención propia
}

func reglasDe(b *baseDatos) reglas {
	rg := reglas{fichas: map[string]string{}, subs: map[string]string{}}
	if b == nil {
		return rg
	}
	for _, r := range b.registros {
		if !esPrincipal(r.Ruta) {
			continue
		}
		for k, v := range r.Valores {
			if normal(k) == normal(campoRetencion) && strings.TrimSpace(v) != "" {
				if _, ya := rg.fichas[strings.ToLower(r.Ruta)]; !ya {
					rg.fichas[strings.ToLower(r.Ruta)] = strings.TrimSpace(v)
				}
			}
		}
	}
	for _, d := range b.subs {
		if strings.TrimSpace(d.Retencion) != "" {
			rg.subs[strings.ToLower(d.Ruta)] = strings.TrimSpace(d.Retencion)
		}
	}
	return rg
}

func (rg reglas) vacias() bool { return len(rg.fichas) == 0 && len(rg.subs) == 0 }

// efectiva: la retención que aplica en la carpeta rel y la carpeta donde se definió.
func (rg reglas) efectiva(rel string) (string, string) {
	for cur := rel; cur != ""; cur = padreDe(cur) {
		k := strings.ToLower(cur)
		if esPrincipal(cur) {
			if v := rg.fichas[k]; v != "" {
				return v, cur
			}
			return "", ""
		}
		if v := rg.subs[k]; v != "" {
			return v, cur
		}
	}
	return "", ""
}

// hayReglaDebajo: alguna subcarpeta dentro de rel tiene retención propia.
func (rg reglas) hayReglaDebajo(rel string) bool {
	p := strings.ToLower(rel) + "/"
	for k := range rg.subs {
		if rel == "" || strings.HasPrefix(k, p) {
			return true
		}
	}
	for k := range rg.fichas {
		if rel == "" || strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

// ---------------- Revisión de vencidos ----------------

// ArchivoVencido: un archivo que se pasó del tiempo de retención.
type ArchivoVencido struct {
	Ruta       string `json:"ruta"`
	Nombre     string `json:"nombre"`
	Ext        string `json:"ext"`
	Carpeta    string `json:"carpeta"` // ruta virtual de la carpeta que lo contiene
	Tamano     int64  `json:"tamano"`
	Modificado string `json:"modificado"`
	Creado     string `json:"creado"`
	Retencion  string `json:"retencion"`
	Origen     string `json:"origen"` // carpeta donde se definió la retención
	Vencio     string `json:"vencio"` // fecha en que se venció
	Dias       int    `json:"dias"`   // días que lleva vencido
}

// ResultadoVencidos de una revisión.
type ResultadoVencidos struct {
	Archivos  []ArchivoVencido `json:"archivos"`
	Revisados int              `json:"revisados"`
	Reglas    int              `json:"reglas"`    // carpetas con retención definida (en el alcance)
	SinExcel  []string         `json:"sinExcel"`  // raíces sin Excel de fichas
	Cortado   bool             `json:"cortado"`   // demasiados vencidos: se muestran los primeros
	Cancelado bool             `json:"cancelado"` // el usuario detuvo la revisión
	Segundos  float64          `json:"segundos"`
}

// CancelarVencidos detiene la revisión en curso.
func (a *App) CancelarVencidos() { a.cancelarVenc.Store(true) }

// Vencidos revisa los archivos de la carpeta v ("" = todas las raíces) contra la retención de las fichas.
func (a *App) Vencidos(v string) (*ResultadoVencidos, error) {
	a.cancelarVenc.Store(false)
	inicio := time.Now()
	ahora := time.Now()
	res := &ResultadoVencidos{Archivos: []ArchivoVencido{}, SinExcel: []string{}}
	type alcance struct {
		ri  *raizInfo
		rel string
	}
	var alcances []alcance
	if v == "" {
		for _, ri := range a.lista() {
			alcances = append(alcances, alcance{ri, ""})
		}
	} else {
		ri, rel, err := a.resolver(v)
		if err != nil {
			return nil, err
		}
		alcances = append(alcances, alcance{ri, rel})
	}
	const maximo = 20000
	fin := errors.New("fin")
	for _, al := range alcances {
		b, err := a.base(al.ri)
		if err != nil || b == nil {
			res.SinExcel = append(res.SinExcel, al.ri.Nombre)
			continue
		}
		rg := reglasDe(b)
		for k := range rg.fichas {
			if dentroDe(k, strings.ToLower(al.rel)) {
				res.Reglas++
			}
		}
		for k := range rg.subs {
			if dentroDe(k, strings.ToLower(al.rel)) {
				res.Reglas++
			}
		}
		if rg.vacias() {
			continue
		}
		absInicio, _ := rutaSegura(al.ri.Ruta, al.rel)
		cache := map[string][2]string{} // carpeta -> {retención, origen}
		regla := func(dir string) (string, string) {
			if r, ok := cache[dir]; ok {
				return r[0], r[1]
			}
			t, o := rg.efectiva(dir)
			cache[dir] = [2]string{t, o}
			return t, o
		}
		err = filepath.WalkDir(absInicio, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if a.cancelarVenc.Load() {
				res.Cancelado = true
				return fin
			}
			if p == absInicio {
				return nil
			}
			relP, _ := filepath.Rel(al.ri.Ruta, p)
			relP = filepath.ToSlash(relP)
			if omitir(d.Name(), !strings.Contains(relP, "/")) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				// Sin regla aquí ni más abajo: no hace falta entrar.
				if t, _ := regla(relP); t == "" && !rg.hayReglaDebajo(relP) {
					return filepath.SkipDir
				}
				return nil
			}
			res.Revisados++
			if res.Revisados%500 == 0 {
				a.emitir("vencidos:progreso", map[string]int{"revisados": res.Revisados, "encontrados": len(res.Archivos)})
			}
			dir := padreDe(relP)
			t, origen := regla(dir)
			if t == "" || t == sinDepurar {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			vence, ok := venceEl(t, info.ModTime())
			if !ok || !vence.Before(ahora) {
				return nil
			}
			res.Archivos = append(res.Archivos, ArchivoVencido{
				Ruta: unir(al.ri.ID, relP), Nombre: d.Name(), Ext: strings.ToLower(filepath.Ext(d.Name())),
				Carpeta: unir(al.ri.ID, dir), Tamano: info.Size(),
				Modificado: info.ModTime().Format(time.RFC3339), Creado: fechaCreacion(info),
				Retencion: t, Origen: unir(al.ri.ID, origen), Vencio: vence.Format(time.RFC3339),
				Dias: int(ahora.Sub(vence).Hours() / 24),
			})
			if len(res.Archivos) >= maximo {
				res.Cortado = true
				return fin
			}
			return nil
		})
		if res.Cancelado || res.Cortado {
			break
		}
	}
	sort.SliceStable(res.Archivos, func(i, j int) bool {
		if res.Archivos[i].Origen != res.Archivos[j].Origen {
			return strings.ToLower(res.Archivos[i].Origen) < strings.ToLower(res.Archivos[j].Origen)
		}
		return res.Archivos[i].Dias > res.Archivos[j].Dias
	})
	res.Segundos = time.Since(inicio).Seconds()
	return res, nil
}

// ResultadoDepurar: lo que se movió a la papelera.
type ResultadoDepurar struct {
	Movidas []string `json:"movidas"` // rutas virtuales que se movieron
	Movidos int      `json:"movidos"`
	Tamano  int64    `json:"tamano"`
	Errores []string `json:"errores"`
}

// Depurar mueve los archivos indicados a la papelera de su raíz
// (_papelera_organizador\<fecha> depuracion\<ruta>) y lo anota en el Historial de cada Excel.
func (a *App) Depurar(rutas []string) (*ResultadoDepurar, error) {
	a.op.Lock()
	defer a.op.Unlock()
	defer a.olvidarTamanos()
	res := &ResultadoDepurar{Errores: []string{}, Movidas: []string{}}
	sello := time.Now().Format("2006-01-02 150405") + " depuracion"
	type hecho struct{ rel, destino string }
	porRaiz := map[*raizInfo][]hecho{}
	for _, v := range rutas {
		ri, rel, abs, err := a.absoluta(v)
		if err != nil || rel == "" {
			res.Errores = append(res.Errores, v+": ruta no válida")
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			res.Errores = append(res.Errores, path.Base(rel)+": ya no existe")
			continue
		}
		if info.IsDir() {
			res.Errores = append(res.Errores, path.Base(rel)+": es una carpeta (solo se depuran archivos)")
			continue
		}
		destino := libre(ri.Ruta, unir(unir(carpetaPapelera, sello), padreDe(rel)), path.Base(rel))
		absDest, _ := rutaSegura(ri.Ruta, destino)
		if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
			res.Errores = append(res.Errores, path.Base(rel)+": "+err.Error())
			continue
		}
		if err := moverEntre(abs, absDest); err != nil {
			res.Errores = append(res.Errores, path.Base(rel)+": no se pudo mover (¿está abierto?)")
			continue
		}
		ocultar(filepath.Join(ri.Ruta, carpetaPapelera))
		res.Movidos++
		res.Movidas = append(res.Movidas, v)
		res.Tamano += info.Size()
		porRaiz[ri] = append(porRaiz[ri], hecho{rel, destino})
	}
	// Historial: una fila por archivo, guardando una sola vez por Excel.
	for ri, hechos := range porRaiz {
		b, err := a.baseEscritura(ri)
		if err != nil {
			res.Errores = append(res.Errores, fmt.Sprintf("%s: se movieron los archivos, pero no se pudo anotar en el Historial (%v)", ri.Nombre, err))
			continue
		}
		if f, _, err := b.abrirEscritura(); err == nil {
			for _, h := range hechos {
				historial(f, "Depurar vencido", rutaAExcel(h.rel), "Movido a "+rutaAExcel(h.destino))
			}
			guardarLibro(f)
			f.Close()
		}
		a.olvidarBase(ri)
	}
	return res, nil
}
