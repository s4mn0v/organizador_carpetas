package main

// Servidor MCP: expone las funciones de la app como herramientas para Claude Code.
//
//   - Solo escucha en 127.0.0.1 (puerto libre al azar) y exige un token.
//   - Las rutas son las mismas rutas virtuales de la app: "r1/Informes/2025".
//   - Claude solo ve las carpetas raíz que el usuario tenga abiertas.
//   - 🔴 eliminar / depurar / guardar_ficha piden confirmación en una ventana antes de ejecutar.
//   - Cada escritura emite "claude:cambio" para que el frontend se refresque.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const nombreServidorMCP = "organizador"

// EstadoClaude es lo que el frontend necesita para el toggle.
type EstadoClaude struct {
	Activo bool   `json:"activo"`
	URL    string `json:"url"`
	Config string `json:"config"` // archivo --mcp-config para Claude Code
}

type servidorMCP struct {
	mu     sync.Mutex
	http   *http.Server
	url    string
	token  string
	config string
}

var srvMCP servidorMCP

// ---------------- Toggle (métodos que ve el frontend) ----------------

// ConectarClaude levanta el servidor MCP y escribe el mcp.json para Claude Code.
func (a *App) ConectarClaude() (*EstadoClaude, error) {
	srvMCP.mu.Lock()
	defer srvMCP.mu.Unlock()
	if srvMCP.http != nil {
		return a.estadoClaude(), nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el puerto local: %w", err)
	}
	tok, err := tokenAleatorio()
	if err != nil {
		ln.Close()
		return nil, err
	}
	url := fmt.Sprintf("http://%s/mcp", ln.Addr().String())

	mux := http.NewServeMux()
	mux.Handle("/mcp", conToken(tok, server.NewStreamableHTTPServer(a.servidorMCP())))
	hs := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	cfg, err := escribirConfigMCP(url, tok)
	if err != nil {
		ln.Close()
		return nil, err
	}
	go func() { _ = hs.Serve(ln) }()

	srvMCP.http, srvMCP.url, srvMCP.token, srvMCP.config = hs, url, tok, cfg
	return a.estadoClaude(), nil
}

// DesconectarClaude apaga el servidor MCP y borra el mcp.json (el token deja de servir).
func (a *App) DesconectarClaude() error {
	srvMCP.mu.Lock()
	defer srvMCP.mu.Unlock()
	return apagarMCP()
}

// EstadoClaude dice si el complemento está conectado.
func (a *App) EstadoClaude() *EstadoClaude {
	srvMCP.mu.Lock()
	defer srvMCP.mu.Unlock()
	return a.estadoClaude()
}

func (a *App) estadoClaude() *EstadoClaude {
	return &EstadoClaude{Activo: srvMCP.http != nil, URL: srvMCP.url, Config: srvMCP.config}
}

// shutdown: al cerrar la app se apaga el servidor.
func (a *App) shutdown(ctx context.Context) {
	detenerVigilancia()
	srvMCP.mu.Lock()
	defer srvMCP.mu.Unlock()
	_ = apagarMCP()
}

func apagarMCP() error {
	if srvMCP.http == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := srvMCP.http.Shutdown(ctx)
	if srvMCP.config != "" {
		_ = os.Remove(srvMCP.config)
	}
	srvMCP.http, srvMCP.url, srvMCP.token, srvMCP.config = nil, "", "", ""
	return err
}

