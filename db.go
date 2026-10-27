package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	hojaFichas     = "Carpetas"
	hojaCampos     = "Campos"
	hojaHistorial  = "Historial"
	colID          = "ID"
	colRuta        = "Ruta carpeta"
	colActualizado = "Actualizado"
)

// Campo describe una columna de la ficha (hoja "Campos").
type Campo struct {
	Nombre   string   `json:"nombre"`
	Grupo    string   `json:"grupo"`
	Tipo     string   `json:"tipo"` // texto | largo | lista | numero | marca
	Opciones []string `json:"opciones"`
	Orden    int      `json:"orden"`
}

// Registro es una fila de la hoja "Carpetas": la ficha técnica de una carpeta.
type Registro struct {
	ID          int               `json:"id"`
	Ruta        string            `json:"ruta"`
	Actualizado string            `json:"actualizado"`
	Valores     map[string]string `json:"valores"`
}

type baseDatos struct {
	ruta      string
	columnas  []string
	campos    []Campo
	registros []*Registro
	subs      []*DescripcionSub // hoja "Subcarpetas"
}

// DescripcionSub explica una subcarpeta dentro de la ficha de su carpeta principal.
type DescripcionSub struct {
	Ruta        string `json:"ruta"`
	Descripcion string `json:"descripcion"`
	Actualizado string `json:"actualizado"`
}

const hojaSubcarpetas = "Subcarpetas"

var errBloqueada = errors.New("la base de datos está abierta en Excel: ciérrela y vuelva a intentar")

func cargarBase(ruta string) (*baseDatos, error) {
	f, err := excelize.OpenFile(ruta)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b := &baseDatos{ruta: ruta}

	filas, err := f.GetRows(hojaFichas)
	if err != nil {
		return nil, fmt.Errorf("no se encontró la hoja %q: %w", hojaFichas, err)
	}
	if len(filas) == 0 {
		return nil, fmt.Errorf("la hoja %q está vacía", hojaFichas)
	}
	for _, h := range filas[0] {
		b.columnas = append(b.columnas, strings.TrimSpace(h))
	}
	for _, fila := range filas[1:] {
		r := &Registro{Valores: map[string]string{}}
		vacia := true
		for i, col := range b.columnas {
			v := ""
			if i < len(fila) {
				v = strings.TrimSpace(fila[i])
			}
			if v != "" {
				vacia = false
			}
			switch col {
			case colID:
				r.ID, _ = strconv.Atoi(v)
			case colRuta:
				r.Ruta = rutaDesdeExcel(v)
			case colActualizado:
				r.Actualizado = v
			default:
				if v != "" {
					r.Valores[col] = v
				}
			}
		}
		if !vacia && r.ID > 0 {
			b.registros = append(b.registros, r)
		}
	}

	// Campos: hoja de configuracion; si falta, todas las columnas como texto.
	if cf, err := f.GetRows(hojaCampos); err == nil && len(cf) > 1 {
		for i, fila := range cf[1:] {
			if len(fila) == 0 || strings.TrimSpace(fila[0]) == "" {
				continue
			}
			c := Campo{Nombre: strings.TrimSpace(fila[0]), Grupo: "Otros", Tipo: "texto", Orden: i + 1}
			if len(fila) > 1 && fila[1] != "" {
				c.Grupo = strings.TrimSpace(fila[1])
			}
			if len(fila) > 2 && fila[2] != "" {
				c.Tipo = strings.TrimSpace(fila[2])
			}
			if len(fila) > 3 && fila[3] != "" {
				for _, o := range strings.Split(fila[3], ";") {
					if o = strings.TrimSpace(o); o != "" {
						c.Opciones = append(c.Opciones, o)
					}
				}
			}
			if len(fila) > 4 {
				if n, err := strconv.Atoi(strings.TrimSpace(fila[4])); err == nil {
					c.Orden = n
				}
			}
			b.campos = append(b.campos, c)
		}
	}
	existe := map[string]bool{}
	for _, c := range b.campos {
		existe[c.Nombre] = true
	}
	for i, col := range b.columnas {
		if col == colID || col == colRuta || col == colActualizado || col == "" || existe[col] {
			continue
		}
		b.campos = append(b.campos, Campo{Nombre: col, Grupo: "Otros", Tipo: "texto", Orden: 1000 + i})
	}
	sort.SliceStable(b.campos, func(i, j int) bool { return b.campos[i].Orden < b.campos[j].Orden })

	// "Tipo de contenido" admite varios valores aunque el Excel (creado antes) diga "lista".
	for i := range b.campos {
		if normal(b.campos[i].Nombre) == normal("Tipo de contenido") && b.campos[i].Tipo == "lista" {
			b.campos[i].Tipo = "multilista"
		}
	}

	// Las secciones retiradas (Gestión documental, Notas) no se muestran aunque sigan en el Excel.
	retirados := map[string]bool{}
	for _, c := range camposRetirados {
		retirados[normal(c)] = true
	}
	visibles := b.campos[:0]
	for _, c := range b.campos {
		if esGrupoRetirado(c.Grupo) || (c.Orden >= 1000 && retirados[normal(c.Nombre)]) {
			continue
		}
		visibles = append(visibles, c)
	}
	b.campos = visibles

	// Subcarpetas: ruta | descripción | actualizado (si la hoja no existe, se crea al guardar).
	if sf, err := f.GetRows(hojaSubcarpetas); err == nil {
		for _, fila := range sf[min(1, len(sf)):] {
			if len(fila) < 2 || strings.TrimSpace(fila[0]) == "" || strings.TrimSpace(fila[1]) == "" {
				continue
			}
			d := &DescripcionSub{Ruta: rutaDesdeExcel(fila[0]), Descripcion: strings.TrimSpace(fila[1])}
			if len(fila) > 2 {
				d.Actualizado = strings.TrimSpace(fila[2])
			}
			b.subs = append(b.subs, d)
		}
	}
	return b, nil
}

