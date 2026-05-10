package tokengenerator

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/disintegration/imaging"
)

const size = 512

func New(imgPath string, framePath string, destination string) (string, error) {
	pic, err := imaging.Open(imgPath)
	if err != nil {
		log.Printf("tokengenerator_new: could not load thumbnail: %s", err)
		return "", err
	}

	frame, err := imaging.Open(framePath)
	if err != nil {
		log.Printf("tokengenerator_new: could not load frame: %s", err)
		return "", err
	}

	/*
		Note:
			Tokens are limited to 512px square (sounds like a pretty reasonable size to me).
			Photos are center-cropped so they fill the square.

			When overlaying the frame on top of position 0,0, the frame covers the full canvas
	*/
	pic = resizeToSquare(pic)
	frame = resizeToSquare(frame)

	dst := applyCircleMask(pic)

	result := imaging.Overlay(dst, frame, image.Pt(0, 0), 1.0)

	f, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := png.Encode(f, result); err != nil {
		return "", err
	}

	return destination, nil
}

/*
DrawBasicFrame draws a circular gradient ring of the given color and thickness over the source
image.

The gradient fades from transparent at the inner edge to fully opaque at the outer edge.
Thickness is clamped to at most half the image size.

Returns the composited image.
*/
func DrawBasicFrame(source string, destination string, frameColor color.RGBA, thickness float32) (*image.RGBA, error) {
	pic, err := imaging.Open(source)
	if err != nil {
		log.Printf("tokengenerator_drawbasicframe: could not load thumbnail: %s", err)
		return nil, err
	}

	if pic.Bounds().Dx() != pic.Bounds().Dy() {
		pic = resizeToSquare(pic)
	}

	bounds := pic.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	maxThickness := float32(min(w, h)) / 2
	if thickness > maxThickness {
		thickness = maxThickness
	}

	src := image.NewRGBA(bounds)
	draw.Draw(src, bounds, pic, bounds.Min, draw.Src)
	return applyGradientRing(src, frameColor, thickness), nil
}

/*
NewBasicToken generates the frame from a solid color with a gradient ring.

Follows same pipeline as New(), but instead of a picture, it's a gradient.
*/
func NewBasicToken(imgPath string, frameColor color.RGBA, thickness float32, destination string) (string, error) {
	pic, err := imaging.Open(imgPath)
	if err != nil {
		log.Printf("tokengenerator_newbasictoken: could not load thumbnail: %s", err)
		return "", err
	}

	pic = resizeToSquare(pic)
	masked := applyCircleMask(pic)

	maxThickness := float32(size) / 2
	if thickness > maxThickness {
		thickness = maxThickness
	}

	result := applyGradientRing(masked, frameColor, thickness)

	f, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := png.Encode(f, result); err != nil {
		return "", err
	}

	return destination, nil
}

func applyCircleMask(img image.Image) *image.RGBA {
	bounds := image.Rect(0, 0, size, size)
	center := image.Point{X: size / 2, Y: size / 2}
	radius := size / 2

	mask := image.NewAlpha(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := x - center.X
			dy := y - center.Y
			if dx*dx+dy*dy <= radius*radius {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}

	dst := image.NewRGBA(bounds)
	draw.DrawMask(dst, bounds, img, image.Point{}, mask, image.Point{}, draw.Over)
	return dst
}

func applyGradientRing(img *image.RGBA, frameColor color.RGBA, thickness float32) *image.RGBA {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	cx := float64(w) / 2
	cy := float64(h) / 2
	outerR := math.Min(float64(w), float64(h)) / 2
	innerR := outerR - float64(thickness)

	ring := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			t := (dist - innerR) / float64(thickness)
			ring.SetRGBA(x, y, color.RGBA{
				R: frameColor.R,
				G: frameColor.G,
				B: frameColor.B,
				A: uint8(t * float64(frameColor.A)),
			})
		}
	}

	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)
	draw.Draw(dst, bounds, ring, bounds.Min, draw.Over)
	return dst
}

func resizeToSquare(img image.Image) image.Image {
	return imaging.Fill(img, size, size, imaging.Center, imaging.Lanczos)
}
