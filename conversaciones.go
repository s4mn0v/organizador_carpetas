package main

// Historial de conversaciones con Claude: máximo 5. Para crear otra hay que eliminar una
// (así no se acumulan temas viejos). Al eliminar se borra también la sesión que guarda
// Claude Code (~/.claude/projects/*/<sesión>.jsonl): no queda "memoria basura".
//
// Archivo: %APPDATA%\organizador_carpetas\claude\conversaciones.json

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxConversaciones = 5

// Conversacion guardada. Mensajes es el JSON de la lista que pinta el panel (lo arma el frontend).
type Conversacion struct {
	ID          string    `json:"id"`
	Titulo      string    `json:"titulo"`
	Sesion      string    `json:"sesion"` // sesión de Claude Code (para --resume)
	Creada      string    `json:"creada"`
	Actualizada string    `json:"actualizada"`
	Mensajes    string    `json:"mensajes"`
	Uso         UsoClaude `json:"uso"` // tokens acumulados
}

// ResumenConversacion: para la lista del historial (sin los mensajes).
type ResumenConversacion struct {
	ID          string `json:"id"`
	Titulo      string `json:"titulo"`
	Actualizada string `json:"actualizada"`
	Actual      bool   `json:"actual"`
}

var convMu sync.Mutex

func rutaConversaciones() string {
	return filepath.Join(filepath.Dir(rutaConfig()), "claude", "conversaciones.json")
}

func leerConversaciones() []*Conversacion {
	var l []*Conversacion
	if b, err := os.ReadFile(rutaConversaciones()); err == nil {
		_ = json.Unmarshal(b, &l)
	}
	return l
}

func guardarConversaciones(l []*Conversacion) error {
	p := rutaConversaciones()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.Marshal(l)
	return os.WriteFile(p, b, 0o644)
}

// ListarConversaciones: más reciente primero.
func (a *App) ListarConversaciones() []ResumenConversacion {
	actual := convActual() // antes de convMu: mismo orden de candados que el resto (chat.mu → convMu)
	convMu.Lock()
	defer convMu.Unlock()
	l := leerConversaciones()
	sort.Slice(l, func(i, j int) bool { return l[i].Actualizada > l[j].Actualizada })
	out := []ResumenConversacion{}
	for _, c := range l {
		out = append(out, ResumenConversacion{ID: c.ID, Titulo: c.Titulo, Actualizada: c.Actualizada, Actual: c.ID == actual})
	}
	return out
}

// NuevaConversacionClaude crea una conversación vacía y la deja como actual. Error si ya hay 5.
func (a *App) NuevaConversacionClaude() (string, error) {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	if chat.proc != nil {
		return "", errors.New("Claude todavía está respondiendo")
	}
	convMu.Lock()
	defer convMu.Unlock()
	l := leerConversaciones()
	if len(l) >= maxConversaciones {
		return "", errors.New("límite de 5 conversaciones: elimine una del historial para crear otra")
	}
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	ahora := time.Now().Format(time.RFC3339)
	c := &Conversacion{ID: hex.EncodeToString(b), Titulo: "Nueva conversación", Creada: ahora, Actualizada: ahora, Mensajes: "[]"}
	if err := guardarConversaciones(append(l, c)); err != nil {
		return "", err
	}
	chat.conv, chat.sesion = c.ID, ""
	return c.ID, nil
}

// AbrirConversacion la deja como actual (el próximo mensaje continúa esa sesión) y la devuelve.
func (a *App) AbrirConversacion(id string) (*Conversacion, error) {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	if chat.proc != nil {
		return nil, errors.New("Claude todavía está respondiendo")
	}
	convMu.Lock()
	defer convMu.Unlock()
	for _, c := range leerConversaciones() {
		if c.ID == id {
			chat.conv, chat.sesion = c.ID, c.Sesion
			return c, nil
		}
	}
	return nil, errors.New("esa conversación ya no existe")
}

// GuardarConversacion guarda título y mensajes de la conversación (y la sesión de Claude si es la actual).
func (a *App) GuardarConversacion(id, titulo, mensajes string) error {
	chat.mu.Lock()
	sesion, actual := chat.sesion, chat.conv
	chat.mu.Unlock()
	convMu.Lock()
	defer convMu.Unlock()
	l := leerConversaciones()
	for _, c := range l {
		if c.ID == id {
			if t := strings.TrimSpace(titulo); t != "" {
				c.Titulo = recortar(t, 60)
			}
			if json.Valid([]byte(mensajes)) {
				c.Mensajes = mensajes
			}
			if id == actual && sesion != "" {
				c.Sesion = sesion
			}
			c.Actualizada = time.Now().Format(time.RFC3339)
			return guardarConversaciones(l)
		}
	}
	return errors.New("esa conversación ya no existe")
}

