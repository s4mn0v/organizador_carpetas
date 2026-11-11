//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
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
	// Entre comillas a mano: el Explorador corta en las comas si la ruta va sin comillas.
	if err := explorerCrudo(`"` + ruta + `"`); err != nil {
		return comandoOculto("rundll32", "url.dll,FileProtocolHandler", ruta).Start()
	}
	return nil
}

// abrirCarpeta abre la carpeta misma en el Explorador (rutas largas: forma corta 8.3).
func abrirCarpeta(ruta string) error {
	r := ruta
	if len(r) >= windows.MAX_PATH {
		if c := rutaCorta(r); c != "" {
			r = c
		}
	}
	return explorerCrudo(`"` + r + `"`)
}

// mostrarEnExplorador abre la carpeta del archivo con el archivo seleccionado.
//
// Antes se usaba `explorer /select, ruta`, pero el Explorador interpreta mal su línea de comandos:
// con rutas de más de 260 caracteres (comunes en Z: y en lo que se encuentra con el buscador) o con
// comas en el nombre, no encuentra el archivo y abre la carpeta del usuario. Ahora:
//  1. API del shell (SHOpenFolderAndSelectItems): sin línea de comandos que interpretar.
//  2. Si la ruta es larga, se prueba su forma corta 8.3.
//  3. Último recurso: explorer con la ruta entre comillas armada a mano, o la carpeta que lo contiene.
func mostrarEnExplorador(ruta string) error {
	r := ruta
	if len(r) >= windows.MAX_PATH {
		if c := rutaCorta(r); c != "" {
			r = c
		}
	}
	if len(r) < windows.MAX_PATH && seleccionarEnShell(r) == nil {
		return nil
	}
	if len(r) < windows.MAX_PATH {
		return explorerCrudo(`/select,"` + r + `"`)
	}
	return explorerCrudo(`"` + filepath.Dir(ruta) + `"`) // ni la forma corta cabe: al menos su carpeta
}

var (
	shell32                        = windows.NewLazySystemDLL("shell32.dll")
	procILCreateFromPathW          = shell32.NewProc("ILCreateFromPathW")
	procILFree                     = shell32.NewProc("ILFree")
	procSHOpenFolderAndSelectItems = shell32.NewProc("SHOpenFolderAndSelectItems")
)

// seleccionarEnShell: SHOpenFolderAndSelectItems (necesita COM en el hilo que llama).
func seleccionarEnShell(ruta string) error {
	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err == nil {
		defer windows.CoUninitialize()
	}
	p, err := syscall.UTF16PtrFromString(ruta)
	if err != nil {
		return err
	}
	pidl, _, _ := procILCreateFromPathW.Call(uintptr(unsafe.Pointer(p)))
	if pidl == 0 {
		return fmt.Errorf("el shell no reconoce la ruta")
	}
	defer procILFree.Call(pidl)
	if hr, _, _ := procSHOpenFolderAndSelectItems.Call(pidl, 0, 0, 0); hr != 0 {
		return fmt.Errorf("SHOpenFolderAndSelectItems: 0x%x", hr)
	}
	return nil
}

// rutaCorta: forma 8.3 de una ruta larga ("" si el disco no la tiene).
func rutaCorta(ruta string) string {
	p, err := syscall.UTF16PtrFromString(rutaLarga(ruta))
	if err != nil {
		return ""
	}
	buf := make([]uint16, 32768)
	n, err := windows.GetShortPathName(p, &buf[0], uint32(len(buf)))
	if err != nil || n == 0 || int(n) > len(buf) {
		return ""
	}
	c := windows.UTF16ToString(buf[:n])
	if strings.HasPrefix(c, `\\?\UNC\`) {
		return `\\` + c[8:]
	}
	return strings.TrimPrefix(c, `\\?\`)
}

// explorerCrudo lanza explorer.exe con la línea de comandos exacta (sin el entrecomillado de Go).
func explorerCrudo(args string) error {
	c := exec.Command("explorer.exe")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: "explorer.exe " + args}
	c.Env = entornoLimpio()
	if err := c.Start(); err != nil {
		return err
	}
	go c.Wait()
	return nil
}

// fechaCreacion lee la fecha de creación que guarda Windows.
func fechaCreacion(info os.FileInfo) string {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds()).Format(time.RFC3339)
	}
	return ""
}
