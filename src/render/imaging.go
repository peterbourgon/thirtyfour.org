package main

// Code taken from https://github.com/disintegration/imaging
// commit 0bd5694c78c9c3d9a3cd06a706a8f3c59296a9ac
// to avoid dependencies.

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

func resize(img image.Image, width, height int, filter resampleFilter) *image.NRGBA {
	dstW, dstH := width, height
	if dstW < 0 || dstH < 0 {
		return &image.NRGBA{}
	}
	if dstW == 0 && dstH == 0 {
		return &image.NRGBA{}
	}

	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	if srcW <= 0 || srcH <= 0 {
		return &image.NRGBA{}
	}

	// If new width or height is 0 then preserve aspect ratio, minimum 1px.
	if dstW == 0 {
		tmpW := float64(dstH) * float64(srcW) / float64(srcH)
		dstW = int(math.Max(1.0, math.Floor(tmpW+0.5)))
	}
	if dstH == 0 {
		tmpH := float64(dstW) * float64(srcH) / float64(srcW)
		dstH = int(math.Max(1.0, math.Floor(tmpH+0.5)))
	}

	if filter.Support <= 0 {
		// Nearest-neighbor special case.
		return resizeNearest(img, dstW, dstH)
	}

	if srcW != dstW && srcH != dstH {
		return resizeVertical(resizeHorizontal(img, dstW, filter), dstH, filter)
	}
	if srcW != dstW {
		return resizeHorizontal(img, dstW, filter)
	}
	if srcH != dstH {
		return resizeVertical(img, dstH, filter)
	}
	return clone(img)
}

type resampleFilter struct {
	Support float64
	Kernel  func(float64) float64
}

var lanczos = resampleFilter{
	Support: 3.0,
	Kernel: func(x float64) float64 {
		x = math.Abs(x)
		if x < 3.0 {
			return sinc(x) * sinc(x/3.0)
		}
		return 0
	},
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(math.Pi*x) / (math.Pi * x)
}

func resizeHorizontal(img image.Image, width int, filter resampleFilter) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, width, src.h))
	weights := precomputeWeights(width, src.w, filter)
	parallel(0, src.h, func(ys <-chan int) {
		scanLine := make([]uint8, src.w*4)
		for y := range ys {
			src.scan(0, y, src.w, y+1, scanLine)
			j0 := y * dst.Stride
			for x := 0; x < width; x++ {
				var r, g, b, a float64
				for _, w := range weights[x] {
					i := w.index * 4
					aw := float64(scanLine[i+3]) * w.weight
					r += float64(scanLine[i+0]) * aw
					g += float64(scanLine[i+1]) * aw
					b += float64(scanLine[i+2]) * aw
					a += aw
				}
				if a != 0 {
					aInv := 1 / a
					j := j0 + x*4
					dst.Pix[j+0] = clamp(r * aInv)
					dst.Pix[j+1] = clamp(g * aInv)
					dst.Pix[j+2] = clamp(b * aInv)
					dst.Pix[j+3] = clamp(a)
				}
			}
		}
	})
	return dst
}

func resizeVertical(img image.Image, height int, filter resampleFilter) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, height))
	weights := precomputeWeights(height, src.h, filter)
	parallel(0, src.w, func(xs <-chan int) {
		scanLine := make([]uint8, src.h*4)
		for x := range xs {
			src.scan(x, 0, x+1, src.h, scanLine)
			for y := 0; y < height; y++ {
				var r, g, b, a float64
				for _, w := range weights[y] {
					i := w.index * 4
					aw := float64(scanLine[i+3]) * w.weight
					r += float64(scanLine[i+0]) * aw
					g += float64(scanLine[i+1]) * aw
					b += float64(scanLine[i+2]) * aw
					a += aw
				}
				if a != 0 {
					aInv := 1 / a
					j := y*dst.Stride + x*4
					dst.Pix[j+0] = clamp(r * aInv)
					dst.Pix[j+1] = clamp(g * aInv)
					dst.Pix[j+2] = clamp(b * aInv)
					dst.Pix[j+3] = clamp(a)
				}
			}
		}
	})
	return dst
}

// resizeNearest is a fast nearest-neighbor resize, no filtering.
func resizeNearest(img image.Image, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	dx := float64(img.Bounds().Dx()) / float64(width)
	dy := float64(img.Bounds().Dy()) / float64(height)

	if dx > 1 && dy > 1 {
		src := newScanner(img)
		parallel(0, height, func(ys <-chan int) {
			for y := range ys {
				srcY := int((float64(y) + 0.5) * dy)
				dstOff := y * dst.Stride
				for x := 0; x < width; x++ {
					srcX := int((float64(x) + 0.5) * dx)
					src.scan(srcX, srcY, srcX+1, srcY+1, dst.Pix[dstOff:dstOff+4])
					dstOff += 4
				}
			}
		})
	} else {
		src := toNRGBA(img)
		parallel(0, height, func(ys <-chan int) {
			for y := range ys {
				srcY := int((float64(y) + 0.5) * dy)
				srcOff0 := srcY * src.Stride
				dstOff := y * dst.Stride
				for x := 0; x < width; x++ {
					srcX := int((float64(x) + 0.5) * dx)
					srcOff := srcOff0 + srcX*4
					copy(dst.Pix[dstOff:dstOff+4], src.Pix[srcOff:srcOff+4])
					dstOff += 4
				}
			}
		})
	}

	return dst
}

