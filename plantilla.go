package main

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Campos por defecto de la ficha técnica de una carpeta. Se pueden cambiar después
// en la hoja "Campos" del Excel (agregar columnas, cambiar grupos, tipos u opciones).
var camposPorDefecto = []Campo{
	{Nombre: "Nombre de la carpeta", Grupo: "General", Tipo: "texto"},
	{Nombre: "Descripción / propósito", Grupo: "General", Tipo: "largo"},
	{Nombre: "Proceso / área", Grupo: "General", Tipo: "texto"},
	{Nombre: "Responsable", Grupo: "General", Tipo: "texto"},
	{Nombre: "Tipo de contenido", Grupo: "General", Tipo: "multilista", // varios valores separados por ";"
		Opciones: []string{"Indicadores", "Informes", "Formatos", "Procedimientos", "Planos y manuales", "Contratos", "Soportes", "Bases de datos", "Otro"}},
}

// crearPlantilla crea el Excel de fichas vacío con sus tres hojas.
func crearPlantilla(ruta string) error {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", hojaFichas); err != nil {
		return err
	}
	f.NewSheet(hojaSubcarpetas)
	f.NewSheet(hojaCampos)
	f.NewSheet(hojaHistorial)

	estiloTitulo, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"10313B"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})

	// Hoja Carpetas: ID, ruta, fecha y un campo por columna.
	cab := []interface{}{colID, colRuta, colActualizado}
	for _, c := range camposPorDefecto {
		cab = append(cab, c.Nombre)
	}
	f.SetSheetRow(hojaFichas, "A1", &cab)
	fin, _ := excelize.CoordinatesToCellName(len(cab), 1)
	f.SetCellStyle(hojaFichas, "A1", fin, estiloTitulo)
	f.SetColWidth(hojaFichas, "A", "A", 6)
	f.SetColWidth(hojaFichas, "B", "B", 48)
	f.SetColWidth(hojaFichas, "C", "C", 17)
	ultima, _ := excelize.ColumnNumberToName(len(cab))
	f.SetColWidth(hojaFichas, "D", ultima, 26)
	f.SetPanes(hojaFichas, &excelize.Panes{Freeze: true, XSplit: 2, YSplit: 1, TopLeftCell: "C2", ActivePane: "bottomRight"})

	// Hoja Subcarpetas: explicación de cada subcarpeta (se llena desde la ficha de su carpeta principal).
	f.SetSheetRow(hojaSubcarpetas, "A1", &[]interface{}{"Ruta subcarpeta", "Descripción", "Actualizado"})
	f.SetCellStyle(hojaSubcarpetas, "A1", "C1", estiloTitulo)
	f.SetColWidth(hojaSubcarpetas, "A", "A", 55)
	f.SetColWidth(hojaSubcarpetas, "B", "B", 90)
	f.SetColWidth(hojaSubcarpetas, "C", "C", 17)

	// Hoja Campos: cómo se muestra cada columna en la app.
	f.SetSheetRow(hojaCampos, "A1", &[]interface{}{"Campo", "Grupo", "Tipo (texto, largo, lista, multilista, numero, marca)", "Opciones (separadas por ;)", "Orden"})
	f.SetCellStyle(hojaCampos, "A1", "E1", estiloTitulo)
	for i, c := range camposPorDefecto {
		ops := ""
		for j, o := range c.Opciones {
			if j > 0 {
				ops += ";"
			}
			ops += o
		}
		celda, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(hojaCampos, celda, &[]interface{}{c.Nombre, c.Grupo, c.Tipo, ops, i + 1})
	}
	f.SetColWidth(hojaCampos, "A", "A", 32)
	f.SetColWidth(hojaCampos, "B", "B", 22)
	f.SetColWidth(hojaCampos, "C", "C", 34)
	f.SetColWidth(hojaCampos, "D", "D", 70)

	// Hoja Historial: cada cambio hecho desde la app.
	f.SetSheetRow(hojaHistorial, "A1", &[]interface{}{"Fecha", "Accion", "Ruta carpeta", "Detalle"})
	f.SetCellStyle(hojaHistorial, "A1", "D1", estiloTitulo)
	f.SetColWidth(hojaHistorial, "A", "A", 17)
	f.SetColWidth(hojaHistorial, "B", "B", 22)
	f.SetColWidth(hojaHistorial, "C", "C", 48)
	f.SetColWidth(hojaHistorial, "D", "D", 80)
	f.SetCellValue(hojaHistorial, "A2", ahora())
	f.SetCellValue(hojaHistorial, "B2", "Crear base")
	f.SetCellValue(hojaHistorial, "D2", fmt.Sprintf("Excel de fichas creado con %d campos", len(camposPorDefecto)))

	f.SetActiveSheet(0)
	return f.SaveAs(ruta)
}
