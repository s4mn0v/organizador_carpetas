//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func aPapelera(ruta string) error { return os.RemoveAll(ruta) }

func abrirConSistema(ruta string) error { return exec.Command("xdg-open", ruta).Start() }

func mostrarEnExplorador(ruta string) error {
	return exec.Command("xdg-open", filepath.Dir(ruta)).Start()
}

func ocultar(string) {} // en Linux/macOS el punto inicial ya lo oculta

func fechaCreacion(os.FileInfo) string { return "" }
