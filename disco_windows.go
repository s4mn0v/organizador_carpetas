//go:build windows

package main

// Lectura rápida de carpetas con la API nativa de Windows (sirve sobre todo en unidades de red como Z:):
//   - FindFirstFileExW con FIND_FIRST_EX_LARGE_FETCH: pide lotes grandes al servidor (menos idas y vueltas).
//   - FindExInfoBasic: no pide los nombres cortos 8.3 (menos trabajo para el servidor).
//   - Prefijo \\?\ para rutas de más de 260 caracteres.

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procFindFirstFileExW = windows.NewLazySystemDLL("kernel32.dll").NewProc("FindFirstFileExW")
	procFindNextFileW    = windows.NewLazySystemDLL("kernel32.dll").NewProc("FindNextFileW")
)

const (
	findExInfoBasic       = 1
	findExSearchNameMatch = 0
	findFirstExLargeFetch = 2
)

// datosBusqueda = WIN32_FIND_DATAW tal como lo llena Windows.
type datosBusqueda struct {
	Atributos   uint32
	Creacion    windows.Filetime
	Acceso      windows.Filetime
	Escritura   windows.Filetime
	TamAlto     uint32
	TamBajo     uint32
	Reservado0  uint32
	Reservado1  uint32
	Nombre      [260]uint16
	NombreCorto [14]uint16
}

func rutaLarga(abs string) string {
	if strings.HasPrefix(abs, `\\?\`) {
		return abs
	}
	if strings.HasPrefix(abs, `\\`) { // \\servidor\recurso -> \\?\UNC\servidor\recurso
		return `\\?\UNC\` + abs[2:]
	}
	return `\\?\` + abs
}

// leerDirRapido lista una carpeta (un nivel) con FindFirstFileExW.
func leerDirRapido(abs string) ([]entradaDir, error) {
	patron, err := syscall.UTF16PtrFromString(strings.TrimRight(rutaLarga(abs), `\`) + `\*`)
	if err != nil {
		return nil, err
	}
	var d datosBusqueda
	h, _, e := procFindFirstFileExW.Call(uintptr(unsafe.Pointer(patron)), findExInfoBasic, uintptr(unsafe.Pointer(&d)),
		findExSearchNameMatch, 0, findFirstExLargeFetch)
	if windows.Handle(h) == windows.InvalidHandle {
		if e == windows.ERROR_FILE_NOT_FOUND {
			return nil, nil
		}
		return nil, e
	}
	defer windows.FindClose(windows.Handle(h))
	var out []entradaDir
	for {
		nombre := windows.UTF16ToString(d.Nombre[:])
		if nombre != "." && nombre != ".." {
			// Enlaces/uniones se tratan como archivo: no se siguen (evita ciclos, igual que WalkDir).
			dir := d.Atributos&windows.FILE_ATTRIBUTE_DIRECTORY != 0 && d.Atributos&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0
			ent := entradaDir{Nombre: nombre, Dir: dir, Mod: d.Escritura.Nanoseconds() / 1e9}
			if !dir {
				ent.Tam = int64(d.TamAlto)<<32 | int64(d.TamBajo)
			}
			out = append(out, ent)
		}
		r, _, e := procFindNextFileW.Call(h, uintptr(unsafe.Pointer(&d)))
		if r == 0 {
			if e == windows.ERROR_NO_MORE_FILES {
				return out, nil
			}
			return out, e
		}
	}
}
