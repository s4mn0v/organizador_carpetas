package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Búsqueda por palabras: un nombre coincide si tiene AL MENOS UNA de las palabras escritas,
// en cualquier orden, sin importar tildes ni mayúsculas, y también si la palabra se parece
// (empieza igual, está dentro de otra o tiene 1–2 letras distintas: "mantenimeinto" ≈ "mantenimiento").
// Los resultados se ordenan: primero los que tienen más palabras, luego los más parecidos.
// La misma regla está en frontend/src/util.js (puntuar) para los buscadores de la pantalla.

// palabras separa un texto en palabras normalizadas ("Informe_Mtto2024.xlsx" -> informe, mtto, 2024, xlsx).
func palabras(s string) []string {
	s = strings.ToLower(sinTildes(s))
	var out []string
	var b strings.Builder
	tipo := 0 // 1 letra, 2 número
	cortar := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for _, r := range s {
		t := 0
		switch {
		case unicode.IsLetter(r):
			t = 1
		case unicode.IsDigit(r):
			t = 2
		}
		if t == 0 {
			cortar()
			tipo = 0
			continue
		}
		if t != tipo {
			cortar()
		}
		tipo = t
		b.WriteRune(r)
	}
	cortar()
	return out
}

// consulta: palabras de búsqueda sin repetir (se ignoran letras sueltas).
func consulta(q string) []string {
	var out []string
	visto := map[string]bool{}
	for _, p := range palabras(q) {
		if visto[p] || (len([]rune(p)) < 2 && !unicode.IsDigit([]rune(p)[0])) {
			continue
		}
		visto[p] = true
		out = append(out, p)
	}
	return out
}

func distancia(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			c := 1
			if a[i-1] == b[j-1] {
				c = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+c)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// parecido de una palabra buscada con una palabra del nombre (0 = nada, 1 = igual).
func parecido(q, t string) float64 {
	switch {
	case q == t:
		return 1
	case strings.HasPrefix(t, q):
		return 0.9
	case len(q) >= 3 && strings.Contains(t, q):
		return 0.75
	case len(t) >= 4 && strings.HasPrefix(q, t):
		return 0.6
	}
	rq := []rune(q)
	if len(rq) < 4 {
		return 0
	}
	tol := 1
	if len(rq) >= 8 {
		tol = 2
	}
	rt := []rune(t)
	if absInt(len(rt)-len(rq)) <= tol {
		if d := distancia(rq, rt); d <= tol {
			return 0.5 - 0.1*float64(d)
		}
	}
	// "mantenim" mal escrito al comienzo de una palabra larga
	if len(rt) > len(rq) {
		if d := distancia(rq, rt[:len(rq)]); d <= tol {
			return 0.4 - 0.1*float64(d)
		}
	}
	return 0
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// puntuar: cuántas palabras de la consulta tiene el nombre y qué tan parecido es.
func puntuar(qs []string, nombre string) (int, float64) {
	ts := palabras(nombre)
	if len(ts) == 0 {
		return 0, 0
	}
	n, total := 0, 0.0
	for _, q := range qs {
		mejor := 0.0
		for _, t := range ts {
			if p := parecido(q, t); p > mejor {
				mejor = p
				if p == 1 {
					break
				}
			}
		}
		if mejor > 0 {
			n++
			total += mejor
		}
	}
	if n == 0 {
		return 0, 0
	}
	if n == len(qs) && len(qs) > 1 {
		total += 0.5
	}
	if len(qs) > 1 && strings.Contains(strings.Join(ts, " "), strings.Join(qs, " ")) {
		total += 0.5 // las palabras están juntas y en el mismo orden
	}
	return n, total
}

// ResultadoBusqueda: coincidencias por nombre (carpetas y archivos).
type ResultadoBusqueda struct {
	Entradas []Entrada `json:"entradas"`
	Total    int       `json:"total"`   // coincidencias encontradas (se devuelven las mejores)
	Cortada  bool      `json:"cortada"` // se llegó al tiempo límite: puede haber más
}

// Buscar recorre todas las raíces (o solo la carpeta "dentro", si se da) buscando las palabras en los nombres.
func (a *App) Buscar(texto, dentro string) (*ResultadoBusqueda, error) {
	qs := consulta(texto)
	res := &ResultadoBusqueda{Entradas: []Entrada{}}
	if len(qs) == 0 {
		return res, nil
	}
	type alcance struct {
		ri  *raizInfo
		rel string
	}
	var alcances []alcance
	if dentro == "" {
		for _, ri := range a.lista() {
			alcances = append(alcances, alcance{ri, ""})
		}
	} else {
		ri, rel, err := a.resolver(dentro)
		if err != nil {
			return nil, err
		}
		alcances = append(alcances, alcance{ri, rel})
	}
	// Índice al día (vigilado en vivo): respuesta instantánea sin recorrer Z:.
	ris := make([]*raizInfo, len(alcances))
	rels := make([]string, len(alcances))
	for i, al := range alcances {
		ris[i], rels[i] = al.ri, al.rel
	}
	if r, ok := a.buscarEnIndice(qs, ris, rels); ok {
		return r, nil
	}
	type hallado struct {
		e       Entrada
		n       int
		puntaje float64
	}
	var hallados []hallado
	const maximo, guardar = 300, 20000
	limite := time.Now().Add(25 * time.Second)
	errFin := errors.New("fin")
	for _, al := range alcances {
		conFicha := a.rutasConFicha(al.ri)
		inicio, _ := rutaSegura(al.ri.Ruta, al.rel)
		err := filepath.WalkDir(inicio, func(p string, d os.DirEntry, err error) error {
			if err != nil || p == inicio {
				return nil
			}
			rel, _ := filepath.Rel(al.ri.Ruta, p)
			rel = filepath.ToSlash(rel)
			if omitir(d.Name(), !strings.Contains(rel, "/")) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if n, pt := puntuar(qs, d.Name()); n > 0 && len(hallados) < guardar {
				if info, err := d.Info(); err == nil {
					e := entradaDe(rel, info)
					e.Ficha = e.EsCarpeta && conFicha[strings.ToLower(rel)]
					e.Ruta = unir(al.ri.ID, rel)
					hallados = append(hallados, hallado{e, n, pt})
				}
			}
			if time.Now().After(limite) {
				res.Cortada = true
				return errFin
			}
			return nil
		})
		if err == errFin {
			break
		}
	}
	sort.SliceStable(hallados, func(i, j int) bool {
		x, y := hallados[i], hallados[j]
		if x.n != y.n {
			return x.n > y.n
		}
		if x.puntaje != y.puntaje {
			return x.puntaje > y.puntaje
		}
		return len(x.e.Nombre) < len(y.e.Nombre)
	})
	res.Total = len(hallados)
	for i := 0; i < len(hallados) && i < maximo; i++ {
		res.Entradas = append(res.Entradas, hallados[i].e)
	}
	return res, nil
}
