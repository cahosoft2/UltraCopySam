//go:build windows

package main

import "sync"

// carpeta es un par de rutas origen/destino ya compuestas. Se comparte por
// puntero entre todos los archivos que contiene, de modo que el texto de la
// ruta se almacena una vez por directorio y no una vez por archivo. En un árbol
// de millones de archivos esa diferencia es la mayor parte de la memoria usada.
type carpeta struct {
	src string
	dst string
}

// archivoJob es un archivo pendiente de copiar. Guarda solo su nombre; las
// rutas completas se componen al copiar, con join.
type archivoJob struct {
	dir  *carpeta
	name string
	size int64
}

// colaDirs es la cola de directorios pendientes de recorrer, con contador de
// pendientes. Los workers de recorrido pueden encolar mientras consumen (cada
// directorio descubre subdirectorios), y la cola se cierra sola cuando no queda
// nada pendiente.
//
// No se acota: los directorios son dos órdenes de magnitud menos numerosos que
// los archivos. La cola que sí se acota es la de archivos, que vive en copier.
type colaDirs struct {
	mu      sync.Mutex
	cond    *sync.Cond
	items   []*carpeta
	pending int
	closed  bool
}

func nuevaColaDirs() *colaDirs {
	q := &colaDirs{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *colaDirs) push(c *carpeta) {
	q.mu.Lock()
	q.items = append(q.items, c)
	q.pending++
	q.mu.Unlock()
	q.cond.Signal()
}

// pop devuelve el siguiente directorio, o ok=false cuando ya no queda trabajo.
func (q *colaDirs) pop() (*carpeta, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return nil, false
	}
	last := len(q.items) - 1
	c := q.items[last]
	q.items = q.items[:last] // LIFO: mejora la localidad al recorrer el árbol
	return c, true
}

// done marca un directorio como terminado; al llegar a cero se despiertan todos
// los workers para que salgan.
func (q *colaDirs) done() {
	q.mu.Lock()
	q.pending--
	if q.pending == 0 {
		q.closed = true
		q.mu.Unlock()
		q.cond.Broadcast()
		return
	}
	q.mu.Unlock()
}
