package ffvp8

// #cgo darwin CFLAGS: -I/Users/jacereda/ffmpeg/b/include
// #cgo darwin LDFLAGS: -L/Users/jacereda/ffmpeg/b/lib
// #cgo LDFLAGS: -lavcodec
//
// #include "libavcodec/avcodec.h"
// extern AVCodec ff_vp8_decoder;
// extern void ff_init_buffer_info(AVCodecContext *s, AVFrame *frame);
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
//   cc->release_buffer = release_buffer;
// }
import "C"

import (
	"container/list"
	"image"
	"log"
	"unsafe"
)

func init() {
	//	C.avcodec_register_all()
	C.avcodec_register(&C.ff_vp8_decoder)
}

type Decoder struct {
	c    *C.AVCodec
	cc   *C.AVCodecContext
	imgs list.List
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

func (d *Decoder) getBuffer(cc *C.AVCodecContext, fr *C.AVFrame) {
	w := int(cc.width)
	h := int(cc.height)
	aw := w + 16
	ah := h + 16
	acw := aw / 2
	ach := ah / 2
	b := make([]byte, aw*ah+2*acw*ach)
	img := &image.YCbCr{
		Y:              b[:aw*ah],
		Cb:             b[aw*ah : aw*ah+1*acw*ach],
		Cr:             b[aw*ah+1*acw*ach : aw*ah+2*acw*ach],
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		YStride:        aw,
		CStride:        acw,
		Rect:           image.Rect(0, 0, w, h),
	}
	e := d.imgs.PushBack(img)
	fr.data[0] = (*C.uint8_t)(&img.Y[0])
	fr.data[1] = (*C.uint8_t)(&img.Cb[0])
	fr.data[2] = (*C.uint8_t)(&img.Cr[0])
	fr.linesize[0] = C.int(img.YStride)
	fr.linesize[1] = C.int(img.CStride)
	fr.linesize[2] = C.int(img.CStride)
	C.ff_init_buffer_info(cc, fr)
	fr.opaque = unsafe.Pointer(e)
}

func (d *Decoder) releaseBuffer(cc *C.AVCodecContext, fr *C.AVFrame) {
	var e *list.Element
	e = (*list.Element)(fr.opaque)
	d.imgs.Remove(e)
}

func NewDecoder() *Decoder {
	var d Decoder
	d.c = C.avcodec_find_decoder(C.AV_CODEC_ID_VP8)
	d.cc = C.avcodec_alloc_context3(d.c)
	d.cc.opaque = unsafe.Pointer(&d)
	C.install_callbacks(d.cc)
	C.avcodec_open2(d.cc, d.c, nil)
	return &d
}

func (d *Decoder) Decode(data []byte) *image.YCbCr {
	var pkt C.AVPacket
	var fr C.AVFrame
	var got C.int
	C.avcodec_get_frame_defaults(&fr)
	C.av_init_packet(&pkt)
	pkt.data = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	pkt.size = C.int(len(data))
	if C.avcodec_decode_video2(d.cc, &fr, &got, &pkt) < 0 {
		log.Panic("Unable to decode")
	}
	if got == 0 {
		log.Panic("Unable to decode")
	}
	return ((*list.Element)(unsafe.Pointer(fr.opaque))).Value.(*image.YCbCr)
}
