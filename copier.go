//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// noBufferingThreshold: a partir de este tamaño se copia sin pasar por la
// caché de archivos de Windows (evita ensuciar la caché y acelera los archivos
// grandes). Para archivos pequeños la caché sí ayuda.
const noBufferingThreshold = 32 << 20 // 32 MiB

// toleranciaFAT es el margen que hay que conceder al comparar fechas cuando el
// destino usa FAT o exFAT, que redondean la hora a 2 segundos. En NTFS la
// resolución es de 100 ns y la comparación debe ser EXACTA: cualquier margen
// haría pasar por idéntico un archivo modificado justo después de copiarlo, si
// además conserva el tamaño. Eso es pérdida de datos silenciosa.
const toleranciaFAT = 2 * time.Second

type stats struct {
	files      atomic.Int64
	dirs       atomic.Int64
	bytes      atomic.Int64
	skipped    atomic.Int64
	sinCambios atomic.Int64 // omitidos por -u: ya estaban iguales en destino
	ahorrados  atomic.Int64 // bytes que no hubo que reescribir gracias a -u
	errors     atomic.Int64
}

// walkersPorDefecto: workers dedicados a recorrer el árbol. Bastan muy pocos
// porque listar directorios es dos órdenes de magnitud más rápido que copiar.
const walkersPorDefecto = 4

type copier struct {
	dirs    *colaDirs
	st      stats
	workers int // copias en paralelo (-w)
	buffer  int // archivos que pueden esperar en cola; acota la memoria
	verbose bool
	follow  bool // seguir symlinks/junctions en vez de omitirlos
	update  bool // saltar los archivos que ya estén iguales en destino

	// tolerancia al comparar fechas en modo -u. Cero (exacta) salvo que el
	// destino sea FAT/exFAT.
	tolerancia time.Duration

	errMu   sync.Mutex
	errShow int
}

// entradaDestino son los datos de un archivo ya presente en destino que se
// necesitan para decidir si hay que reescribirlo.
type entradaDestino struct {
	tamano       int64
	modificacion time.Time
}

// newCopier construye el motor de copia. fsDestino es el nombre del sistema de
// archivos del volumen de destino ("NTFS", "exFAT"...), del que depende la
// precisión con la que se pueden comparar fechas en modo -u.
func newCopier(workers, buffer int, verbose, follow, update bool, fsDestino string) *copier {
	tolerancia := time.Duration(0) // exacta: lo correcto en NTFS
	if esFAT(fsDestino) {
		tolerancia = toleranciaFAT
	}

	return &copier{
		dirs:       nuevaColaDirs(),
		workers:    workers,
		buffer:     buffer,
		verbose:    verbose,
		follow:     follow,
		update:     update,
		tolerancia: tolerancia,
	}
}

// esFAT indica si el sistema de archivos redondea las fechas a 2 segundos.
// Ante un nombre desconocido o vacío se asume que no, porque un margen de más
// puede ocultar cambios reales y la comparación exacta solo provoca copias de
// más, que es el error inocuo.
func esFAT(nombre string) bool {
	return strings.HasPrefix(strings.ToUpper(nombre), "FAT") ||
		strings.EqualFold(nombre, "exFAT")
}

// run arranca los dos pools y bloquea hasta que se copia el árbol entero.
//
// Recorrido y copia se reparten en pools separados sobre colas distintas. El
// motivo es la memoria: listar directorios es unas 300 veces más rápido que
// copiar, así que un recorrido sin freno llena la cola con el árbol completo
// antes de copiar nada, y el consumo crece con el número de archivos. Aquí la
// cola de archivos tiene capacidad fija y el recorrido se bloquea cuando está
// llena, lo que pone un techo a la memoria.
//
// La separación es además lo que evita un bloqueo mutuo: los workers de copia
// solo consumen de la cola de archivos y nunca escriben en ella, de modo que
// siempre hay quien la drene y el recorrido acaba desbloqueándose. Con un único
// pool, todos los workers podrían quedarse bloqueados encolando y no quedaría
// nadie para vaciar.
func (c *copier) run(src, dst string) {
	c.dirs.push(&carpeta{src: src, dst: dst})

	archivos := make(chan archivoJob, c.buffer)

	var recorrido sync.WaitGroup
	recorrido.Add(walkersPorDefecto)
	for i := 0; i < walkersPorDefecto; i++ {
		go func() {
			defer recorrido.Done()
			for {
				dir, ok := c.dirs.pop()
				if !ok {
					return
				}
				c.walkDir(dir, archivos)
				c.dirs.done()
			}
		}()
	}

	var copia sync.WaitGroup
	copia.Add(c.workers)
	for i := 0; i < c.workers; i++ {
		go func() {
			defer copia.Done()
			for j := range archivos {
				c.copyOne(join(j.dir.src, j.name), join(j.dir.dst, j.name), j.size)
			}
		}()
	}

	recorrido.Wait()
	close(archivos) // ya no entrarán más archivos; los copiadores drenan y salen
	copia.Wait()
}

