// Copyright (c) 2012, Jorge Acereda Maciá. All rights reserved.  
//
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE file.

// Package ffvp8 provides a wrapper around the VP8 codec in ffmpeg.
package ffvp8

// #cgo LDFLAGS: -lavcodec -lavutil
//
// #include "libavcodec/avcodec.h"
// #include "libavutil/frame.h"
// #if LIBAVCODEC_VERSION_MAJOR == 53
// #define AV_CODEC_ID_VP8 CODEC_ID_VP8
// #endif
import "C"

import (
	"container/list"
	"image"
	"log"
	"reflect"
	"unsafe"
)

func dupPlane(src []byte, stride, w, h int) []byte {
	dst := make([]byte, w*h)
	for i := 0; i < h; i++ {
		drow := i * w
		srow := i * stride
		copy(dst[drow:drow+w], src[srow:srow+w])
	}
	return dst
}

func dup(f *image.YCbCr) {
	w := f.Rect.Dx()
	h := f.Rect.Dy()
	cw := (w + 1) / 2
	ch := (h + 1) / 2
	f.Y = dupPlane(f.Y, f.YStride, w, h)
	f.Cb = dupPlane(f.Cb, f.CStride, cw, ch)
	f.Cr = dupPlane(f.Cr, f.CStride, cw, ch)
	f.YStride = w
	f.CStride = cw
}

func init() {
	C.avcodec_register_all()
}

type Decoder struct {
	c    *C.AVCodec
	cc   *C.AVCodecContext
	imgs list.List
}

func NewDecoder() *Decoder {
	var d Decoder
	d.c = C.avcodec_find_decoder(C.AV_CODEC_ID_VP8)
	d.cc = C.avcodec_alloc_context3(d.c)
	d.cc.opaque = unsafe.Pointer(&d)
	C.avcodec_open2(d.cc, d.c, nil)
	return &d
}

func mkslice(p *C.uint8_t, sz int) []byte {
	var slice []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	hdr.Cap = sz
	hdr.Len = sz
	hdr.Data = uintptr(unsafe.Pointer(p))
	return slice
}

func (d *Decoder) Decode(data []byte) *image.YCbCr {
	var pkt C.AVPacket
	var fr *C.AVFrame
	var got C.int
	fr = C.av_frame_alloc()
	defer C.av_frame_free(&fr)
	C.av_init_packet(&pkt)
	pkt.data = (*C.uint8_t)(&data[0])
	pkt.size = C.int(len(data))
	if C.avcodec_decode_video2(d.cc, fr, &got, &pkt) < 0 {
		log.Panic("Unable to decode")
	}
	if got == 0 {
		return nil
	}
	ys := int(fr.linesize[0])
	cs := int(fr.linesize[1])
	yw := int(d.cc.width)
	yh := int(d.cc.height)
	ysz := ys * yh
	csz := cs * yh / 2
	img := &image.YCbCr{
		Y:              mkslice(fr.data[0], ysz),
		Cb:             mkslice(fr.data[1], csz),
		Cr:             mkslice(fr.data[2], csz),
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		YStride:        ys,
		CStride:        cs,
		Rect:           image.Rect(0, 0, yw, yh),
	}
	dup(img)
	return img
}

func (d *Decoder) Flush() {
	C.avcodec_flush_buffers(d.cc)
}
