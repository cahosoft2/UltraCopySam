//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Caracteres que Windows prohíbe en nombres de archivo o directorio.
// Los dos puntos se validan aparte, porque sí son válidos en la letra de unidad.
const invalidPathChars = `<>"|?*`

// validarNumeroDeArgumentos exige exactamente dos rutas y detecta el caso
// clásico de "D:\origen\" "E:\destino\", donde las barras finales escapan las
// comillas y Windows entrega ambas rutas fusionadas en un solo argumento.
func validarNumeroDeArgumentos(args []string) error {
	switch {
	case len(args) == 2:
		return nil
	case len(args) == 0:
		return fmt.Errorf("faltan las dos rutas (origen y destino)")
	case len(args) == 1 && strings.Contains(args[0], `"`):
		return fmt.Errorf(
			"se recibió un solo argumento con una comilla dentro: %s\n"+
				"  Las barras invertidas finales dentro de las comillas fusionaron ambas rutas.\n"+
				"  Escriba \"D:\\origen\" \"E:\\destino\" sin barra final",
			args[0])
	case len(args) == 1:
		return fmt.Errorf("falta la ruta de destino (solo se recibió: %s)", args[0])
	default:
		return fmt.Errorf(
			"se recibieron %d argumentos y solo se esperan 2 (origen y destino): %s\n"+
				"  Si alguna ruta lleva espacios, enciérrela entre comillas dobles",
			len(args), strings.Join(args, " | "))
	}
}

// sanitizeArg normaliza un argumento de ruta recibido por línea de comandos.
//
// Windows ya elimina las comillas dobles que envuelven un argumento, así que
// "D:\Mis Datos\origen" llega limpio. Pero hay dos casos que sí hay que tratar:
//
//   - Comillas escritas dentro del argumento (p. ej. desde un .bat o al pegar
//     una ruta copiada del Explorador con comillas incluidas).
//   - La barra invertida final: "D:\origen\" hace que la \ escape a la comilla,
//     con lo que ambas rutas se fusionan en un único argumento.
func sanitizeArg(raw, etiqueta string) (string, error) {
	s := strings.TrimSpace(raw)

	// Comillas envolventes residuales (llegan así desde .bat o al pegar rutas).
	for len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}

	if s == "" {
		return "", fmt.Errorf("la ruta de %s está vacía", etiqueta)
	}

	// Comilla suelta en medio: casi siempre es el efecto de la barra final.
	if strings.Contains(s, `"`) {
		return "", fmt.Errorf(
			"la ruta de %s llegó mal formada (%s).\n"+
				"  Causa habitual: una barra invertida final dentro de las comillas, como \"D:\\ruta\\\",\n"+
				"  donde la \\ escapa a la comilla y fusiona los dos parámetros.\n"+
				"  Solución: escriba \"D:\\ruta\" (sin barra final) o \"D:\\ruta\\\\\" (barra duplicada)",
			etiqueta, s)
	}

	if err := validarCaracteres(s, etiqueta); err != nil {
		return "", err
	}

	abs, err := filepath.Abs(s)
	if err != nil {
		return "", fmt.Errorf("la ruta de %s no se puede resolver (%s): %v", etiqueta, s, err)
	}
	return abs, nil
}

// validarCaracteres rechaza caracteres que Windows no admite en rutas, sin
// confundirse con la letra de unidad ni con los prefijos UNC / \\?\.
func validarCaracteres(s, etiqueta string) error {
	resto := s
	switch {
	case strings.HasPrefix(resto, `\\?\UNC\`):
		resto = resto[len(`\\?\UNC\`):]
	case strings.HasPrefix(resto, `\\?\`):
		resto = resto[len(`\\?\`):]
	case strings.HasPrefix(resto, `\\`):
		resto = resto[2:]
	}
	// Letra de unidad: "D:" o "D:\..."
	if len(resto) >= 2 && resto[1] == ':' {
		resto = resto[2:]
	}

	if i := strings.IndexAny(resto, invalidPathChars); i >= 0 {
		return fmt.Errorf("la ruta de %s contiene un carácter no permitido en Windows (%q): %s",
			etiqueta, resto[i], s)
	}
	if strings.Contains(resto, ":") {
		return fmt.Errorf("la ruta de %s contiene ':' fuera de la letra de unidad: %s", etiqueta, s)
	}
	for _, r := range resto {
		if r < 0x20 {
			return fmt.Errorf("la ruta de %s contiene un carácter de control no imprimible: %s", etiqueta, s)
		}
	}
	return nil
}

// validarDirectorios comprueba que origen y destino sean utilizables entre sí.
func validarDirectorios(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("el directorio de origen no existe: %s", src)
		}
		return fmt.Errorf("no se puede leer el origen %s: %v", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("el origen no es un directorio, es un archivo: %s", src)
	}

	if dinfo, err := os.Stat(dst); err == nil && !dinfo.IsDir() {
		return fmt.Errorf("el destino existe pero es un archivo, no un directorio: %s", dst)
	}

	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		return fmt.Errorf("el origen y el destino son el mismo directorio: %s", src)
	}
	if isInside(dst, src) {
		return fmt.Errorf("el destino está dentro del origen (%s dentro de %s): la copia sería recursiva", dst, src)
	}
	return nil
}
