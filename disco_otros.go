//go:build !windows

package main

import "os"

// leerDirRapido (Linux/macOS): os.ReadDir; la versión rápida es la de Windows.
func leerDirRapido(abs string) ([]entradaDir, error) {
	ents, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	out := make([]entradaDir, 0, len(ents))
	for _, e := range ents {
		ed := entradaDir{Nombre: e.Name(), Dir: e.IsDir()}
		if info, err := e.Info(); err == nil {
			ed.Mod = info.ModTime().Unix()
			if !ed.Dir {
				ed.Tam = info.Size()
			}
		}
		out = append(out, ed)
	}
	return out, nil
}

// vigilar no está implementado fuera de Windows: el índice funciona igual, pero sin avisos en vivo.
func vigilar(string, func([]string), func(string)) (func(), error) {
	return nil, errNoVigilancia
}
