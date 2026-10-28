package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Raiz es una carpeta raíz abierta en la app.
type Raiz struct {
	ID     string `json:"id"`     // "r1", "r2"… (primer tramo de las rutas virtuales)
	Ruta   string `json:"ruta"`   // ruta en Windows, p. ej. Z:\MANTENIMIENTO\PRIVADA
	Nombre string `json:"nombre"` // lo que se muestra: "MANTENIMIENTO\PRIVADA"
	Color  string `json:"color"`  // color de los íconos de sus carpetas
}

type raizInfo struct {
	Raiz
	cache      *baseDatos
	cacheFecha time.Time
	migrado    bool
	aviso      string
}

// Colores que se asignan en orden a las raíces (el usuario los puede cambiar).
var paleta = []string{"#f2b33d", "#4a90d9", "#3fae6a", "#9b6dd6", "#e0664f", "#2fb5b0", "#d6609c", "#8aa0b0"}

// etiquetaRaiz: los dos últimos tramos de la ruta, para distinguir p. ej.
// MANTENIMIENTO\PRIVADA de RRHH\PRIVADA.
func etiquetaRaiz(dir string) string {
	var partes []string
	for _, p := range strings.FieldsFunc(dir, func(r rune) bool { return r == '\\' || r == '/' }) {
		if p != "" {
			partes = append(partes, p)
		}
	}
	if len(partes) == 0 {
		return dir
	}
	if len(partes) == 1 {
		return partes[0]
	}
	return partes[len(partes)-2] + `\` + partes[len(partes)-1]
}

func (a *App) lista() []*raizInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]*raizInfo(nil), a.raices...)
}

func (a *App) raizPorID(id string) (*raizInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ri := range a.raices {
		if ri.ID == id {
			return ri, nil
		}
	}
	return nil, errors.New("esa carpeta raíz ya no está abierta")
}

// resolver separa una ruta virtual en su raíz y la ruta dentro de ella ("" = la raíz misma).
func (a *App) resolver(v string) (*raizInfo, string, error) {
	v, ok := limpiarRel(v)
	if !ok || v == "" {
		return nil, "", errors.New("ruta no válida")
	}
	id, rel := v, ""
	if i := strings.Index(v, "/"); i >= 0 {
		id, rel = v[:i], v[i+1:]
	}
	ri, err := a.raizPorID(id)
	if err != nil {
		return nil, "", err
	}
	return ri, rel, nil
}

// absoluta: raíz, ruta dentro de la raíz y ruta completa en disco.
func (a *App) absoluta(v string) (*raizInfo, string, string, error) {
	ri, rel, err := a.resolver(v)
	if err != nil {
		return nil, "", "", err
	}
	abs, ok := rutaSegura(ri.Ruta, rel)
	if !ok {
		return nil, "", "", errors.New("ruta no válida")
	}
	return ri, rel, abs, nil
}

// AgregarRaiz pide una carpeta y la agrega a las abiertas (nil si el usuario cancela).
func (a *App) AgregarRaiz() (*Raiz, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Carpeta raíz a organizar"})
	if err != nil || dir == "" {
		return nil, err
	}
	return a.AgregarRaizRuta(dir)
}

// AgregarRaizRuta agrega una carpeta conocida (lista de recientes).
func (a *App) AgregarRaizRuta(dir string) (*Raiz, error) {
	dir = filepath.Clean(dir)
	if !dirExiste(dir) {
		return nil, fmt.Errorf("no se encuentra la carpeta %s", dir)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	usados := map[string]bool{}
	for _, ri := range a.raices {
		if strings.EqualFold(ri.Ruta, dir) {
			return &ri.Raiz, nil // ya está abierta
		}
		if dentroDeDisco(dir, ri.Ruta) {
			return nil, fmt.Errorf("esa carpeta está dentro de %s, que ya está abierta", ri.Nombre)
		}
		if dentroDeDisco(ri.Ruta, dir) {
			return nil, fmt.Errorf("esa carpeta contiene a %s, que ya está abierta: quítela primero", ri.Nombre)
		}
		usados[strings.ToLower(ri.Color)] = true
	}
	color := colorGuardado(dir)
	if color == "" {
		color = paleta[len(a.raices)%len(paleta)]
		for _, c := range paleta {
			if !usados[c] {
				color = c
				break
			}
		}
	}
	a.sigID++
	ri := &raizInfo{Raiz: Raiz{ID: fmt.Sprintf("r%d", a.sigID), Ruta: dir, Nombre: etiquetaRaiz(dir), Color: color}}
	a.raices = append(a.raices, ri)
	recordarRaiz(dir)
	return &ri.Raiz, nil
}

// QuitarRaiz la cierra en la app (no toca la carpeta ni su Excel).
func (a *App) QuitarRaiz(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, ri := range a.raices {
		if ri.ID == id {
			a.raices = append(a.raices[:i], a.raices[i+1:]...)
			return nil
		}
	}
	return nil
}

// ColorRaiz cambia el color de los íconos de una raíz (se recuerda para la próxima vez).
func (a *App) ColorRaiz(id, color string) error {
	ri, err := a.raizPorID(id)
	if err != nil {
		return err
	}
	if len(color) != 7 || color[0] != '#' {
		return errors.New("color no válido")
	}
	a.mu.Lock()
	ri.Color = color
	a.mu.Unlock()
	return guardarColor(ri.Ruta, color)
}

// dentroDeDisco: hijo está dentro de padre (rutas de Windows, sin distinguir mayúsculas).
func dentroDeDisco(hijo, padre string) bool {
	h := strings.ToLower(filepath.Clean(hijo))
	p := strings.TrimRight(strings.ToLower(filepath.Clean(padre)), `\/`)
	return strings.HasPrefix(h, p+string(filepath.Separator)) || strings.HasPrefix(h, p+"/")
}