func tokenAleatorio() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// conToken rechaza cualquier petición sin "Authorization: Bearer <token>".
func conToken(tok string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tok {
			http.Error(w, "no autorizado", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// escribirConfigMCP deja %APPDATA%\organizador_carpetas\mcp.json listo para `claude --mcp-config`.
func escribirConfigMCP(url, tok string) (string, error) {
	cfg := map[string]any{
		"mcpServers": map[string]any{
			nombreServidorMCP: map[string]any{
				"type":    "http",
				"url":     url,
				"headers": map[string]string{"Authorization": "Bearer " + tok},
			},
		},
	}
	p := filepath.Join(filepath.Dir(rutaConfig()), "mcp.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return "", err
	}
	return p, nil
}

// ---------------- Herramientas ----------------

const instruccionesMCP = `Herramientas del "Organizador de carpetas" (SAO6).
Rutas virtuales: "<id raíz>/<ruta>" con "/" (ej. "r1/Informes/2025"); "" = todas las raíces.
Para ubicar rutas o ver la estructura use primero "indice" (pregrabado, instantáneo); "buscar" y "listar" leen el disco y son lentos.
Solo carpetas principales (directo en la raíz) tienen ficha técnica.
Eliminar y depurar mueven a la papelera de la raíz (_papelera_organizador), no borran para siempre.`

func (a *App) servidorMCP() *server.MCPServer {
	s := server.NewMCPServer(nombreServidorMCP, "1.0.0",
		server.WithToolCapabilities(false),
		server.WithInstructions(instruccionesMCP),
	)
	ruta := func(desc string) mcp.ToolOption {
		return mcp.WithString("ruta", mcp.Required(), mcp.Description(desc))
	}
	lectura := mcp.WithReadOnlyHintAnnotation(true)
	escritura := mcp.WithDestructiveHintAnnotation(false)
	peligro := mcp.WithDestructiveHintAnnotation(true)

	// ---------- 🟢 Lectura ----------

	s.AddTool(mcp.NewTool("estado", lectura,
		mcp.WithDescription("Raíces abiertas (id, ruta), si tienen Excel de fichas, campos de la ficha y fichas huérfanas.")),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return resultado(a.Estado())
		})

	s.AddTool(mcp.NewTool("listar", lectura,
		mcp.WithDescription("Contenido directo (carpetas y archivos) de una carpeta. \"\" = lista de raíces."),
		mcp.WithString("ruta", mcp.Description("Ruta virtual; \"\" = raíces"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return resultado(a.Listar(r.GetString("ruta", "")))
		})

	s.AddTool(mcp.NewTool("detalle_carpeta", lectura,
		mcp.WithDescription("Detalle de una carpeta: contenido, conteos, ficha técnica, descripciones y retención de subcarpetas."),
		ruta("Ruta virtual de la carpeta")),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			v, err := r.RequireString("ruta")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return resultado(a.Carpeta(v))
		})

	s.AddTool(mcp.NewTool("detalle_archivo", lectura,
		mcp.WithDescription("Detalle de un archivo: tamaño, fechas, carpeta y retención que le aplica."),
		ruta("Ruta virtual del archivo")),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			v, err := r.RequireString("ruta")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return resultado(a.Archivo(v))
		})

	s.AddTool(mcp.NewTool("buscar", lectura,
		mcp.WithDescription("Busca por nombre leyendo el disco (lento). Úsela solo si 'indice' con filtro no encontró algo reciente."),
		mcp.WithString("texto", mcp.Required(), mcp.Description("Palabras a buscar")),
		mcp.WithString("dentro", mcp.Description("Ruta virtual donde buscar; \"\" = todas las raíces"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			t, err := r.RequireString("texto")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			res, err := a.Buscar(t, r.GetString("dentro", ""))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			// Texto compacto (antes el JSON llegaba a ~120.000 caracteres).
			var b strings.Builder
			fmt.Fprintf(&b, "%d coincidencia(s)%s\n", res.Total, map[bool]string{true: " (búsqueda cortada por tiempo)", false: ""}[res.Cortada])
			for i, e := range res.Entradas {
				if i == 40 {
					fmt.Fprintf(&b, "… %d más: afine las palabras o use 'dentro'\n", len(res.Entradas)-40)
					break
				}
				if e.EsCarpeta {
					fmt.Fprintf(&b, "%s/\n", e.Ruta)
				} else {
					fmt.Fprintf(&b, "%s (%s)\n", e.Ruta, tamanoTexto(e.Tamano))
				}
			}
			return mcp.NewToolResultText(b.String()), nil
		})

	s.AddTool(mcp.NewTool("duplicados", lectura,
		mcp.WithDescription("Archivos idénticos (por contenido) dentro de una carpeta. Puede tardar en carpetas grandes."),
		mcp.WithString("ruta", mcp.Description("Ruta virtual; \"\" = todas las raíces"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			defer siCancela(ctx, a.CancelarDuplicados)()
			return resultado(a.Duplicados(r.GetString("ruta", "")))
		})

	s.AddTool(mcp.NewTool("tamano", lectura,
		mcp.WithDescription("Tamaño total (bytes) y cantidad de archivos de una carpeta, incluyendo subcarpetas."),
		mcp.WithString("ruta", mcp.Description("Ruta virtual; \"\" = todas las raíces"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return resultado(a.TamanoCarpeta(r.GetString("ruta", "")))
		})

	s.AddTool(mcp.NewTool("vencidos", lectura,
		mcp.WithDescription("Archivos que pasaron su tiempo de retención (según las fichas). Paso previo a 'depurar'."),
		mcp.WithString("ruta", mcp.Description("Ruta virtual; \"\" = todas las raíces"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			defer siCancela(ctx, a.CancelarVencidos)()
			return resultado(a.Vencidos(r.GetString("ruta", "")))
		})

	s.AddTool(mcp.NewTool("indice", lectura,
		mcp.WithDescription("PRIMERA OPCIÓN para ubicar carpetas/archivos y ver la estructura: índice pregrabado, responde al instante "+
			"sin recorrer el disco. Sin filtro: árbol con archivos y tamaño por carpeta. Con filtro: rutas cuyo nombre coincide."),
		mcp.WithString("ruta", mcp.Description("Ruta virtual desde donde mirar; \"\" = todas las raíces")),
		mcp.WithNumber("profundidad", mcp.Description("Niveles del árbol (1 a 6)"), mcp.DefaultNumber(2)),
		mcp.WithBoolean("archivos", mcp.Description("Incluir archivos en el árbol"), mcp.DefaultBool(false)),
		mcp.WithString("filtro", mcp.Description("Palabras a buscar en los nombres (ignora tildes y mayúsculas)"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			p := min(max(r.GetInt("profundidad", 2), 1), 6)
			txt, err := a.consultarIndice(r.GetString("ruta", ""), p, r.GetBool("archivos", false), strings.TrimSpace(r.GetString("filtro", "")))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(txt), nil
		})

	// ---------- 🟡 Escritura (quedan en el Historial del Excel) ----------

	s.AddTool(mcp.NewTool("crear_carpeta", escritura,
		mcp.WithDescription("Crea una subcarpeta. Devuelve su ruta virtual."),
		mcp.WithString("padre", mcp.Required(), mcp.Description("Ruta virtual donde se crea (raíz o carpeta)")),
		mcp.WithString("nombre", mcp.Required(), mcp.Description("Nombre de la carpeta nueva"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			p, err1 := r.RequireString("padre")
			n, err2 := r.RequireString("nombre")
			if err := errors.Join(err1, err2); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return a.cambio(a.CrearCarpeta(p, n))
		})

	s.AddTool(mcp.NewTool("renombrar", escritura,
		mcp.WithDescription("Cambia el nombre de una carpeta o archivo (misma ubicación). Devuelve la ruta nueva."),
		ruta("Ruta virtual actual"),
		mcp.WithString("nuevo_nombre", mcp.Required(), mcp.Description("Solo el nombre, sin ruta"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			v, err1 := r.RequireString("ruta")
			n, err2 := r.RequireString("nuevo_nombre")
			if err := errors.Join(err1, err2); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return a.cambio(a.Renombrar(v, n))
		})

	s.AddTool(mcp.NewTool("mover", escritura,
		mcp.WithDescription("Mueve una carpeta o archivo a otra carpeta (puede ser de otra raíz). Devuelve la ruta nueva."),
		ruta("Ruta virtual de lo que se mueve"),
		mcp.WithString("destino", mcp.Required(), mcp.Description("Ruta virtual de la carpeta destino"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			v, err1 := r.RequireString("ruta")
			d, err2 := r.RequireString("destino")
			if err := errors.Join(err1, err2); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return a.cambio(a.Mover(v, d))
		})

	// ---------- 🔴 Con confirmación ----------

	s.AddTool(mcp.NewTool("guardar_ficha", peligro,
		mcp.WithDescription("Crea o actualiza la ficha técnica de una carpeta principal. Solo cambia los campos enviados; "+
			"los demás se conservan. Nombres de campo: ver 'estado'. Pide confirmación al usuario."),
		ruta("Ruta virtual de la carpeta principal"),
		mcp.WithObject("valores", mcp.Required(),
			mcp.Description("Campo -> valor, ej. {\"Responsable\":\"Juan\"}"),
			mcp.AdditionalProperties(map[string]any{"type": "string"}))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				Ruta    string            `json:"ruta"`
				Valores map[string]string `json:"valores"`
			}
			if err := r.BindArguments(&args); err != nil || args.Ruta == "" || len(args.Valores) == 0 {
				return mcp.NewToolResultError("se requieren 'ruta' y 'valores'"), nil
			}
			reg, err := a.fichaDe(args.Ruta)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			var cambios []string
			for k, v := range args.Valores {
				cambios = append(cambios, fmt.Sprintf("• %s: %q → %q", k, reg.Valores[k], v))
				reg.Valores[k] = v
			}
			accion := "Actualizar"
			if reg.ID == 0 {
				accion = "Crear"
			}
			if !a.confirmar(accion+" ficha técnica",
				fmt.Sprintf("Claude quiere %s la ficha de:\n%s\n\n%s", strings.ToLower(accion), args.Ruta, strings.Join(cambios, "\n"))) {
				return mcp.NewToolResultError("el usuario no lo autorizó"), nil
			}
			// subs = nil: no toca las descripciones de subcarpetas.
			return a.cambio(a.GuardarRegistro(*reg, nil))
		})

	s.AddTool(mcp.NewTool("eliminar", peligro,
		mcp.WithDescription("Mueve una carpeta o archivo a la papelera de su raíz (y borra sus fichas del Excel). "+
			"Pide confirmación al usuario."),
		ruta("Ruta virtual de lo que se elimina")),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			v, err := r.RequireString("ruta")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			res, err := a.ContarEliminar(v)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			msg := fmt.Sprintf("Claude quiere eliminar:\n%s\n\n%d carpetas, %d archivos, %d fichas, %d descripciones.",
				v, res.Carpetas, res.Archivos, res.Fichas, res.Descripciones)
			if res.Cortado {
				msg += "\n(Es muy grande: el conteo es parcial.)"
			}
			if !a.confirmar("Eliminar", msg+"\n\nSe mueve a la papelera de la raíz.") {
				return mcp.NewToolResultError("el usuario no lo autorizó"), nil
			}
			return a.cambio(a.Eliminar(v))
		})

	s.AddTool(mcp.NewTool("depurar", peligro,
		mcp.WithDescription("Mueve a la papelera los archivos indicados (normalmente los que devolvió 'vencidos'). "+
			"Pide confirmación al usuario."),
		mcp.WithArray("rutas", mcp.Required(), mcp.WithStringItems(), mcp.Description("Rutas virtuales de archivos"))),
		func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rutas, err := r.RequireStringSlice("rutas")
			if err != nil || len(rutas) == 0 {
				return mcp.NewToolResultError("se requiere 'rutas'"), nil
			}
			muestra := rutas
			if len(muestra) > 10 {
				muestra = muestra[:10]
			}
			msg := fmt.Sprintf("Claude quiere depurar %d archivos:\n%s", len(rutas), strings.Join(muestra, "\n"))
			if len(rutas) > 10 {
				msg += fmt.Sprintf("\n… y %d más", len(rutas)-10)
			}
			if !a.confirmar("Depurar archivos", msg+"\n\nSe mueven a la papelera de cada raíz.") {
				return mcp.NewToolResultError("el usuario no lo autorizó"), nil
			}
			return a.cambio(a.Depurar(rutas))
		})

	return s
}

// ---------------- Ayudantes ----------------

// resultado convierte (valor, error) de los métodos de App en respuesta MCP (JSON compacto).
func resultado(v any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Tope de tokens: una respuesta enorme no le sirve a Claude y cuesta mucho.
	const maxTexto = 15000
	if len(b) > maxTexto {
		return mcp.NewToolResultText(strings.ToValidUTF8(string(b[:maxTexto]), "") + "\n… [respuesta recortada: use 'indice' con ruta o filtro para ir más preciso]"), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// cambio = resultado + avisar al frontend para que recargue.
func (a *App) cambio(v any, err error) (*mcp.CallToolResult, error) {
	if err == nil {
		a.emitir("claude:cambio", v)
	}
	return resultado(v, err)
}

// siCancela: si Claude cancela la llamada, detiene la tarea larga. Devuelve la función para soltar el vigilante.
func siCancela(ctx context.Context, cancelar func()) func() {
	listo := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cancelar()
		case <-listo:
		}
	}()
	return func() { close(listo) }
}

// confirmar muestra una ventana Sí/No. Sin ventana (ctx nil) no autoriza.
func (a *App) confirmar(titulo, mensaje string) bool {
	if a.ctx == nil {
		return false
	}
	runtime.WindowShow(a.ctx)
	res, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         titulo,
		Message:       mensaje,
		Buttons:       []string{"Sí", "No"},
		DefaultButton: "No",
		CancelButton:  "No",
	})
	if err != nil {
		return false
	}
	return res == "Sí" || strings.EqualFold(res, "Yes")
}

// fichaDe devuelve la ficha actual de una carpeta principal (o una nueva vacía) con sus valores copiados.
func (a *App) fichaDe(v string) (*Registro, error) {
	d, err := a.Carpeta(v)
	if err != nil {
		return nil, err
	}
	if !d.EsPrincipal {
		return nil, errors.New("solo las carpetas principales (directo en la raíz) tienen ficha técnica")
	}
	v, _ = limpiarRel(v)
	reg := &Registro{Ruta: v, Valores: map[string]string{}}
	for _, r := range d.Registros {
		if strings.EqualFold(r.Ruta, v) {
			reg.ID = r.ID
			for k, val := range r.Valores {
				reg.Valores[k] = val
			}
			break
		}
	}
	return reg, nil
}
