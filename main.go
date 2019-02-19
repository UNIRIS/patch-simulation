package main

import (
	"encoding/csv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/golang/geo/s2"
)

func main() {

	//Prepare the output image
	img := image.NewRGBA(image.Rect(-180, -90, 180, 90))
	white := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.ZP, draw.Src)

	//Load CSV positions
	filePositions, err := os.OpenFile("positions.csv", os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	rPos := csv.NewReader(filePositions)
	positions, err := rPos.ReadAll()
	if err != nil {
		panic(err)
	}

	mapColors := []color.Color{
		color.RGBA{46, 49, 49, 255},
		color.RGBA{103, 128, 159, 255},
	}

	indexColor := 0

	//Build the geolocation activatedPatches for 10° between and draw a grid map
	geoPatches := make([]patch, 0)
	for x := -180.0; x < 180.0; x += 10 {
		for y := -90.0; y < 90.0; y += 10 {
			geoPatches = append(geoPatches, patch{
				left:   x,
				right:  x + 10,
				bottom: y,
				top:    y + 10,
			})

			rect := image.Rect(int(x), int(y), int(x+10), int(y+10))
			draw.Draw(img, rect, &image.Uniform{mapColors[indexColor]}, image.ZP, draw.Src)
			indexColor = 1 - indexColor
		}
		indexColor = 1 - indexColor
	}

	//Build activates patches
	activatedPatches := make([]patch, 0)

	//Loop over the peer positions
	for _, pos := range positions {
		lat, _ := strconv.ParseFloat(strings.TrimSpace(pos[0]), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(pos[1]), 64)

		//If the patch exist and if the size of patch is not too small we split it into subactivatedPatches
		//otherwise we add the found patch into the available list
		isActivated, p, index := findPatch(lat, lon, activatedPatches)
		if isActivated {

			//Size of a patch
			patchSize := p.right - p.left

			//When we do not reached 1% degree for subpatch (about 100 km), we continue to spit into subactivatedPatches
			if patchSize > 1 {

				activatedPatches = append(activatedPatches[:index], activatedPatches[index+1:]...)

				//We reduce the internal between patch by 2
				patchSize = patchSize / 2

				for x := p.left; x < p.right; x += patchSize {
					for y := p.bottom; y < p.top; y += patchSize {
						activatedPatches = append(activatedPatches, patch{
							left:   x,
							right:  x + patchSize,
							bottom: y,
							top:    y + patchSize,
						})
					}
				}
			}

		} else {
			_, patch, _ := findPatch(lat, lon, geoPatches)
			activatedPatches = append(activatedPatches, patch)
		}
	}

	//Display patches
	printAvailablePatch(activatedPatches, img)

	//Display positions
	for _, pos := range positions {
		lat, _ := strconv.ParseFloat(strings.TrimSpace(pos[0]), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(pos[1]), 64)

		proj := s2.NewPlateCarreeProjection(180)
		latlon := s2.LatLngFromDegrees(lat, lon)
		point := proj.FromLatLng(latlon)

		rect := image.Rect(int(point.X), -int(point.Y), int(point.X+1), -int(point.Y+1))
		draw.Draw(img, rect, &image.Uniform{color.RGBA{240, 255, 0, 255}}, image.ZP, draw.Src)
	}

	//Generate the output image
	f, _ := os.Create("draw.png")
	png.Encode(f, img)
}

func printAvailablePatch(activatedPatches []patch, img *image.RGBA) {

	for _, p := range activatedPatches {

		col := color.RGBA{200, 0, 0, 255}

		rect := image.Rect(int(p.left), -int(p.bottom), int(p.right), -int(p.top))
		draw.Draw(img, rect, &image.Uniform{col}, image.ZP, draw.Src)

		black := color.RGBA{0, 0, 0, 255}
		for i := int(p.left); i < int(p.right); i++ {
			img.Set(i, -int(p.bottom), black)
			img.Set(i, -int(p.top), black)
		}

		for i := int(p.bottom); i < int(p.top); i++ {
			img.Set(int(p.left), -i, black)
			img.Set(int(p.right), -i, black)
		}
	}
}

type patch struct {
	left   float64
	right  float64
	top    float64
	bottom float64
}

func findPatch(lat float64, lon float64, patches []patch) (bool, patch, int) {
	for i, patch := range patches {

		if lon >= patch.left && lon <= patch.right && lat >= patch.bottom && lat <= patch.top {
			return true, patch, i
		}
	}

	return false, patch{}, -1
}
