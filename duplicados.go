package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Duplicados: archivos con el MISMO NOMBRE y el MISMO CONTENIDO, a cualquier profundidad.
// 1) se agrupan por nombre + tamaño (rápido, sin leer los archivos);
// 2) solo esos candidatos se leen para sacar su huella SHA-256 y confirmar que son idénticos.

// CopiaDup es uno de los archivos repetidos.
type CopiaDup struct {
	Ruta       string `json:"ruta"`
	Carpeta    string `json:"carpeta"` // ruta virtual de la carpeta que lo contiene
	Ubicacion  string `json:"ubicacion"`
	Tamano     int64  `json:"tamano"`
	Creado     string `json:"creado"`
	Modificado string `json:"modificado"`
}

// GrupoDup: un archivo y todas sus copias.
type GrupoDup struct {
	Nombre      string     `json:"nombre"`
	Ext         string     `json:"ext"`
	Tamano      int64      `json:"tamano"`
	Huella      string     `json:"huella"` // primeros caracteres del SHA-256
	Copias      []CopiaDup `json:"copias"`
	Desperdicio int64      `json:"desperdicio"` // espacio que se liberaría dejando una sola copia
}

// ResultadoDuplicados de una búsqueda.
type ResultadoDuplicados struct {
	Grupos      []GrupoDup `json:"grupos"`
	Revisados   int        `json:"revisados"`   // archivos recorridos
	Comparados  int        `json:"comparados"`  // archivos que se leyeron para compararlos
	Desperdicio int64      `json:"desperdicio"` // total
	Cancelado   bool       `json:"cancelado"`
	Errores     int        `json:"errores"` // archivos que no se pudieron leer (abiertos o sin permiso)
	Segundos    float64    `json:"segundos"`
}

// CancelarDuplicados detiene la búsqueda en curso.
func (a *App) CancelarDuplicados() { a.cancelarDup.Store(true) }

type candidato struct {
	ri   *raizInfo
	rel  string
	abs  string
	info os.FileInfo
}

// Duplicados busca en la carpeta v ("" = todas las raíces). Avisa el avance con el evento "duplicados:progreso".
func (a *App) Duplicados(v string) (*ResultadoDuplicados, error) {
	a.cancelarDup.Store(false)
	inicio := time.Now()
	res := &ResultadoDuplicados{Grupos: []GrupoDup{}}
	type alcance struct {
		ri  *raizInfo
		abs string
	}
	var alcances []alcance
	if v == "" {
		for _, ri := range a.lista() {
			alcances = append(alcances, alcance{ri, ri.Ruta})
		}
	} else {
		ri, _, abs, err := a.absoluta(v)
		if err != nil {
			return nil, err
		}
		alcances = append(alcances, alcance{ri, abs})
	}
	progreso := func(fase string, hechos, total int) {
		a.emitir("duplicados:progreso", map[string]interface{}{"fase": fase, "hechos": hechos, "total": total, "revisados": res.Revisados})
	}

	// 1) Recorrer y agrupar por nombre + tamaño.
	fin := errors.New("fin")
	grupos := map[string][]candidato{}
	for _, al := range alcances {
		err := filepath.WalkDir(al.abs, func(p string, d os.DirEntry, err error) error {
			if err != nil || p == al.abs {
				return nil
			}
			if a.cancelarDup.Load() {
				return fin
			}
			rel, _ := filepath.Rel(al.ri.Ruta, p)
			rel = filepath.ToSlash(rel)
			if omitir(d.Name(), !strings.Contains(rel, "/")) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil || info.Size() == 0 {
				return nil
			}
			res.Revisados++
			if res.Revisados%1000 == 0 {
				progreso("recorrer", res.Revisados, 0)
			}
			k := strings.ToLower(d.Name()) + "|" + itoa64(info.Size())
			grupos[k] = append(grupos[k], candidato{al.ri, rel, p, info})
			return nil
		})
		if err == fin {
			break
		}
	}
	if a.cancelarDup.Load() {
		res.Cancelado = true
		res.Segundos = time.Since(inicio).Seconds()
		return res, nil
	}

	// 2) Comparar el contenido de los candidatos.
	var aComparar [][]candidato
	total := 0
	for _, g := range grupos {
		if len(g) > 1 {
			aComparar = append(aComparar, g)
			total += len(g)
		}
	}
	hechos := 0
	for _, g := range aComparar {
		porHuella := map[string][]candidato{}
		for _, c := range g {
			if a.cancelarDup.Load() {
				res.Cancelado = true
				break
			}
			h, err := huella(c.abs)
			hechos++
			if hechos%20 == 0 {
				progreso("comparar", hechos, total)
			}
			if err != nil {
				res.Errores++
				continue
			}
			porHuella[h] = append(porHuella[h], c)
		}
		if res.Cancelado {
			break
		}
		for h, cs := range porHuella {
			if len(cs) < 2 {
				continue
			}
			gd := GrupoDup{Nombre: cs[0].info.Name(), Ext: strings.ToLower(filepath.Ext(cs[0].info.Name())),
				Tamano: cs[0].info.Size(), Huella: h[:12]}
			for _, c := range cs {
				gd.Copias = append(gd.Copias, CopiaDup{
					Ruta: unir(c.ri.ID, c.rel), Carpeta: unir(c.ri.ID, padreDe(c.rel)), Ubicacion: c.abs,
					Tamano: c.info.Size(), Creado: fechaCreacion(c.info), Modificado: c.info.ModTime().Format(time.RFC3339),
				})
			}
			sort.Slice(gd.Copias, func(i, j int) bool { return gd.Copias[i].Modificado < gd.Copias[j].Modificado })
			gd.Desperdicio = gd.Tamano * int64(len(gd.Copias)-1)
			res.Desperdicio += gd.Desperdicio
			res.Grupos = append(res.Grupos, gd)
		}
	}
	res.Comparados = hechos
	sort.SliceStable(res.Grupos, func(i, j int) bool { return res.Grupos[i].Desperdicio > res.Grupos[j].Desperdicio })
	progreso("fin", hechos, total)
	res.Segundos = time.Since(inicio).Seconds()
	return res, nil
}

func huella(ruta string) (string, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