func clone(img image.Image) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	size := src.w * 4
	parallel(0, src.h, func(ys <-chan int) {
		for y := range ys {
			i := y * dst.Stride
			src.scan(0, y, src.w, y+1, dst.Pix[i:i+size])
		}
	})
	return dst
}

type scanner struct {
	image   image.Image
	w, h    int
	palette []color.NRGBA
}

func newScanner(img image.Image) *scanner {
	s := &scanner{
		image: img,
		w:     img.Bounds().Dx(),
		h:     img.Bounds().Dy(),
	}
	if img, ok := img.(*image.Paletted); ok {
		s.palette = make([]color.NRGBA, len(img.Palette))
		for i := 0; i < len(img.Palette); i++ {
			s.palette[i] = color.NRGBAModel.Convert(img.Palette[i]).(color.NRGBA)
		}
	}
	return s
}

func (s *scanner) scan(x1, y1, x2, y2 int, dst []uint8) {
	switch img := s.image.(type) {
	case *image.NRGBA:
		size := (x2 - x1) * 4
		j := 0
		i := y1*img.Stride + x1*4
		for y := y1; y < y2; y++ {
			copy(dst[j:j+size], img.Pix[i:i+size])
			j += size
			i += img.Stride
		}

	case *image.NRGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				dst[j+0] = img.Pix[i+0]
				dst[j+1] = img.Pix[i+2]
				dst[j+2] = img.Pix[i+4]
				dst[j+3] = img.Pix[i+6]
				j += 4
				i += 8
			}
		}

	case *image.RGBA:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*4
			for x := x1; x < x2; x++ {
				a := img.Pix[i+3]
				switch a {
				case 0:
					dst[j+0] = 0
					dst[j+1] = 0
					dst[j+2] = 0
				case 0xff:
					dst[j+0] = img.Pix[i+0]
					dst[j+1] = img.Pix[i+1]
					dst[j+2] = img.Pix[i+2]
				default:
					r16 := uint16(img.Pix[i+0])
					g16 := uint16(img.Pix[i+1])
					b16 := uint16(img.Pix[i+2])
					a16 := uint16(a)
					dst[j+0] = uint8(r16 * 0xff / a16)
					dst[j+1] = uint8(g16 * 0xff / a16)
					dst[j+2] = uint8(b16 * 0xff / a16)
				}
				dst[j+3] = a
				j += 4
				i += 4
			}
		}

	case *image.RGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				a := img.Pix[i+6]
				switch a {
				case 0:
					dst[j+0] = 0
					dst[j+1] = 0
					dst[j+2] = 0
				case 0xff:
					dst[j+0] = img.Pix[i+0]
					dst[j+1] = img.Pix[i+2]
					dst[j+2] = img.Pix[i+4]
				default:
					r32 := uint32(img.Pix[i+0])<<8 | uint32(img.Pix[i+1])
					g32 := uint32(img.Pix[i+2])<<8 | uint32(img.Pix[i+3])
					b32 := uint32(img.Pix[i+4])<<8 | uint32(img.Pix[i+5])
					a32 := uint32(img.Pix[i+6])<<8 | uint32(img.Pix[i+7])
					dst[j+0] = uint8((r32 * 0xffff / a32) >> 8)
					dst[j+1] = uint8((g32 * 0xffff / a32) >> 8)
					dst[j+2] = uint8((b32 * 0xffff / a32) >> 8)
				}
				dst[j+3] = a
				j += 4
				i += 8
			}
		}

	case *image.Gray:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				dst[j+0] = c
				dst[j+1] = c
				dst[j+2] = c
				dst[j+3] = 0xff
				j += 4
				i++
			}
		}

	case *image.Gray16:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*2
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				dst[j+0] = c
				dst[j+1] = c
				dst[j+2] = c
				dst[j+3] = 0xff
				j += 4
				i += 2
			}
		}

	case *image.YCbCr:
		j := 0
		x1 += img.Rect.Min.X
		x2 += img.Rect.Min.X
		y1 += img.Rect.Min.Y
		y2 += img.Rect.Min.Y
		for y := y1; y < y2; y++ {
			iy := (y-img.Rect.Min.Y)*img.YStride + (x1 - img.Rect.Min.X)
			for x := x1; x < x2; x++ {
				var ic int
				switch img.SubsampleRatio {
				case image.YCbCrSubsampleRatio444:
					ic = (y-img.Rect.Min.Y)*img.CStride + (x - img.Rect.Min.X)
				case image.YCbCrSubsampleRatio422:
					ic = (y-img.Rect.Min.Y)*img.CStride + (x/2 - img.Rect.Min.X/2)
				case image.YCbCrSubsampleRatio420:
					ic = (y/2-img.Rect.Min.Y/2)*img.CStride + (x/2 - img.Rect.Min.X/2)
				case image.YCbCrSubsampleRatio440:
					ic = (y/2-img.Rect.Min.Y/2)*img.CStride + (x - img.Rect.Min.X)
				default:
					ic = img.COffset(x, y)
				}

				yy := int(img.Y[iy])
				cb := int(img.Cb[ic]) - 128
				cr := int(img.Cr[ic]) - 128

				r := (yy<<16 + 91881*cr + 1<<15) >> 16
				if r > 0xff {
					r = 0xff
				} else if r < 0 {
					r = 0
				}

				g := (yy<<16 - 22554*cb - 46802*cr + 1<<15) >> 16
				if g > 0xff {
					g = 0xff
				} else if g < 0 {
					g = 0
				}

				b := (yy<<16 + 116130*cb + 1<<15) >> 16
				if b > 0xff {
					b = 0xff
				} else if b < 0 {
					b = 0
				}

				dst[j+0] = uint8(r)
				dst[j+1] = uint8(g)
				dst[j+2] = uint8(b)
				dst[j+3] = 0xff

				iy++
				j += 4
			}
		}

	case *image.Paletted:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := s.palette[img.Pix[i]]
				dst[j+0] = c.R
				dst[j+1] = c.G
				dst[j+2] = c.B
				dst[j+3] = c.A
				j += 4
				i++
			}
		}

	default:
		j := 0
		b := s.image.Bounds()
		x1 += b.Min.X
		x2 += b.Min.X
		y1 += b.Min.Y
		y2 += b.Min.Y
		for y := y1; y < y2; y++ {
			for x := x1; x < x2; x++ {
				r16, g16, b16, a16 := s.image.At(x, y).RGBA()
				switch a16 {
				case 0xffff:
					dst[j+0] = uint8(r16 >> 8)
					dst[j+1] = uint8(g16 >> 8)
					dst[j+2] = uint8(b16 >> 8)
					dst[j+3] = 0xff
				case 0:
					dst[j+0] = 0
					dst[j+1] = 0
					dst[j+2] = 0
					dst[j+3] = 0
				default:
					dst[j+0] = uint8(((r16 * 0xffff) / a16) >> 8)
					dst[j+1] = uint8(((g16 * 0xffff) / a16) >> 8)
					dst[j+2] = uint8(((b16 * 0xffff) / a16) >> 8)
					dst[j+3] = uint8(a16 >> 8)
				}
				j += 4
			}
		}
	}
}

