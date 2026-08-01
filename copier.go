//go:build windows

package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
)

// noBufferingThreshold: a partir de este tamaño se copia sin pasar por la
// caché de archivos de Windows (evita ensuciar la caché y acelera los archivos
// grandes). Para archivos pequeños la caché sí ayuda.
const noBufferingThreshold = 32 << 20 // 32 MiB

type stats struct {
	files   atomic.Int64
	dirs    atomic.Int64
	bytes   atomic.Int64
	skipped atomic.Int64
	errors  atomic.Int64
}

type copier struct {
	q       *queue
	st      stats
	workers int
	verbose bool
	follow  bool // seguir symlinks/junctions en vez de omitirlos

	errMu   sync.Mutex
	errShow int
}

func newCopier(workers int, verbose, follow bool) *copier {
	return &copier{q: newQueue(), workers: workers, verbose: verbose, follow: follow}
}

// run arranca el pool y bloquea hasta que se vacía el árbol.
func (c *copier) run(src, dst string) {
	c.q.push(job{src: src, dst: dst, isDir: true})

	var wg sync.WaitGroup
	wg.Add(c.workers)
	for i := 0; i < c.workers; i++ {
		go func() {
			defer wg.Done()
			for {
				j, ok := c.q.pop()
				if !ok {
					return
				}
				if j.isDir {
					c.walkDir(j.src, j.dst)
				} else {
					c.copyOne(j.src, j.dst, j.size)
				}
				c.q.done()
			}
		}()
	}
	wg.Wait()
}

func (c *copier) walkDir(src, dst string) {
	if err := createDirectory(dst); err != nil {
		c.reportErr(dst, fmt.Errorf("crear directorio: %w", err))
		return
	}
	c.st.dirs.Add(1)

	f, err := os.Open(src)
	if err != nil {
		c.reportErr(src, fmt.Errorf("abrir directorio: %w", err))
		return
	}
	entries, err := f.ReadDir(-1) // -1: sin ordenar, es el camino más rápido
	f.Close()
	if err != nil {
		c.reportErr(src, fmt.Errorf("listar directorio: %w", err))
		return
	}

	for _, e := range entries {
		name := e.Name()
		childSrc := join(src, name)
		childDst := join(dst, name)

		if !c.follow && e.Type()&os.ModeSymlink != 0 {
			c.st.skipped.Add(1)
			if c.verbose {
				fmt.Fprintf(os.Stderr, "omitido (enlace/junction): %s\n", displayPath(childSrc))
			}
			continue
		}

		if e.IsDir() {
			c.q.push(job{src: childSrc, dst: childDst, isDir: true})
			continue
		}

		var size int64
		if info, err := e.Info(); err == nil {
			size = info.Size()
		}
		c.q.push(job{src: childSrc, dst: childDst, size: size})
	}
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