// puedeEscribir detecta si Excel tiene el archivo bloqueado.
func puedeEscribir(ruta string) error {
	f, err := os.OpenFile(ruta, os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return errBloqueada
	}
	return f.Close()
}

func (b *baseDatos) indiceColumna(nombre string) int {
	for i, c := range b.columnas {
		if c == nombre {
			return i
		}
	}
	return -1
}

func (b *baseDatos) siguienteID() int {
	m := 0
	for _, r := range b.registros {
		if r.ID > m {
			m = r.ID
		}
	}
	return m + 1
}

// abrirEscritura abre el libro para modificarlo y devuelve la fila (1-based) de cada ID.
func (b *baseDatos) abrirEscritura() (*excelize.File, map[int]int, error) {
	if err := puedeEscribir(b.ruta); err != nil {
		return nil, nil, err
	}
	f, err := excelize.OpenFile(b.ruta)
	if err != nil {
		return nil, nil, err
	}
	filas, err := f.GetRows(hojaFichas)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	iID := b.indiceColumna(colID)
	pos := map[int]int{}
	for i, fila := range filas {
		if i == 0 || iID < 0 || iID >= len(fila) {
			continue
		}
		if id, err := strconv.Atoi(strings.TrimSpace(fila[iID])); err == nil {
			pos[id] = i + 1
		}
	}
	return f, pos, nil
}

func guardarLibro(f *excelize.File) error {
	if err := f.Save(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "being used") ||
			strings.Contains(strings.ToLower(err.Error()), "access is denied") ||
			strings.Contains(strings.ToLower(err.Error()), "permission") {
			return errBloqueada
		}
		return err
	}
	return nil
}

func ahora() string { return time.Now().Format("2006-01-02 15:04") }

func historial(f *excelize.File, accion, ruta, detalle string) {
	if idx, _ := f.GetSheetIndex(hojaHistorial); idx < 0 {
		f.NewSheet(hojaHistorial)
		f.SetSheetRow(hojaHistorial, "A1", &[]interface{}{"Fecha", "Accion", "Ruta carpeta", "Detalle"})
	}
	filas, _ := f.GetRows(hojaHistorial)
	celda, _ := excelize.CoordinatesToCellName(1, len(filas)+1)
	f.SetSheetRow(hojaHistorial, celda, &[]interface{}{ahora(), accion, ruta, detalle})
}

func (b *baseDatos) escribirFila(f *excelize.File, fila int, r *Registro) error {
	tipos := map[string]string{}
	for _, c := range b.campos {
		tipos[c.Nombre] = c.Tipo
	}
	for i, col := range b.columnas {
		celda, _ := excelize.CoordinatesToCellName(i+1, fila)
		var v interface{}
		switch col {
		case colID:
			v = r.ID
		case colRuta:
			v = rutaAExcel(r.Ruta)
		case colActualizado:
			v = r.Actualizado
		case "":
			continue
		default:
			s := strings.TrimSpace(r.Valores[col])
			v = s
			if tipos[col] == "numero" && s != "" {
				if n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64); err == nil {
					v = n
				}
			}
			if s == "" {
				v = nil
			}
		}
		if v == nil {
			if err := f.SetCellValue(hojaFichas, celda, ""); err != nil {
				return err
			}
			continue
		}
		if err := f.SetCellValue(hojaFichas, celda, v); err != nil {
			return err
		}
	}
	return nil
}

