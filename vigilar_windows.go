//go:build windows

package main

// Vigilancia en vivo con ReadDirectoryChangesW: Windows avisa cada archivo/carpeta que se crea, borra,
// renombra o modifica dentro de la raíz (también si lo hace otra persona desde otro PC en Z:).
// Así el índice nunca queda desactualizado y no hay que volver a recorrer todo.

import (
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const filtroCambios = windows.FILE_NOTIFY_CHANGE_FILE_NAME | windows.FILE_NOTIFY_CHANGE_DIR_NAME |
	windows.FILE_NOTIFY_CHANGE_SIZE | windows.FILE_NOTIFY_CHANGE_LAST_WRITE

// vigilar arranca la vigilancia de raiz. alCambio recibe rutas relativas ("a/b.xlsx");
// alFallo avisa que se perdieron avisos ("desborde") o que la conexión falló: hay que re-escanear.
func vigilar(raiz string, alCambio func([]string), alFallo func(string)) (func(), error) {
	abrir := func() (windows.Handle, error) {
		p, err := syscall.UTF16PtrFromString(rutaLarga(raiz))
		if err != nil {
			return windows.InvalidHandle, err
		}
		return windows.CreateFile(p, windows.FILE_LIST_DIRECTORY,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	}
	h, err := abrir()
	if err != nil {
		return nil, err
	}
	var detenido atomic.Bool
	var actual atomic.Uintptr
	actual.Store(uintptr(h))

	go func() {
		buf := make([]uint32, 16*1024) // 64 KB alineado (máximo que acepta una unidad de red)
		for !detenido.Load() {
			hh := windows.Handle(actual.Load())
			var n uint32
			err := windows.ReadDirectoryChanges(hh, (*byte)(unsafe.Pointer(&buf[0])), uint32(len(buf)*4), true, filtroCambios, &n, nil, 0)
			if detenido.Load() {
				return
			}
			if err != nil {
				if err == windows.ERROR_NOTIFY_ENUM_DIR {
					alFallo("desborde")
					continue
				}
				// Se cayó la conexión con Z: (u otro error): se reintenta y se re-escanea al volver.
				alFallo("conexión: " + err.Error())
				windows.CloseHandle(hh)
				for !detenido.Load() {
					time.Sleep(5 * time.Second)
					if nh, err := abrir(); err == nil {
						actual.Store(uintptr(nh))
						alFallo("reconectado")
						break
					}
				}
				continue
			}
			if n == 0 { // demasiados cambios a la vez: Windows no alcanzó a guardarlos todos
				alFallo("desborde")
				continue
			}
			var rels []string
			b := unsafe.Slice((*byte)(unsafe.Pointer(&buf[0])), n)
			for off := uint32(0); off+12 <= n; {
				sig := *(*uint32)(unsafe.Pointer(&b[off]))
				largo := *(*uint32)(unsafe.Pointer(&b[off+8]))
				if off+12+largo > n {
					break
				}
				nombre := unsafe.Slice((*uint16)(unsafe.Pointer(&b[off+12])), largo/2)
				rels = append(rels, windows.UTF16ToString(nombre))
				if sig == 0 {
					break
				}
				off += sig
			}
			if len(rels) > 0 {
				alCambio(rels)
			}
		}
	}()

	return func() {
		if detenido.Swap(true) {
			return
		}
		hh := windows.Handle(actual.Load())
		_ = windows.CancelIoEx(hh, nil)
		_ = windows.CloseHandle(hh) // destraba ReadDirectoryChangesW
	}, nil
}