type indexWeight struct {
	index  int
	weight float64
}

func precomputeWeights(dstSize, srcSize int, filter resampleFilter) [][]indexWeight {
	du := float64(srcSize) / float64(dstSize)
	scale := du
	if scale < 1.0 {
		scale = 1.0
	}
	ru := math.Ceil(scale * filter.Support)

	out := make([][]indexWeight, dstSize)
	tmp := make([]indexWeight, 0, dstSize*int(ru+2)*2)

	for v := 0; v < dstSize; v++ {
		fu := (float64(v)+0.5)*du - 0.5

		begin := int(math.Ceil(fu - ru))
		if begin < 0 {
			begin = 0
		}
		end := int(math.Floor(fu + ru))
		if end > srcSize-1 {
			end = srcSize - 1
		}

		var sum float64
		for u := begin; u <= end; u++ {
			w := filter.Kernel((float64(u) - fu) / scale)
			if w != 0 {
				sum += w
				tmp = append(tmp, indexWeight{index: u, weight: w})
			}
		}
		if sum != 0 {
			for i := range tmp {
				tmp[i].weight /= sum
			}
		}

		out[v] = tmp
		tmp = tmp[len(tmp):]
	}

	return out
}

func parallel(start, stop int, fn func(<-chan int)) {
	count := stop - start
	if count < 1 {
		return
	}

	procs := runtime.GOMAXPROCS(0)
	if procs > count {
		procs = count
	}

	c := make(chan int, count)
	for i := start; i < stop; i++ {
		c <- i
	}
	close(c)

	var wg sync.WaitGroup
	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(c)
		}()
	}
	wg.Wait()
}

func clamp(x float64) uint8 {
	v := int64(x + 0.5)
	if v > 255 {
		return 255
	}
	if v > 0 {
		return uint8(v)
	}
	return 0
}

func toNRGBA(img image.Image) *image.NRGBA {
	if img, ok := img.(*image.NRGBA); ok {
		return &image.NRGBA{
			Pix:    img.Pix,
			Stride: img.Stride,
			Rect:   img.Rect.Sub(img.Rect.Min),
		}
	}
	return clone(img)
}