// guardar crea (ID 0) o actualiza una ficha y reemplaza las descripciones de sus subcarpetas
// (subs == nil deja las descripciones como están).
func (b *baseDatos) guardar(r *Registro, subs []DescripcionSub) (*Registro, error) {
	f, pos, err := b.abrirEscritura()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r.Actualizado = ahora()
	if r.Valores == nil {
		r.Valores = map[string]string{}
	}
	nuevo := r.ID == 0
	fila := 0
	if nuevo {
		r.ID = b.siguienteID()
		filas, _ := f.GetRows(hojaFichas)
		fila = len(filas) + 1
	} else {
		var ok bool
		if fila, ok = pos[r.ID]; !ok {
			return nil, fmt.Errorf("no se encontró el registro %d en el Excel", r.ID)
		}
	}
	if err := b.escribirFila(f, fila, r); err != nil {
		return nil, err
	}
	if subs != nil {
		antes := map[string]*DescripcionSub{}
		var quedan []*DescripcionSub
		for _, d := range b.subs {
			if dentroDe(d.Ruta, r.Ruta) && !strings.EqualFold(d.Ruta, r.Ruta) {
				antes[strings.ToLower(d.Ruta)] = d
			} else {
				quedan = append(quedan, d)
			}
		}
		for _, d := range subs {
			d.Ruta, _ = limpiarRel(d.Ruta)
			d.Descripcion = strings.TrimSpace(d.Descripcion)
			if d.Descripcion == "" || !dentroDe(d.Ruta, r.Ruta) || strings.EqualFold(d.Ruta, r.Ruta) {
				continue
			}
			d.Actualizado = ahora()
			if a := antes[strings.ToLower(d.Ruta)]; a != nil && a.Descripcion == d.Descripcion {
				d.Actualizado = a.Actualizado
			}
			nd := d
			quedan = append(quedan, &nd)
		}
		b.subs = quedan
		if err := b.escribirSubs(f); err != nil {
			return nil, err
		}
	}
	accion := "Actualizar ficha"
	if nuevo {
		accion = "Crear ficha"
	}
	historial(f, accion, rutaAExcel(r.Ruta), fmt.Sprintf("ID %d", r.ID))
	if err := guardarLibro(f); err != nil {
		return nil, err
	}
	return r, nil
}

// escribirSubs reescribe la hoja "Subcarpetas" con b.subs (ordenadas por ruta).
func (b *baseDatos) escribirSubs(f *excelize.File) error {
	if idx, _ := f.GetSheetIndex(hojaSubcarpetas); idx < 0 {
		if _, err := f.NewSheet(hojaSubcarpetas); err != nil {
			return err
		}
		f.SetColWidth(hojaSubcarpetas, "A", "A", 55)
		f.SetColWidth(hojaSubcarpetas, "B", "B", 90)
		f.SetColWidth(hojaSubcarpetas, "C", "C", 17)
	}
	filas, _ := f.GetRows(hojaSubcarpetas)
	for i := len(filas); i >= 2; i-- {
		f.RemoveRow(hojaSubcarpetas, i)
	}
	f.SetSheetRow(hojaSubcarpetas, "A1", &[]interface{}{"Ruta subcarpeta", "Descripción", "Actualizado"})
	sort.SliceStable(b.subs, func(i, j int) bool { return strings.ToLower(b.subs[i].Ruta) < strings.ToLower(b.subs[j].Ruta) })
	for i, d := range b.subs {
		celda, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow(hojaSubcarpetas, celda, &[]interface{}{rutaAExcel(d.Ruta), d.Descripcion, d.Actualizado}); err != nil {
			return err
		}
	}
	return nil
}

// descripcionDe: la explicación guardada para una subcarpeta ("" si no tiene).
func (b *baseDatos) descripcionDe(rel string) string {
	for _, d := range b.subs {
		if strings.EqualFold(d.Ruta, rel) {
			return d.Descripcion
		}
	}
	return ""
}

