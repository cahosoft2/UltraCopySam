//go:build windows

package main

import "sync"

// job es una unidad de trabajo: copiar un archivo o recorrer un directorio.
type job struct {
	src   string
	dst   string
	isDir bool
	size  int64
}

// queue es una cola de trabajo con contador de pendientes. Los workers pueden
// encolar nuevos jobs mientras consumen (el recorrido descubre subdirectorios
// sobre la marcha); la cola se cierra sola cuando no queda nada pendiente.
// Se usa cola explícita en vez de una goroutine por entrada para acotar la
// memoria en árboles con millones de archivos.
type queue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	items   []job
	pending int
	closed  bool
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(j job) {
	q.mu.Lock()
	q.items = append(q.items, j)
	q.pending++
	q.mu.Unlock()
	q.cond.Signal()
}

// pop devuelve el siguiente job, o ok=false cuando ya no queda trabajo.
func (q *queue) pop() (job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return job{}, false
	}
	last := len(q.items) - 1
	j := q.items[last]
	q.items = q.items[:last] // LIFO: mejora la localidad al recorrer el árbol
	return j, true
}

// done marca un job como terminado; al llegar a cero se despiertan todos los
// workers para que salgan.
func (q *queue) done() {
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