// walkDir recorre un directorio: crea su equivalente en destino, encola los
// subdirectorios y envía sus archivos al canal de copia. El envío puede
// bloquearse si el canal está lleno, y eso es intencionado: es el freno que
// impide que el recorrido se adelante sin límite y dispare la memoria.
func (c *copier) walkDir(dir *carpeta, archivos chan<- archivoJob) {
	yaExistia, err := createDirectory(dir.dst)
	if err != nil {
		c.reportErr(dir.dst, fmt.Errorf("crear directorio: %w", err))
		return
	}
	c.st.dirs.Add(1)

	f, err := os.Open(dir.src)
	if err != nil {
		c.reportErr(dir.src, fmt.Errorf("abrir directorio: %w", err))
		return
	}
	entries, err := f.ReadDir(-1) // -1: sin ordenar, es el camino más rápido
	f.Close()
	if err != nil {
		c.reportErr(dir.src, fmt.Errorf("listar directorio: %w", err))
		return
	}

	// En modo -u se necesita saber qué hay ya en destino. Se resuelve con un
	// único listado por carpeta, no con un stat por archivo. Si la carpeta
	// acabamos de crearla está vacía y no hay nada que consultar.
	var existentes map[string]entradaDestino
	if c.update && yaExistia {
		existentes = leerDestino(dir.dst)
	}

	for _, e := range entries {
		name := e.Name()

		if !c.follow && e.Type()&os.ModeSymlink != 0 {
			c.st.skipped.Add(1)
			if c.verbose {
				fmt.Fprintf(os.Stderr, "omitido (enlace/junction): %s\n",
					displayPath(join(dir.src, name)))
			}
			continue
		}

		if e.IsDir() {
			c.dirs.push(&carpeta{src: join(dir.src, name), dst: join(dir.dst, name)})
			continue
		}

		var size int64
		var modificacion time.Time
		if info, err := e.Info(); err == nil {
			size = info.Size()
			modificacion = info.ModTime()
		}

		if existentes != nil && c.sinCambios(existentes[strings.ToLower(name)], size, modificacion) {
			c.st.sinCambios.Add(1)
			c.st.ahorrados.Add(size)
			if c.verbose {
				fmt.Fprintf(os.Stderr, "sin cambios: %s\n", displayPath(join(dir.dst, name)))
			}
			continue
		}

		// Solo el nombre viaja en la cola; dir se comparte con el resto de
		// archivos de esta misma carpeta.
		archivos <- archivoJob{dir: dir, name: name, size: size}
	}
}

// leerDestino indexa el contenido de una carpeta de destino por nombre en
// minúsculas, porque el sistema de archivos de Windows no distingue mayúsculas.
// Si la carpeta no se puede listar devuelve nil y todo se copiará: ante la duda,
// copiar.
func leerDestino(dst string) map[string]entradaDestino {
	f, err := os.Open(dst)
	if err != nil {
		return nil
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return nil
	}

	m := make(map[string]entradaDestino, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		m[strings.ToLower(e.Name())] = entradaDestino{
			tamano:       info.Size(),
			modificacion: info.ModTime(),
		}
	}
	return m
}

// sinCambios decide si el archivo de destino puede darse por idéntico al de
// origen. Solo lo afirma cuando coinciden tamaño y fecha; cualquier otra
// situación —incluido que el archivo no exista, que es el valor cero— devuelve
// false y provoca la copia.
func (c *copier) sinCambios(destino entradaDestino, tamano int64, modificacion time.Time) bool {
	if destino.modificacion.IsZero() || destino.tamano != tamano {
		return false
	}
	if c.tolerancia == 0 {
		return destino.modificacion.Equal(modificacion)
	}
	diferencia := destino.modificacion.Sub(modificacion)
	if diferencia < 0 {
		diferencia = -diferencia
	}
	return diferencia <= c.tolerancia
}

func (c *copier) copyOne(src, dst string, size int64) {
	var flags uint32
	if size >= noBufferingThreshold {
		flags |= copyFileNoBuffering
	}
	flags |= copyFileAllowDecryptedDest

	err := copyFileEx(src, dst, flags)

	// Algunos volúmenes (red, ciertos filesystems) rechazan NO_BUFFERING.
	if err != nil && flags&copyFileNoBuffering != 0 && isUnsupported(err) {
		flags &^= copyFileNoBuffering
		err = copyFileEx(src, dst, flags)
	}

	// Destino de solo lectura / oculto / sistema: se limpia y se reintenta,
	// porque el requisito es reemplazar sin preguntar.
	if err != nil && isAccessDenied(err) {
		if clearBlockingAttrs(dst) == nil {
			err = copyFileEx(src, dst, flags)
		}
	}

	if err != nil {
		c.reportErr(src, err)
		return
	}

	c.st.files.Add(1)
	c.st.bytes.Add(size)
	if c.verbose {
		fmt.Fprintln(os.Stderr, displayPath(dst))
	}
}

func isAccessDenied(err error) bool {
	errno, ok := err.(syscall.Errno)
	return ok && errno == errorAccessDenied
}

func isUnsupported(err error) bool {
	errno, ok := err.(syscall.Errno)
	return ok && (errno == errorInvalidParam || errno == errorNotSupported || errno == errorInvalidFunc)
}

// reportErr contabiliza el fallo y lo imprime, pero no aborta: la copia
// continúa con el resto del árbol.
func (c *copier) reportErr(path string, err error) {
	c.st.errors.Add(1)
	c.errMu.Lock()
	defer c.errMu.Unlock()
	if c.errShow >= 50 && !c.verbose {
		if c.errShow == 50 {
			fmt.Fprintln(os.Stderr, "... más errores omitidos (use -v para verlos todos)")
			c.errShow++
		}
		return
	}
	c.errShow++
	fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", displayPath(path), err)
}
