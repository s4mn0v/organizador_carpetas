package main

// Runner de Claude Code: la app lanza `claude -p` (usa la suscripción del usuario, sin API key),
// lee su salida stream-json línea por línea y la manda al frontend como eventos "claude:evento".
//
// Claude solo puede usar las herramientas del servidor MCP de la app (mcp.go):
// las herramientas propias de Claude Code (Bash, Read, Write…) quedan bloqueadas.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// EventoClaude es lo que recibe el frontend en "claude:evento".
type EventoClaude struct {
	Tipo        string     `json:"tipo"`                  // inicio | texto | herramienta | resultado | fin | error
	Texto       string     `json:"texto,omitempty"`       // texto de Claude, resultado o error
	Herramienta string     `json:"herramienta,omitempty"` // nombre corto: "buscar", "mover"…
	Entrada     any        `json:"entrada,omitempty"`     // argumentos de la herramienta
	ID          string     `json:"id,omitempty"`          // une "herramienta" con su "resultado"
	EsError     bool       `json:"esError,omitempty"`
	Sesion      string     `json:"sesion,omitempty"`
	Costo       float64    `json:"costo,omitempty"` // equivalente en USD que reporta Claude Code (informativo)
	Uso         *UsoClaude `json:"uso,omitempty"`   // tokens de esta respuesta (en "fin"/"error")
}

// ConversacionClaude: estado del chat para el frontend.
type ConversacionClaude struct {
	Conv    string `json:"conv"`    // id en el historial ("" = ninguna)
	Sesion  string `json:"sesion"`  // "" = conversación nueva
	Ocupado bool   `json:"ocupado"` // hay una respuesta en curso
}

type runnerClaude struct {
	mu     sync.Mutex
	conv   string // conversación actual del historial
	sesion string
	proc   *exec.Cmd
}

var chat runnerClaude

// Una sola línea y sin comillas: así pasa intacto como argumento aunque claude sea un .cmd.
const promptSistema = "Eres el asistente del Organizador de carpetas de mantenimiento (SAO6). " +
	"Responde en español, corto y directo. " +
	"Solo puedes ver y cambiar archivos con las herramientas mcp__organizador__*; no inventes rutas: búscalas o lístalas. " +
	"Antes de eliminar, depurar o cambiar fichas, di en una línea qué vas a hacer; la app le pedirá confirmación al usuario. " +
	"Si el usuario no autoriza una acción, no la repitas: pregunta qué prefiere. " +
	"Ubica todo con la herramienta indice (instantánea); usa buscar o listar solo si indice no basta. " +
	"Si una herramienta responde que algo no está listo o falla dos veces, detente y dilo en una línea: no reintentes."

// Herramientas propias de Claude Code que se bloquean (solo se trabaja con las del organizador).
var herramientasBloqueadas = []string{
	"Bash", "BashOutput", "KillShell", "Read", "Write", "Edit", "MultiEdit", "NotebookEdit",
	"Glob", "Grep", "WebFetch", "WebSearch", "Task", "TodoWrite", "SlashCommand", "Skill",
}

// HerramientaClaude: para los botones del panel (el usuario elige primero qué quiere hacer).
type HerramientaClaude struct {
	Nombre   string `json:"nombre"`   // nombre en el servidor MCP
	Etiqueta string `json:"etiqueta"` // lo que ve el usuario
	Tipo     string `json:"tipo"`     // lectura | escritura | peligro
	Pista    string `json:"pista"`    // ejemplo para la caja de texto
}

// catalogo: mismas herramientas que registra mcp.go (mismo orden que en el panel).
var catalogo = []HerramientaClaude{
	{"indice", "Estructura", "lectura", "Ej.: dónde están los informes de bogies 2025 / resume esta raíz"},
	{"buscar", "Buscar en disco", "lectura", "Ej.: busca en el disco un archivo recién creado"},
	{"detalle_carpeta", "Ver carpeta", "lectura", "Ej.: qué tiene esta carpeta y su ficha"},
	{"detalle_archivo", "Ver archivo", "lectura", "Ej.: fechas y retención de este archivo"},
	{"listar", "Listar", "lectura", "Ej.: qué hay dentro de esta carpeta"},
	{"tamano", "Tamaño", "lectura", "Ej.: cuánto pesa esta carpeta"},
	{"duplicados", "Duplicados", "lectura", "Ej.: archivos repetidos en esta carpeta"},
	{"vencidos", "Vencidos", "lectura", "Ej.: qué archivos ya pasaron su retención"},
	{"estado", "Estado", "lectura", "Ej.: qué raíces hay abiertas y si tienen Excel"},
	{"crear_carpeta", "Crear carpeta", "escritura", "Ej.: crea la carpeta 2026 aquí"},
	{"renombrar", "Renombrar", "escritura", "Ej.: renombra esto a Informes 2025"},
	{"mover", "Mover", "escritura", "Ej.: mueve esto a Archivo histórico"},
	{"guardar_ficha", "Editar ficha", "peligro", "Ej.: pon Responsable = Juan Pérez"},
	{"eliminar", "Eliminar", "peligro", "Ej.: elimina esta carpeta"},
	{"depurar", "Depurar", "peligro", "Ej.: depura los vencidos de esta carpeta"},
}

