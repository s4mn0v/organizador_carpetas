package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	nombreBase      = "Fichas tecnicas carpetas.xlsx"
	carpetaPapelera = "_papelera_organizador"
	carpetaApp      = "organizador_carpetas"
	maxRecientes    = 8
)

// configuracion se guarda en %APPDATA%\organizador_carpetas\config.json.
type configuracion struct {
	Recientes []string          `json:"recientes,omitempty"` // últimas carpetas raíz usadas
	Colores   map[string]string `json:"colores,omitempty"`   // ruta raíz (minúsculas) -> color de sus carpetas
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
	if c.Colores == nil {
		c.Colores = map[string]string{}
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

// recordarRaiz pone la raíz de primera en recientes (máx. 8).
func recordarRaiz(dir string) error {
	c := leerConfig()
	l := []string{dir}
	for _, r := range c.Recientes {
		if !strings.EqualFold(r, dir) && len(l) < maxRecientes {
			l = append(l, r)
		}
	}
	c.Recientes = l
	return guardarConfig(c)
}

// colorGuardado: el color que el usuario le dio a esa raíz ("" si no tiene).
func colorGuardado(dir string) string { return leerConfig().Colores[strings.ToLower(dir)] }

func guardarColor(dir, color string) error {
	c := leerConfig()
	c.Colores[strings.ToLower(dir)] = color
	return guardarConfig(c)
}

func dirExiste(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