// subsBajo: cuántas descripciones hay dentro de rel (incluida rel).
func (b *baseDatos) subsBajo(rel string) int {
	n := 0
	for _, d := range b.subs {
		if dentroDe(d.Ruta, rel) {
			n++
		}
	}
	return n
}

// eliminar borra filas por ID y deja copia de los datos en el historial.
func (b *baseDatos) eliminar(ids []int, motivo string) error {
	if len(ids) == 0 {
		return nil
	}
	f, pos, err := b.abrirEscritura()
	if err != nil {
		return err
	}
	defer f.Close()
	var filas []int
	for _, id := range ids {
		p, ok := pos[id]
		if !ok {
			continue
		}
		filas = append(filas, p)
		for _, r := range b.registros {
			if r.ID == id {
				js, _ := json.Marshal(r.Valores)
				historial(f, motivo, rutaAExcel(r.Ruta), fmt.Sprintf("ID %d: %s", id, js))
			}
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(filas)))
	for _, p := range filas {
		if err := f.RemoveRow(hojaFichas, p); err != nil {
			return err
		}
	}
	return guardarLibro(f)
}

// cambiarRuta actualiza la ruta de las fichas y de las descripciones de subcarpetas
// de la carpeta movida/renombrada y de todo lo que tiene dentro.
func (b *baseDatos) cambiarRuta(vieja, nueva string) (int, error) {
	afectadas := len(b.idsBajo(vieja)) + b.subsBajo(vieja)
	if afectadas == 0 {
		return 0, nil
	}
	f, pos, err := b.abrirEscritura()
	if err != nil {
		return 0, err
	}
	defer f.Close()
	iR := b.indiceColumna(colRuta)
	iA := b.indiceColumna(colActualizado)
	for _, r := range b.registros {
		if !dentroDe(r.Ruta, vieja) {
			continue
		}
		p, ok := pos[r.ID]
		if !ok || iR < 0 {
			continue
		}
		r.Ruta = nueva + r.Ruta[len(vieja):]
		celda, _ := excelize.CoordinatesToCellName(iR+1, p)
		f.SetCellValue(hojaFichas, celda, rutaAExcel(r.Ruta))
		if iA >= 0 {
			celda, _ = excelize.CoordinatesToCellName(iA+1, p)
			f.SetCellValue(hojaFichas, celda, ahora())
		}
	}
	if b.subsBajo(vieja) > 0 {
		for _, d := range b.subs {
			if dentroDe(d.Ruta, vieja) {
				d.Ruta = nueva + d.Ruta[len(vieja):]
			}
		}
		if err := b.escribirSubs(f); err != nil {
			return 0, err
		}
	}
	historial(f, "Mover / renombrar", rutaAExcel(vieja), fmt.Sprintf("Nueva ruta: %s (%d registros del Excel)", rutaAExcel(nueva), afectadas))
	return afectadas, guardarLibro(f)
}

// eliminarBajo borra las fichas y las descripciones de la carpeta eliminada y de su contenido
// (las fichas quedan copiadas en el Historial).
func (b *baseDatos) eliminarBajo(rel string) error {
	ids := b.idsBajo(rel)
	nSubs := b.subsBajo(rel)
	if len(ids) == 0 && nSubs == 0 {
		return nil
	}
	if len(ids) > 0 {
		if err := b.eliminar(ids, "Eliminar carpeta"); err != nil {
			return err
		}
	}
	if nSubs == 0 {
		return nil
	}
	f, _, err := b.abrirEscritura()
	if err != nil {
		return err
	}
	defer f.Close()
	var quedan []*DescripcionSub
	for _, d := range b.subs {
		if dentroDe(d.Ruta, rel) {
			historial(f, "Eliminar descripción", rutaAExcel(d.Ruta), d.Descripcion)
		} else {
			quedan = append(quedan, d)
		}
	}
	b.subs = quedan
	if err := b.escribirSubs(f); err != nil {
		return err
	}
	return guardarLibro(f)
}

// idsBajo: fichas de la carpeta y de todo lo que tiene dentro.
func (b *baseDatos) idsBajo(rel string) []int {
	var ids []int
	for _, r := range b.registros {
		if dentroDe(r.Ruta, rel) {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

// En el Excel las rutas se ven como en Windows ("Área\Sub"); la raíz es ".".
func rutaAExcel(rel string) string {
	if rel == "" {
		return "."
	}
	return strings.ReplaceAll(rel, "/", "\\")
}

func rutaDesdeExcel(v string) string {
	r, _ := limpiarRel(strings.TrimSpace(v))
	return r
}
