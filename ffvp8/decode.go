// Copyright (c) 2012, Jorge Acereda Maciá. All rights reserved.  
//
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE file.

// Package ffvp8 provides a wrapper around the VP8 codec in ffmpeg.
package ffvp8

// #cgo LDFLAGS: -lavcodec
//
// #include "libavcodec/avcodec.h"
// #include <string.h>
// extern AVCodec ff_vp8_decoder;
// static int get_buffer(AVCodecContext * cc, AVFrame * f) { 
//   void vp8GetBuffer(AVCodecContext * cc, AVFrame * f);
//   f->type = FF_BUFFER_TYPE_USER;
//   f->extended_data = f->data;
//   vp8GetBuffer(cc, f);
//   return 0;
// }
// static void release_buffer(AVCodecContext * cc, AVFrame * f) { 
//   void vp8ReleaseBuffer(AVCodecContext * cc, AVFrame * f);
//   vp8ReleaseBuffer(cc, f);
// }
// static void install_callbacks(AVCodecContext * cc) {
//   cc->get_buffer = get_buffer;
//   cc->reget_buffer = 0;
//   cc->release_buffer = release_buffer;
// }
import "C"

import (
	"container/list"
	"image"
	"log"
	"time"
	"unsafe"
)

type Frame struct {
	*image.YCbCr
	Timecode time.Duration
	dec      *Decoder
}

func (f *Frame) Release() {
	f.dec.free <- f
}

func init() {
	C.avcodec_register(&C.ff_vp8_decoder)
}

type Decoder struct {
	c    *C.AVCodec
	cc   *C.AVCodecContext
	busy list.List
	free chan *Frame
}

//export vp8GetBuffer
func vp8GetBuffer(cc *C.AVCodecContext, fr *C.AVFrame) {
	var d *Decoder
	d = (*Decoder)(cc.opaque)
	d.getBuffer(cc, fr)
}

//export vp8ReleaseBuffer
func vp8ReleaseBuffer(cc *C.AVCodecContext, fr *C.AVFrame) {
	var d *Decoder
	d = (*Decoder)(cc.opaque)
	d.releaseBuffer(cc, fr)
}

func aligned(x int) int {
	return (x+15)&-16 + 16
}

func (d *Decoder) allocFrame() *Frame {
	var frame *Frame
	select {
	case frame = <-d.free:
	default:
		w := int(d.cc.width)
		h := int(d.cc.height)
		aw := aligned(w)
		ah := aligned(h)
		acw := aligned(w / 2)
		ach := aligned(h / 2)
		ysz := aw * ah
		csz := acw * ach
		b := make([]byte, ysz+2*csz)
		frame = &Frame{
			&image.YCbCr{
				Y:              b[:ysz],
				Cb:             b[ysz : ysz+csz],
				Cr:             b[ysz+csz : ysz+2*csz],
				SubsampleRatio: image.YCbCrSubsampleRatio420,
				YStride:        aw,
				CStride:        acw,
				Rect:           image.Rect(0, 0, w, h),
			}, 0, d}
	}
	return frame
}

func (d *Decoder) getBuffer(cc *C.AVCodecContext, fr *C.AVFrame) {
	frame := d.allocFrame()
	fr.opaque = unsafe.Pointer(d.busy.PushBack(frame))
	img := frame.YCbCr
	fr.data[0] = (*C.uint8_t)(&img.Y[0])
	fr.data[1] = (*C.uint8_t)(&img.Cb[0])
	fr.data[2] = (*C.uint8_t)(&img.Cr[0])
	fr.linesize[0] = C.int(img.YStride)
	fr.linesize[1] = C.int(img.CStride)
	fr.linesize[2] = C.int(img.CStride)
	fr.width = C.int(cc.width)
	fr.height = C.int(cc.height)
	fr.format = C.int(cc.pix_fmt)
	fr.sample_aspect_ratio = cc.sample_aspect_ratio
	fr.pkt_pts = C.AV_NOPTS_VALUE
}

func (d *Decoder) releaseBuffer(cc *C.AVCodecContext, fr *C.AVFrame) {
	fr.data[0] = nil
	fr.data[1] = nil
	fr.data[2] = nil
	d.busy.Remove((*list.Element)(fr.opaque))
}

func NewDecoder() *Decoder {
	var d Decoder
	d.c = C.avcodec_find_decoder(C.AV_CODEC_ID_VP8)
	d.cc = C.avcodec_alloc_context3(d.c)
	d.cc.opaque = unsafe.Pointer(&d)
	C.install_callbacks(d.cc)
	C.avcodec_open2(d.cc, d.c, nil)
	d.free = make(chan *Frame, 8)
	return &d
}

func (d *Decoder) Decode(data []byte, tc time.Duration) *Frame {
	var pkt C.AVPacket
	var fr C.AVFrame
	var got C.int
	C.avcodec_get_frame_defaults(&fr)
	C.av_init_packet(&pkt)
	pkt.data = (*C.uint8_t)(&data[0])
	pkt.size = C.int(len(data))
	if C.avcodec_decode_video2(d.cc, &fr, &got, &pkt) < 0 || got == 0 {
		log.Panic("Unable to decode")
	}
	frame := (*list.Element)(fr.opaque).Value.(*Frame)
	frame.Timecode = tc
	return frame
}
