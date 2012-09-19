package ffvp8

// #cgo darwin CFLAGS: -I/Users/jacereda/ffmpeg/b/include
// #cgo darwin LDFLAGS: -L/Users/jacereda/ffmpeg/b/lib -lavcodec
//
// #include "libavcodec/avcodec.h"
// extern AVCodec ff_vp8_decoder;
import "C"

import(
	"image"
        "unsafe"
        "log"
)

func init() {
//	C.avcodec_register_all()
	C.avcodec_register(&C.ff_vp8_decoder)
}

type Decoder struct {
	c *C.AVCodec
	cc *C.AVCodecContext
}

func NewDecoder() (*Decoder) {
	var d Decoder
	d.c = C.avcodec_find_decoder(C.AV_CODEC_ID_VP8)
	d.cc = C.avcodec_alloc_context3(d.c)
	C.avcodec_open2(d.cc, d.c, nil);
	return &d
}

func mkslice(p *C.uint8_t, sz C.int) []byte {
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(unsafe.Pointer(p)), int(sz), int(sz)}
	return *(*[]byte)(unsafe.Pointer(&sl))
}

func (d *Decoder) Decode(data []byte) (*image.YCbCr) {
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
	return &image.YCbCr{
		Y:              mkslice(fr.data[0], fr.linesize[0] * fr.height),
		Cb:             mkslice(fr.data[1], fr.linesize[1] * fr.height),
		Cr:             mkslice(fr.data[2], fr.linesize[2] * fr.height),
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		YStride:        int(fr.linesize[0]),
		CStride:        int(fr.linesize[1]),
		Rect:           image.Rect(0, 0, int(fr.width), int(fr.height)),
	}
}
