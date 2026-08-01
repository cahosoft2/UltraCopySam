//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCopyFileExW           = kernel32.NewProc("CopyFileExW")
	procCreateDirectoryW      = kernel32.NewProc("CreateDirectoryW")
	procSetFileAttributesW    = kernel32.NewProc("SetFileAttributesW")
	procGetFileAttributesW    = kernel32.NewProc("GetFileAttributesW")
	procGetVolumeInformationW = kernel32.NewProc("GetVolumeInformationW")
)

// Flags de CopyFileExW / atributos de archivo.
const (
	copyFileNoBuffering             = 0x00001000
	copyFileAllowDecryptedDest      = 0x00000008
	invalidFileAttributes    uint32 = 0xFFFFFFFF

	fileAttributeReadonly  = 0x00000001
	fileAttributeHidden    = 0x00000002
	fileAttributeSystem    = 0x00000004
	fileAttributeDirectory = 0x00000010
	fileAttributeNormal    = 0x00000080

	errorFileExists     syscall.Errno = 80
	errorAlreadyExists  syscall.Errno = 183
	errorAccessDenied   syscall.Errno = 5
	errorInvalidParam   syscall.Errno = 87
	errorPathNotFound   syscall.Errno = 3
	errorFileNotFound   syscall.Errno = 2
	errorNotSupported   syscall.Errno = 50
	errorInvalidFunc    syscall.Errno = 1
	errorSharingViolate syscall.Errno = 32
)

// copyFileEx envuelve CopyFileExW: la copia la realiza el kernel de Windows,
// sin mover los bytes al espacio de usuario. Sobrescribe el destino si existe.
func copyFileEx(src, dst string, flags uint32) error {
	psrc, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	pdst, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	r, _, e := procCopyFileExW.Call(
		uintptr(unsafe.Pointer(psrc)),
		uintptr(unsafe.Pointer(pdst)),
		0, // sin callback de progreso: menos overhead por bloque
		0,
		0, // sin pbCancel
		uintptr(flags),
	)
	if r == 0 {
		return e
	}
	return nil
}

// createDirectory crea un único directorio.
//
// Devuelve yaExistia=true si el directorio ya estaba creado. Ese dato importa:
// un directorio recién creado está vacío por definición, así que el modo -u
// puede ahorrarse listarlo para buscar archivos que no pueden existir.
func createDirectory(path string) (yaExistia bool, err error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	r, _, e := procCreateDirectoryW.Call(uintptr(unsafe.Pointer(p)), 0)
	if r == 0 {
		if e == errorAlreadyExists {
			return true, nil
		}
		return false, e
	}
	return false, nil
}

// sistemaDeArchivos devuelve el nombre del sistema de archivos del volumen
// ("NTFS", "exFAT", "FAT32"...). raiz debe ser la raíz del volumen, con barra
// final: "D:\" o "\\servidor\recurso\".
//
// Se usa para decidir la precisión con la que comparar fechas: NTFS las guarda
// con resolución de 100 ns, mientras que FAT32 redondea a 2 segundos.
func sistemaDeArchivos(raiz string) string {
	p, err := syscall.UTF16PtrFromString(raiz)
	if err != nil {
		return ""
	}
	nombre := make([]uint16, 261)

	r, _, _ := procGetVolumeInformationW.Call(
		uintptr(unsafe.Pointer(p)),
		0, 0, // nombre del volumen: no interesa
		0, 0, 0, // número de serie, longitud máxima, flags: no interesan
		uintptr(unsafe.Pointer(&nombre[0])),
		uintptr(len(nombre)),
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(nombre)
}

func getFileAttributes(path string) (uint32, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return invalidFileAttributes, err
	}
	r, _, e := procGetFileAttributesW.Call(uintptr(unsafe.Pointer(p)))
	if uint32(r) == invalidFileAttributes {
		return invalidFileAttributes, e
	}
	return uint32(r), nil
}

func setFileAttributes(path string, attrs uint32) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, e := procSetFileAttributesW.Call(uintptr(unsafe.Pointer(p)), uintptr(attrs))
	if r == 0 {
		return e
	}
	return nil
}

// clearBlockingAttrs quita readonly/hidden/system del destino para poder
// sobrescribirlo sin preguntar.
func clearBlockingAttrs(path string) error {
	attrs, err := getFileAttributes(path)
	if err != nil {
		return err
	}
	cleaned := attrs &^ (fileAttributeReadonly | fileAttributeHidden | fileAttributeSystem)
	if cleaned == attrs {
		return nil
	}
	if cleaned == 0 {
		cleaned = fileAttributeNormal
	}
	return setFileAttributes(path, cleaned)
}
