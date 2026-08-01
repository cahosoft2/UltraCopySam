//go:build windows

// UltraCopySam — copia recursiva de directorios en Windows a bajo nivel.
//
// Usa CopyFileExW (el kernel mueve los bytes, sin pasarlos por espacio de
// usuario), rutas extendidas \\?\ para saltarse MAX_PATH y un pool de workers
// que recorre y copia en paralelo. Sobrescribe el destino siempre, sin
// preguntar.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	workers := flag.Int("w", runtime.NumCPU()*2, "número de copias en paralelo")
	verbose := flag.Bool("v", false, "listar cada archivo copiado")
	quiet := flag.Bool("q", false, "sin progreso ni resumen")
	follow := flag.Bool("L", false, "seguir enlaces simbólicos y junctions (por defecto se omiten)")

	flag.Usage = func() {
		exe := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "Uso: %s [opciones] \"<directorio-origen>\" \"<directorio-destino>\"\n\n", exe)
		fmt.Fprintf(os.Stderr, "Copia recursivamente todo el contenido del origen al destino,\n")
		fmt.Fprintf(os.Stderr, "reemplazando lo que exista en destino sin preguntar.\n\n")
		fmt.Fprintf(os.Stderr, "Encierre las rutas entre comillas dobles si contienen espacios,\n")
		fmt.Fprintf(os.Stderr, "y NO deje una barra invertida final dentro de las comillas.\n\n")
		fmt.Fprintf(os.Stderr, "Ejemplos:\n")
		fmt.Fprintf(os.Stderr, "  %s \"D:\\Mis Datos\\origen\" \"E:\\Copia de seguridad\\destino\"\n", exe)
		fmt.Fprintf(os.Stderr, "  %s -w 4 \"\\\\servidor\\share\\datos\" \"D:\\local\"\n\n", exe)
		fmt.Fprintf(os.Stderr, "Opciones:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := validarNumeroDeArgumentos(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "UltraCopySam: %v\n\n", err)
		flag.Usage()
		os.Exit(2)
	}
	if *workers < 1 {
		*workers = 1
	}

	src, err := sanitizeArg(flag.Arg(0), "origen")
	if err != nil {
		fatal("%v", err)
	}
	dst, err := sanitizeArg(flag.Arg(1), "destino")
	if err != nil {
		fatal("%v", err)
	}
	if err := validarDirectorios(src, dst); err != nil {
		fatal("%v", err)
	}

	srcExt := extendedPath(src)
	dstExt := extendedPath(dst)

	if err := mkdirAll(dstExt); err != nil {
		fatal("no se puede crear el destino %s: %v", dst, err)
	}

	c := newCopier(*workers, *verbose, *follow)

	start := time.Now()
	var stopProgress chan struct{}
	if !*quiet && !*verbose {
		stopProgress = startProgress(c, start)
	}

	c.run(srcExt, dstExt)

	if stopProgress != nil {
		close(stopProgress)
	}

	elapsed := time.Since(start)
	if !*quiet {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 78))
		fmt.Printf("%d archivos, %d directorios, %s en %s (%s/s)\n",
			c.st.files.Load(), c.st.dirs.Load(), megaBytes(c.st.bytes.Load()),
			elapsed.Round(time.Millisecond), megaBytes(rate(c.st.bytes.Load(), elapsed)))
		if n := c.st.skipped.Load(); n > 0 {
			fmt.Printf("%d entradas omitidas (enlaces/junctions)\n", n)
		}
		if n := c.st.errors.Load(); n > 0 {
			fmt.Printf("%d errores\n", n)
		}
	}

	if c.st.errors.Load() > 0 {
		os.Exit(1)
	}
}

// startProgress imprime el avance en stderr; devuelve el canal para detenerlo.
func startProgress(c *copier, start time.Time) chan struct{} {
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				b := c.st.bytes.Load()
				fmt.Fprintf(os.Stderr, "\r%-77s",
					fmt.Sprintf("%d archivos | %s | %s/s",
						c.st.files.Load(), megaBytes(b), megaBytes(rate(b, time.Since(start)))))
			}
		}
	}()
	return stop
}

// mkdirAll crea el directorio y todos sus padres sobre una ruta extendida.
func mkdirAll(path string) error {
	if err := createDirectory(path); err == nil {
		return nil
	}
	i := strings.LastIndexByte(path, '\\')
	if i <= len(`\\?\`) {
		// Se llegó a la raíz del volumen: nada más que crear.
		return createDirectory(path)
	}
	if err := mkdirAll(path[:i]); err != nil {
		return err
	}
	return createDirectory(path)
}

// isInside indica si child cuelga de parent (comparación case-insensitive,
// como el sistema de archivos de Windows).
func isInside(child, parent string) bool {
	c := strings.ToLower(filepath.Clean(child))
	p := strings.ToLower(filepath.Clean(parent))
	return c == p || strings.HasPrefix(c, p+`\`)
}

func rate(bytes int64, d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64(float64(bytes) / d.Seconds())
}

// megaBytes formatea siempre en MB, sin escalar a GB. Mantener una unidad fija
// evita que la cifra del progreso salte de unidad mientras avanza la copia, que
// es lo que hace difícil comparar de un vistazo si va más rápido o más lento.
func megaBytes(n int64) string {
	return fmt.Sprintf("%.2f MB", float64(n)/(1024*1024))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "UltraCopySam: "+format+"\n", args...)
	os.Exit(2)
}
