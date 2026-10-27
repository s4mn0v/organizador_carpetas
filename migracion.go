package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Secciones que se quitaron de la ficha técnica. Los Excel creados antes las traen
// (hoja Campos + columnas en la hoja Carpetas): la app las oculta y las borra del Excel.
var gruposRetirados = []string{"Gestión documental", "Notas"}

// Campos que traían esas secciones en la primera plantilla (por si la hoja Campos ya no los lista).
var camposRetirados = []string{"Clasificación", "Frecuencia de actualización", "Tiempo de retención", "Estado", "Palabras clave", "Observaciones"}

func normal(s string) string { return strings.ToLower(sinTildes(strings.TrimSpace(s))) }

func esGrupoRetirado(g string) bool {
	for _, r := range gruposRetirados {
		if normal(g) == normal(r) {
			return true
		}
	}
	return false
}

// nombresRetirados: campos a quitar según la hoja Campos (grupo retirado) y, si un campo de la
// primera plantilla no aparece en Campos con otro grupo, también ese.
func nombresRetirados(f *excelize.File) (map[string]bool, []int) {
	quitar := map[string]bool{}
	enOtroGrupo := map[string]bool{}
	var filasCampos []int
	if cf, err := f.GetRows(hojaCampos); err == nil {
		for i, fila := range cf {
			if i == 0 || len(fila) == 0 || strings.TrimSpace(fila[0]) == "" {
				continue
			}
			grupo := ""
			if len(fila) > 1 {
				grupo = fila[1]
			}
			if esGrupoRetirado(grupo) {
				quitar[normal(fila[0])] = true
				filasCampos = append(filasCampos, i+1)
			} else {
				enOtroGrupo[normal(fila[0])] = true
			}
		}
	}
	for _, c := range camposRetirados {
		if !enOtroGrupo[normal(c)] {
			quitar[normal(c)] = true
		}
	}
	return quitar, filasCampos
}

// migrarBase quita del Excel las secciones retiradas. Antes guarda un respaldo en la papelera
// de la app. Devuelve qué campos quitó y dónde quedó el respaldo ("" si no había nada que hacer).
func migrarBase(ruta, raiz string) ([]string, string, error) {
	f, err := excelize.OpenFile(ruta)
	if err != nil {
		return nil, "", err
	}
	quitar, filasCampos := nombresRetirados(f)
	cab, _ := f.GetRows(hojaFichas)
	var columnas []int // índices (1-based) de columnas a borrar en la hoja Carpetas
	var nombres []string
	if len(cab) > 0 {
		for i, h := range cab[0] {
			if quitar[normal(h)] {
				columnas = append(columnas, i+1)
				nombres = append(nombres, strings.TrimSpace(h))
			}
		}
	}
	f.Close()
	if len(columnas) == 0 && len(filasCampos) == 0 {
		return nil, "", nil
	}
	if err := puedeEscribir(ruta); err != nil {
		return nil, "", err
	}

	// Respaldo del Excel tal como estaba.
	sello := time.Now().Format("2006-01-02 150405")
	dirRespaldo := filepath.Join(raiz, carpetaPapelera, "respaldos Excel")
	if err := os.MkdirAll(dirRespaldo, 0o755); err != nil {
		return nil, "", err
	}
	respaldo := filepath.Join(dirRespaldo, sello+" "+filepath.Base(ruta))
	if err := copiarConFecha(ruta, respaldo); err != nil {
		return nil, "", fmt.Errorf("no se pudo respaldar el Excel: %w", err)
	}
	ocultar(filepath.Join(raiz, carpetaPapelera))

	f, err = excelize.OpenFile(ruta)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	sort.Sort(sort.Reverse(sort.IntSlice(filasCampos)))
	for _, fila := range filasCampos {
		if err := f.RemoveRow(hojaCampos, fila); err != nil {
			return nil, "", err
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(columnas)))
	for _, c := range columnas {
		letra, _ := excelize.ColumnNumberToName(c)
		if err := f.RemoveCol(hojaFichas, letra); err != nil {
			return nil, "", err
		}
	}
	relRespaldo, _ := filepath.Rel(raiz, respaldo)
	historial(f, "Quitar secciones", "", fmt.Sprintf("Se quitaron %s de la ficha (campos: %s). Respaldo: %s",
		strings.Join(gruposRetirados, " y "), strings.Join(nombres, ", "), relRespaldo))
	if err := guardarLibro(f); err != nil {
		return nil, "", err
	}
	return nombres, relRespaldo, nil
}