// EliminarConversacion la borra del historial y borra la sesión que guardó Claude Code.
func (a *App) EliminarConversacion(id string) error {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	if chat.proc != nil && chat.conv == id {
		return errors.New("Claude todavía está respondiendo en esa conversación")
	}
	convMu.Lock()
	defer convMu.Unlock()
	l := leerConversaciones()
	var quedan []*Conversacion
	for _, c := range l {
		if c.ID == id {
			borrarSesionClaude(c.Sesion, chat.proc == nil)
			continue
		}
		quedan = append(quedan, c)
	}
	quitarLogConv(id)
	if chat.conv == id {
		chat.conv, chat.sesion = "", ""
	}
	if quedan == nil {
		quedan = []*Conversacion{}
	}
	return guardarConversaciones(quedan)
}

var reSesion = regexp.MustCompile(`^[0-9a-fA-F-]{8,64}$`)

// borrarSesionClaude elimina TODO lo que Claude Code guardó en este PC de esa sesión:
//   - cualquier archivo o carpeta de ~/.claude cuyo nombre lleve el id (transcript en projects/,
//     todos/, session-env/, file-history/, debug/…), y
//   - las líneas de ~/.claude/history.jsonl que lo mencionen (solo si no hay otra respuesta en curso,
//     para no pisar a Claude Code mientras escribe).
//
// Solo actúa con ids de sesión válidos (UUID): nunca borra por un texto vacío o genérico.
func borrarSesionClaude(sesion string, tocarHistorial bool) {
	if !reSesion.MatchString(sesion) || len(sesion) < 32 {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	base := filepath.Join(home, ".claude")
	var borrar []string
	_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == base {
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(sesion)) {
			borrar = append(borrar, p)
			if d.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
	for _, p := range borrar {
		_ = os.RemoveAll(p)
	}
	if tocarHistorial {
		quitarLineas(filepath.Join(base, "history.jsonl"), sesion)
	}
}

// quitarLineas reescribe un archivo de texto sin las líneas que contienen marca (si no hay ninguna, no lo toca).
func quitarLineas(ruta, marca string) {
	b, err := os.ReadFile(ruta)
	if err != nil || !strings.Contains(string(b), marca) {
		return
	}
	info, err := os.Stat(ruta)
	if err != nil {
		return
	}
	var quedan []string
	for _, l := range strings.SplitAfter(string(b), "\n") {
		if l != "" && !strings.Contains(l, marca) {
			quedan = append(quedan, l)
		}
	}
	tmp := ruta + ".tmp"
	if os.WriteFile(tmp, []byte(strings.Join(quedan, "")), info.Mode().Perm()) == nil {
		_ = os.Rename(tmp, ruta)
	}
}

// convActual: id de la conversación actual (con chat.mu tomado o no: solo lectura rápida).
func convActual() string {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	return chat.conv
}

// ---------------- Estilo de respuesta ("skill") ----------------

const estiloPorDefecto = "Respuestas cortas: máximo 5 líneas o 6 viñetas. Sin saludos, sin introducción y sin resumir lo que hiciste. " +
	"No narres herramientas ni pasos internos. Si falta un dato, haz UNA pregunta corta. " +
	"Prefiere viñetas a párrafos y pon las rutas en código. Si el usuario pide detalle, ahí sí extiéndete."

// EstiloClaude devuelve el estilo guardado (o el de por defecto).
func (a *App) EstiloClaude() string {
	if s := strings.TrimSpace(leerConfig().EstiloClaude); s != "" {
		return s
	}
	return estiloPorDefecto
}

// GuardarEstiloClaude guarda el estilo ("" = volver al de por defecto). Devuelve el que queda.
func (a *App) GuardarEstiloClaude(texto string) (string, error) {
	c := leerConfig()
	c.EstiloClaude = strings.TrimSpace(texto)
	if c.EstiloClaude == estiloPorDefecto {
		c.EstiloClaude = ""
	}
	if err := guardarConfig(c); err != nil {
		return "", err
	}
	return a.EstiloClaude(), nil
}

// unaLinea: el estilo va como argumento de línea de comandos: sin saltos ni comillas dobles.
func unaLinea(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ", "\"", "'").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
