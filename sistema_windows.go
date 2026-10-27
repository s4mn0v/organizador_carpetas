//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Papelera de reciclaje con SHFileOperationW (se puede recuperar el archivo).
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

const (
	foDelete          = 0x0003
	fofSilent         = 0x0004
	fofNoConfirmation = 0x0010
	fofAllowUndo      = 0x0040
	fofNoErrorUI      = 0x0400
)

var procSHFileOperation = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// ocultar marca un archivo como oculto en el Explorador (si falla no pasa nada).
func ocultar(ruta string) {
	if p, err := syscall.UTF16PtrFromString(ruta); err == nil {
		attrs, err := syscall.GetFileAttributes(p)
		if err == nil {
			syscall.SetFileAttributes(p, attrs|syscall.FILE_ATTRIBUTE_HIDDEN)
		}
	}
}

func aPapelera(ruta string) error {
	from, err := syscall.UTF16FromString(ruta)
	if err != nil {
		return err
	}
	from = append(from, 0) // doble terminador requerido
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	r, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	if r != 0 {
		return fmt.Errorf("Windows no pudo enviar a la papelera (código %d). ¿El archivo está abierto?", r)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("operación cancelada")
	}
	return nil
}

// entornoLimpio: el entorno de la app sin las variables de WebView2.
// La app usa WebView2 (Wails); si un programa que se abre desde aquí (Power BI también usa
// WebView2) hereda esa configuración, su WebView2 falla con "Acceso denegado (0x80070005)".
func entornoLimpio() []string {
	var out []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "WEBVIEW2_") {
			out = append(out, kv)
		}
	}
	return out
}

func comandoOculto(nombre string, args ...string) *exec.Cmd {
	c := exec.Command(nombre, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	c.Env = entornoLimpio()
	return c
}

// abrirConSistema abre el archivo a través del Explorador de Windows, igual que un doble clic:
// el programa (p. ej. Power BI) no queda como proceso hijo de la app ni hereda su entorno.
func abrirConSistema(ruta string) error {
	c := exec.Command("explorer.exe", ruta)
	c.Env = entornoLimpio()
	if err := c.Start(); err != nil {
		return comandoOculto("rundll32", "url.dll,FileProtocolHandler", ruta).Start()
	}
	go c.Wait() // explorer.exe termina enseguida; se recoge para no dejar el proceso colgado
	return nil
}

func mostrarEnExplorador(ruta string) error {
	c := exec.Command("explorer", "/select,", ruta)
	c.Env = entornoLimpio()
	return c.Start()
}

// fechaCreacion lee la fecha de creación que guarda Windows.
func fechaCreacion(info os.FileInfo) string {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds()).Format(time.RFC3339)
	}
	return ""
}
