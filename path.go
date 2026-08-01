//go:build windows

package main

import (
	"path/filepath"
	"strings"
)

// extendedPath convierte una ruta a la forma extendida \\?\ para saltarse el
// límite de MAX_PATH (260 caracteres). La ruta debe venir ya absoluta y limpia,
// porque Win32 no normaliza nada bajo el prefijo \\?\.
func extendedPath(p string) string {
	if strings.HasPrefix(p, `\\?\`) || strings.HasPrefix(p, `\\.\`) {
		return p
	}
	p = filepath.Clean(p)
	if strings.HasPrefix(p, `\\`) { // UNC: \\servidor\recurso
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + p
}

// displayPath deshace extendedPath para mostrar rutas legibles al usuario.
func displayPath(p string) string {
	switch {
	case strings.HasPrefix(p, `\\?\UNC\`):
		return `\\` + p[len(`\\?\UNC\`):]
	case strings.HasPrefix(p, `\\?\`):
		return p[len(`\\?\`):]
	default:
		return p
	}
}

// join concatena sin pasar por filepath.Join, que volvería a limpiar la ruta
// (y es el punto caliente del recorrido).
func join(dir, name string) string {
	if strings.HasSuffix(dir, `\`) {
		return dir + name
	}
	return dir + `\` + name
}
