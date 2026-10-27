package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func sinTildes(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

func unir(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "/" + b
}

// limpiarRel normaliza una ruta relativa ("a\b" -> "a/b", "." -> "") y rechaza las que salen de la raíz.
func limpiarRel(rel string) (string, bool) {
	rel = strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(rel, "\\", "/")), "/")
	if strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

// rutaSegura convierte una ruta relativa en absoluta sin salir de la raíz ("" = la raíz).
func rutaSegura(raiz, rel string) (string, bool) {
	rel, ok := limpiarRel(rel)
	if !ok {
		return "", false
	}
	if rel == "" {
		return raiz, true
	}
	return filepath.Join(raiz, filepath.FromSlash(rel)), true
}

func padreDe(rel string) string {
	if d := path.Dir(rel); d != "." {
		return d
	}
	return ""
}

// dentroDe: rel es igual a base o está dentro de ella (sin distinguir mayúsculas, como Windows).
func dentroDe(rel, base string) bool {
	r, b := strings.ToLower(rel), strings.ToLower(base)
	return b == "" || r == b || strings.HasPrefix(r, b+"/")
}

const caracteresInvalidos = `\/:*?"<>|`

func nombreValido(n string) error {
	if strings.TrimSpace(n) == "" {
		return errors.New("el nombre no puede estar vacío")
	}
	if strings.ContainsAny(n, caracteresInvalidos) {
		return fmt.Errorf(`el nombre “%s” no puede tener \ / : * ? " < > |`, n)
	}
	if strings.HasSuffix(n, ".") || strings.HasSuffix(n, " ") {
		return fmt.Errorf("el nombre “%s” no puede terminar en punto o espacio", n)
	}
	if n == "." || n == ".." {
		return errors.New("nombre no válido")
	}
	return nil
}

// conTiempo evita que la app se congele si la unidad de red no responde.
func conTiempo(d time.Duration, fn func() error) error {
	hecho := make(chan error, 1)
	go func() { hecho <- fn() }()
	select {
	case err := <-hecho:
		return err
	case <-time.After(d):
		return errors.New("la carpeta no respondió a tiempo (¿unidad de red desconectada?)")
	}
}

// libre devuelve "dir/nombre" o "dir/nombre (2)…" que no exista.
func libre(raiz, dir, nombre string) string {
	ext := filepath.Ext(nombre)
	base := strings.TrimSuffix(nombre, ext)
	for i := 1; ; i++ {
		n := nombre
		if i > 1 {
			n = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		rel := unir(dir, n)
		abs, _ := rutaSegura(raiz, rel)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return rel
		}
	}
}

// copiarConFecha copia un archivo conservando su fecha de modificación.
func copiarConFecha(origen, destino string) error {
	info, err := os.Stat(origen)
	if err != nil {
		return err
	}
	in, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := destino + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, destino); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Chtimes(destino, time.Now(), info.ModTime())
}

// moverEntre mueve entre unidades si os.Rename no puede (p. ej. de E: a Z:).
func moverEntre(origen, destino string) error {
	if err := os.Rename(origen, destino); err == nil {
		return nil
	}
	info, err := os.Stat(origen)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if err := copiarConFecha(origen, destino); err != nil {
			return err
		}
		return os.Remove(origen)
	}
	err = filepath.WalkDir(origen, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(origen, p)
		dst := filepath.Join(destino, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		return copiarConFecha(p, dst)
	})
	if err != nil {
		os.RemoveAll(destino)
		return err
	}
	return os.RemoveAll(origen)
}
