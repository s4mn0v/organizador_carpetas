package main

// Consumo de tokens y registro tipo terminal de Claude Code (para el popup del panel).
//
//   - Uso: cada respuesta de `claude -p` trae sus tokens; se suman en la conversación guardada.
//   - Registro: últimas 1000 líneas de lo que pasa (comando, sesión, herramientas, errores).
//     Va también en vivo al frontend con el evento "claude:log".

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// UsoClaude: tokens de una respuesta o acumulados de una conversación.
type UsoClaude struct {
	Entrada       int     `json:"entrada"`       // input_tokens (sin caché)
	Salida        int     `json:"salida"`        // output_tokens
	CacheLectura  int     `json:"cacheLectura"`  // cache_read_input_tokens (más baratos)
	CacheCreacion int     `json:"cacheCreacion"` // cache_creation_input_tokens
	CostoUSD      float64 `json:"costoUSD"`      // equivalente en precios de API (informativo con suscripción)
	Turnos        int     `json:"turnos"`        // vueltas modelo↔herramientas
	DuracionMs    int     `json:"duracionMs"`
	Mensajes      int     `json:"mensajes"` // respuestas sumadas (solo en el acumulado)
}

// Total: todos los tokens procesados.
func (u UsoClaude) Total() int { return u.Entrada + u.Salida + u.CacheLectura + u.CacheCreacion }

func (u *UsoClaude) sumar(o UsoClaude) {
	u.Entrada += o.Entrada
	u.Salida += o.Salida
	u.CacheLectura += o.CacheLectura
	u.CacheCreacion += o.CacheCreacion
	u.CostoUSD += o.CostoUSD
	u.Turnos += o.Turnos
	u.DuracionMs += o.DuracionMs
	u.Mensajes++
}

// sumarUso agrega el consumo de una respuesta a la conversación (si sigue existiendo).
func sumarUso(conv string, u UsoClaude) {
	if conv == "" {
		return
	}
	convMu.Lock()
	defer convMu.Unlock()
	l := leerConversaciones()
	for _, c := range l {
		if c.ID == conv {
			c.Uso.sumar(u)
			_ = guardarConversaciones(l)
			return
		}
	}
}

// UsoConversacion: consumo acumulado de una conversación del historial.
func (a *App) UsoConversacion(id string) UsoClaude {
	convMu.Lock()
	defer convMu.Unlock()
	for _, c := range leerConversaciones() {
		if c.ID == id {
			return c.Uso
		}
	}
	return UsoClaude{}
}

// ---------------- Registro ----------------

const maxLineasLog = 1000

type lineaLog struct {
	conv  string // conversación a la que pertenece ("" = general, p. ej. el índice)
	texto string
}

var registro struct {
	mu     sync.Mutex
	lineas []lineaLog
}

// logClaude agrega una línea general (no de una conversación).
func (a *App) logClaude(formato string, args ...any) { a.logConv("", formato, args...) }

// logConv agrega una línea con hora, marcada con su conversación, y la manda al frontend.
func (a *App) logConv(conv, formato string, args ...any) {
	l := time.Now().Format("15:04:05") + "  " + fmt.Sprintf(formato, args...)
	registro.mu.Lock()
	registro.lineas = append(registro.lineas, lineaLog{conv, l})
	if n := len(registro.lineas); n > maxLineasLog {
		registro.lineas = append([]lineaLog(nil), registro.lineas[n-maxLineasLog:]...)
	}
	registro.mu.Unlock()
	a.emitir("claude:log", l)
}

// LogClaude: el registro completo (para abrir el popup).
func (a *App) LogClaude() []string {
	registro.mu.Lock()
	defer registro.mu.Unlock()
	out := make([]string, 0, len(registro.lineas))
	for _, l := range registro.lineas {
		out = append(out, l.texto)
	}
	return out
}

// quitarLogConv borra del registro todas las líneas de una conversación.
func quitarLogConv(conv string) {
	if conv == "" {
		return
	}
	registro.mu.Lock()
	defer registro.mu.Unlock()
	var quedan []lineaLog
	for _, l := range registro.lineas {
		if l.conv != conv {
			quedan = append(quedan, l)
		}
	}
	registro.lineas = quedan
}

// LimpiarLogClaude borra el registro.
func (a *App) LimpiarLogClaude() {
	registro.mu.Lock()
	registro.lineas = nil
	registro.mu.Unlock()
}

// escritorLog: lo que Claude Code escribe en stderr va al registro línea por línea (y se guarda para el error final).
type escritorLog struct {
	a      *App
	conv   string
	buf    strings.Builder
	todo   strings.Builder
	prefij string
}

func (w *escritorLog) Write(p []byte) (int, error) {
	w.todo.Write(p)
	w.buf.Write(p)
	s := w.buf.String()
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			break
		}
		if l := strings.TrimSpace(s[:i]); l != "" {
			w.a.logConv(w.conv, "%s%s", w.prefij, recortar(l, 400))
		}
		s = s[i+1:]
	}
	w.buf.Reset()
	w.buf.WriteString(s)
	return len(p), nil
}

func (w *escritorLog) String() string { return w.todo.String() }

// miles: 12345 -> "12.345".
func miles(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}