// Siempre disponibles cuando se elige una herramienta: para ubicar raíces y rutas.
var herramientasBase = []string{"estado", "indice"}

// HerramientasClaude: catálogo para el panel.
func (a *App) HerramientasClaude() []HerramientaClaude { return catalogo }

// EnviarAClaude manda un mensaje. La respuesta llega por eventos "claude:evento"; termina con tipo "fin" o "error".
//   - herramienta: la que eligió el usuario ("" = todas). Claude solo recibe esa + estado e indice,
//     así no tiene que revisar las 15 (menos tokens, más directo).
//   - contexto: ruta virtual seleccionada en la app ("" = ninguna).
func (a *App) EnviarAClaude(mensaje, herramienta, contexto string) error {
	mensaje = strings.TrimSpace(mensaje)
	if mensaje == "" {
		return errors.New("mensaje vacío")
	}
	permitidas, err := herramientasPermitidas(herramienta)
	if err != nil {
		return err
	}
	// Sin índice listo no se gasta ni un token: Claude tendría que recorrer el disco a ciegas.
	for _, st := range a.EstadosIndice() {
		if !st.Listo || st.Dudoso {
			return fmt.Errorf("el índice de %s todavía se está creando (%d elementos leídos): espere a que termine", st.Raiz, st.Leidos)
		}
	}
	var pre []string
	if herramienta != "" {
		pre = append(pre, "[Herramienta elegida por el usuario: "+herramienta+"]")
	}
	if contexto != "" {
		pre = append(pre, "[Carpeta/archivo seleccionado en la app: "+contexto+"]")
	}
	if len(pre) > 0 {
		mensaje = strings.Join(pre, "\n") + "\n\n" + mensaje
	}
	st := a.EstadoClaude()
	if !st.Activo {
		var err error
		if st, err = a.ConectarClaude(); err != nil {
			return err
		}
	}
	exe, err := buscarClaude()
	if err != nil {
		return err
	}
	dir, err := carpetaChat()
	if err != nil {
		return err
	}

	chat.mu.Lock()
	defer chat.mu.Unlock()
	if chat.proc != nil {
		return errors.New("Claude todavía está respondiendo")
	}
	if chat.conv == "" {
		return errors.New("no hay conversación abierta: cree una o abra una del historial")
	}
	conv := chat.conv // el registro marca cada línea con su conversación (para borrarla con ella)

	args := []string{
		"-p",
		"--output-format", "stream-json", "--verbose",
		"--mcp-config", st.Config, "--strict-mcp-config",
		"--allowedTools", "mcp__" + nombreServidorMCP + "__*",
		"--disallowedTools", strings.Join(append(append([]string{}, herramientasBloqueadas...), fuera(permitidas)...), ","),
		"--append-system-prompt", promptSistema + " ESTILO: " + unaLinea(a.EstiloClaude()),
	}
	if chat.sesion != "" {
		args = append(args, "--resume", chat.sesion)
	}
	c := comandoOculto(exe, args...)
	// Pocas herramientas: sin ToolSearch (evita un viaje extra y que el panel quede "cargando").
	// Sin tráfico no esencial: Claude Code no envía telemetría ni reportes de error a terceros.
	if c.Env == nil {
		c.Env = os.Environ()
	}
	c.Env = append(c.Env, "ENABLE_TOOL_SEARCH=false", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")
	c.Dir = dir                          // carpeta fija: Claude Code guarda las sesiones por carpeta
	c.Stdin = strings.NewReader(mensaje) // el mensaje va por stdin: sin problemas de comillas
	salida, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	errores := &escritorLog{a: a, conv: conv, prefij: "stderr │ "}
	c.Stderr = errores
	a.logConv(conv, "$ claude -p --output-format stream-json%s  [herramientas: %s]", map[bool]string{true: " --resume " + chat.sesion, false: ""}[chat.sesion != ""], listaPermitidas(permitidas))
	if contexto != "" {
		a.logConv(conv, "  contexto: %s", contexto)
	}
	if err := c.Start(); err != nil {
		a.logConv(conv, "✗ no se pudo iniciar: %v", err)
		return fmt.Errorf("no se pudo iniciar Claude Code: %w", err)
	}
	chat.proc = c

	go a.leerStream(c, salida, errores)
	return nil
}

// CancelarClaude detiene la respuesta en curso.
func (a *App) CancelarClaude() {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	if chat.proc != nil && chat.proc.Process != nil {
		_ = chat.proc.Process.Kill()
	}
}

// ConversacionClaude dice en qué va el chat.
func (a *App) ConversacionClaude() *ConversacionClaude {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	return &ConversacionClaude{Conv: chat.conv, Sesion: chat.sesion, Ocupado: chat.proc != nil}
}

// ---------------- Lectura del stream ----------------

// Líneas de `claude -p --output-format stream-json` (solo los campos que se usan).
type lineaStream struct {
	Type      string   `json:"type"`    // system | assistant | user | result
	Subtype   string   `json:"subtype"` // init | success | error_…
	SessionID string   `json:"session_id"`
	IsError   bool     `json:"is_error"`
	Result    string   `json:"result"`
	CostoUSD  float64  `json:"total_cost_usd"`
	Model     string   `json:"model"`
	Tools     []string `json:"tools"`
	Duracion  int      `json:"duration_ms"`
	Turnos    int      `json:"num_turns"`
	Usage     struct {
		Input         int `json:"input_tokens"`
		Output        int `json:"output_tokens"`
		CacheRead     int `json:"cache_read_input_tokens"`
		CacheCreation int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
	Message struct {
		Content json.RawMessage `json:"content"` // lista de bloques o, a veces, texto
	} `json:"message"`
}

type bloqueStream struct {
	Type    string          `json:"type"` // text | tool_use | tool_result
	ID      string          `json:"id"`
	UsoID   string          `json:"tool_use_id"`
	Text    string          `json:"text"`
	Name    string          `json:"name"`
	Input   any             `json:"input"`
	Content json.RawMessage `json:"content"` // tool_result: texto o lista de bloques
	IsError bool            `json:"is_error"`
}

func (a *App) leerStream(c *exec.Cmd, salida io.Reader, errores *escritorLog) {
	conv := errores.conv
	terminado := false
	sc := bufio.NewScanner(salida)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // resultados grandes (listados, vencidos)
	for sc.Scan() {
		var l lineaStream
		if json.Unmarshal(sc.Bytes(), &l) != nil {
			continue
		}
		if l.SessionID != "" {
			chat.mu.Lock()
			chat.sesion = l.SessionID
			chat.mu.Unlock()
		}
		switch l.Type {
		case "system":
			if l.Subtype == "init" {
				a.logConv(conv, "● sesión %s · modelo %s · %d herramientas", l.SessionID, l.Model, len(l.Tools))
				a.emitir("claude:evento", EventoClaude{Tipo: "inicio", Sesion: l.SessionID})
			}
		case "assistant":
			for _, b := range bloques(l.Message.Content) {
				switch b.Type {
				case "text":
					if strings.TrimSpace(b.Text) != "" {
						a.logConv(conv, "◆ texto (%d caracteres)", len([]rune(b.Text)))
						a.emitir("claude:evento", EventoClaude{Tipo: "texto", Texto: b.Text})
					}
				case "tool_use":
					ent, _ := json.Marshal(b.Input)
					a.logConv(conv, "→ %s %s", nombreCorto(b.Name), recortar(string(ent), 200))
					a.emitir("claude:evento", EventoClaude{Tipo: "herramienta", ID: b.ID, Herramienta: nombreCorto(b.Name), Entrada: b.Input})
				}
			}
		case "user":
			for _, b := range bloques(l.Message.Content) {
				if b.Type == "tool_result" {
					txt := textoResultado(b.Content)
					a.logConv(conv, "← resultado%s: %s caracteres (~%s tokens)", map[bool]string{true: " ✗ ERROR", false: ""}[b.IsError], miles(len([]rune(txt))), miles(len([]rune(txt))/4))
					a.emitir("claude:evento", EventoClaude{Tipo: "resultado", ID: b.UsoID, Texto: recortar(textoResultado(b.Content), 2000), EsError: b.IsError})
				}
			}
		case "result":
			terminado = true
			t := "fin"
			if l.IsError {
				t = "error"
			}
			uso := UsoClaude{Entrada: l.Usage.Input, Salida: l.Usage.Output, CacheLectura: l.Usage.CacheRead,
				CacheCreacion: l.Usage.CacheCreation, CostoUSD: l.CostoUSD, Turnos: l.Turnos, DuracionMs: l.Duracion}
			sumarUso(conv, uso)
			a.logConv(conv, "■ %s · %d turnos · %.1f s · tokens: entrada %s, salida %s, caché leída %s, caché creada %s · ≈ US$%.4f",
				map[bool]string{true: "error", false: "fin"}[l.IsError], l.Turnos, float64(l.Duracion)/1000,
				miles(uso.Entrada), miles(uso.Salida), miles(uso.CacheLectura), miles(uso.CacheCreacion), l.CostoUSD)
			a.emitir("claude:evento", EventoClaude{Tipo: t, Texto: l.Result, Sesion: l.SessionID, Costo: l.CostoUSD, Uso: &uso})
		}
	}
	err := c.Wait()

	chat.mu.Lock()
	chat.proc = nil
	chat.mu.Unlock()

	if err != nil {
		a.logConv(conv, "✗ proceso terminó: %v", err)
	}
	if !terminado { // se cayó, se canceló o no arrancó (p. ej. sin sesión iniciada)
		msg := strings.TrimSpace(errores.String())
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "Claude terminó sin responder"
		}
		a.emitir("claude:evento", EventoClaude{Tipo: "error", Texto: recortar(msg, 2000)})
	}
}

// ---------------- Ayudantes ----------------

// listaPermitidas: nombres en el orden del catálogo (para el registro).
func listaPermitidas(m map[string]bool) string {
	var l []string
	for _, c := range catalogo {
		if m[c.Nombre] {
			l = append(l, c.Nombre)
		}
	}
	return strings.Join(l, ", ")
}

// herramientasPermitidas: "" = todas; si no, la elegida + las base.
func herramientasPermitidas(h string) (map[string]bool, error) {
	ok := map[string]bool{}
	if h == "" {
		for _, c := range catalogo {
			ok[c.Nombre] = true
		}
		return ok, nil
	}
	for _, c := range catalogo {
		if c.Nombre == h {
			ok[h] = true
			for _, b := range herramientasBase {
				ok[b] = true
			}
			return ok, nil
		}
	}
	return nil, fmt.Errorf("herramienta desconocida: %s", h)
}

// fuera: nombres completos (mcp__organizador__x) de las herramientas que NO se permiten.
func fuera(permitidas map[string]bool) []string {
	var l []string
	for _, c := range catalogo {
		if !permitidas[c.Nombre] {
			l = append(l, "mcp__"+nombreServidorMCP+"__"+c.Nombre)
		}
	}
	return l
}

// buscarClaude encuentra el ejecutable (PATH o la ruta del instalador nativo).
func buscarClaude() (string, error) {
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, p := range []string{
			filepath.Join(home, ".local", "bin", "claude.exe"),
			filepath.Join(home, ".local", "bin", "claude"),
		} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", errors.New("no se encontró Claude Code: instálelo y ejecute `claude` una vez para iniciar sesión")
}

// carpetaChat: %APPDATA%\organizador_carpetas\claude (fija, para que --resume encuentre la sesión).
func carpetaChat() (string, error) {
	d := filepath.Join(filepath.Dir(rutaConfig()), "claude")
	return d, os.MkdirAll(d, 0o755)
}

// bloques: el contenido de un mensaje del stream viene como lista de bloques o como texto suelto.
func bloques(raw json.RawMessage) []bloqueStream {
	var l []bloqueStream
	if json.Unmarshal(raw, &l) == nil {
		return l
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return []bloqueStream{{Type: "text", Text: s}}
	}
	return nil
}

// nombreCorto: "mcp__organizador__buscar" -> "buscar".
func nombreCorto(n string) string {
	if i := strings.LastIndex(n, "__"); i >= 0 {
		return n[i+2:]
	}
	return n
}

// textoResultado saca el texto de un tool_result (viene como string o como lista de bloques).
func textoResultado(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var bloques []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &bloques) == nil {
		var partes []string
		for _, b := range bloques {
			if b.Type == "text" {
				partes = append(partes, b.Text)
			}
		}
		return strings.Join(partes, "\n")
	}
	return string(raw)
}

func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
