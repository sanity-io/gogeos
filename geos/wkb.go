package geos

/*
#include "go-geos.h"
*/
import "C"

import (
	"encoding/hex"
	"runtime"
	"sync"
	"unsafe"
)

type wkbDecoder struct {
	r *C.GEOSWKBReader
	m sync.Mutex
}

func newWkbDecoder() *wkbDecoder {
	r := cGEOSWKBReader_create()
	d := &wkbDecoder{r: r}
	runtime.SetFinalizer(d, (*wkbDecoder).destroy)
	return d
}

func (d *wkbDecoder) destroy() {
	d.m.Lock()
	defer d.m.Unlock()

	cGEOSWKBReader_destroy(d.r)
	d.r = nil
}

func (d *wkbDecoder) decode(wkb []byte) (*Geometry, error) {
	d.m.Lock()
	defer d.m.Unlock()

	var cwkb []C.uchar
	for i := range wkb {
		cwkb = append(cwkb, C.uchar(wkb[i]))
	}
	g := cGEOSWKBReader_read(d.r, &cwkb[0], C.size_t(len(wkb)))
	if g == nil {
		return nil, Error()
	}
	return geomFromPtr(g), nil
}

func (d *wkbDecoder) decodeHex(wkbHex string) (*Geometry, error) {
	wkb, err := hex.DecodeString(wkbHex)
	if err != nil {
		return nil, err
	}
	return d.decode(wkb)
}

type wkbEncoder struct {
	w *C.GEOSWKBWriter
	m sync.Mutex
}

func newWkbEncoder() *wkbEncoder {
	w := cGEOSWKBWriter_create()
	if w == nil {
		return nil
	}
	e := &wkbEncoder{w: w}
	runtime.SetFinalizer(e, (*wkbEncoder).destroy)
	return e
}

func encodeWkb(e *wkbEncoder, g *Geometry, fn func(*C.GEOSWKBWriter, *C.GEOSGeometry, *C.size_t) *C.uchar) ([]byte, error) {
	e.m.Lock()
	defer e.m.Unlock()

	var size C.size_t
	bytes := fn(e.w, g.g, &size)
	if bytes == nil {
		return nil, Error()
	}
	ptr := unsafe.Pointer(bytes)
	defer C.free(ptr)
	l := int(size)
	var out []byte
	for i := 0; i < l; i++ {
		el := unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof(C.uchar(0))*uintptr(i))
		out = append(out, byte(*(*C.uchar)(el)))
	}
	return out, nil
}

func (e *wkbEncoder) encode(g *Geometry) ([]byte, error) {
	return encodeWkb(e, g, cGEOSWKBWriter_write)
}

func (e *wkbEncoder) encodeHex(g *Geometry) ([]byte, error) {
	return encodeWkb(e, g, cGEOSWKBWriter_writeHEX)
}

func (e *wkbEncoder) destroy() {
	e.m.Lock()
	defer e.m.Unlock()

	cGEOSWKBWriter_destroy(e.w)
	e.w = nil
}
