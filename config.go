package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	nombreBase      = "Fichas tecnicas carpetas.xlsx"
	carpetaPapelera = "_papelera_organizador"
	carpetaApp      = "organizador_carpetas"
)

type configuracion struct {
	Raiz      string   `json:"raiz"`
	Recientes []string `json:"recientes,omitempty"` // últimas raíces usadas
}

func rutaConfig() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, carpetaApp, "config.json")
}

func leerConfig() configuracion {
	var c configuracion
	if b, err := os.ReadFile(rutaConfig()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}

func guardarConfig(c configuracion) error {
	p := rutaConfig()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

// recordarRaiz guarda la raíz elegida y la pone primera en recientes (máx. 6).
func recordarRaiz(dir string) error {
	c := leerConfig()
	c.Raiz = dir
	l := []string{dir}
	for _, r := range c.Recientes {
		if r != dir && len(l) < 6 {
			l = append(l, r)
		}
	}
	c.Recientes = l
	return guardarConfig(c)
}

func dirExiste(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
