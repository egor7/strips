package main

import (
	"flag"
	"fmt"
	"image"
	"math"
//	"io/ioutil"
	"log"
	"os"
//	"path/filepath"
//	"strconv"

	"image/color"
	_ "image/gif"  // register the GIF format with the image package
	_ "image/jpeg" // register the JPEG format with the image package
	"image/png"
)

var act = flag.String("act", "remove", "Action to perform: add=add some lines, remove=remove them")
var lvl = flag.Float64("lvl", 0, "Dispersion level to start apply algorithm from")
var lines = flag.Int("lines", 5, "Number of lines around to get a middle dispersion from")
var lvlm = flag.Float64("lvlm", 10, "Dispersion level for nearby rows to become stripes")

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n")
	fmt.Fprintf(os.Stderr, "  strips [flags] <image>\n")
	fmt.Fprintf(os.Stderr, "rules:\n")
	fmt.Fprintf(os.Stderr, "  - <image> is a jpg or png image format file\n")
	fmt.Fprintf(os.Stderr, "flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if flag.NArg() != 1 {
		usage()
	}

	// open image
	imf, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	defer imf.Close()

	// read image size
	im, _, err := image.Decode(imf)
	if err != nil {
		log.Fatal(err)
	}
	bounds := im.Bounds()
	w, h := bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y

	fmt.Printf("process %s [%dx%d]: ", imf.Name(), w, h)
	wim := image.NewRGBA(image.Rect(0, 0, w, h))

	// process - add
	if *act=="add" {
		for r := 0; r < h; r++ {
			if r%3 == 0 {
				var rm, gm, bm float64
				for c := 0; c < w; c++ {
					rc, gc, bc, _ := im.At(c, r).RGBA()
					rm += float64(rc>>8)/float64(w)
					gm += float64(gc>>8)/float64(w)
					bm += float64(bc>>8)/float64(w)
					// rm += float64(color.NRGBAModel.Convert(im.At(c, r)).(color.NRGBA).R)/float64(w)
					// gm += float64(color.NRGBAModel.Convert(im.At(c, r)).(color.NRGBA).G)/float64(w)
					// bm += float64(color.NRGBAModel.Convert(im.At(c, r)).(color.NRGBA).B)/float64(w)
				}
				for c := 0; c < w; c++ {
					wim.Set(c, r, color.NRGBA{uint8(rm), uint8(gm), uint8(bm), 255})
				}
			} else {
				for c := 0; c < w; c++ {
					wim.Set(c, r, im.At(c, r))
				}
			}
			if r%100 == 0 {
				fmt.Printf(".")
			}
		}
		fmt.Printf(" DONE\n")
        } else {
		// detect stripes rows
		// by dispersion level on row pixels
		stripe := make([]bool, h)
		rowdisp := make([]float64, h)
		rowmidlr := make([]float64, h)
		rowmidlg := make([]float64, h)
		rowmidlb := make([]float64, h)
		for r := 0; r < h; r++ {
		        var x, x2 float64
		        for c := 0; c < w; c++ {
		                gray := float64(color.GrayModel.Convert(im.At(c, r)).(color.Gray).Y)
		                x += gray
		                x2 += gray*gray

				cr, cg, cb, _ := im.At(c, r).RGBA()
				rowmidlr[r] += float64(cr>>8)/float64(w)
				rowmidlg[r] += float64(cg>>8)/float64(w)
				rowmidlb[r] += float64(cb>>8)/float64(w)
		        }

		        if float64(w)*x2/(x*x) - 1 <= *lvl {
		                stripe[r] = true
			}
			rowdisp[r] = float64(w)*x2/(x*x) - 1
		}

		// detect stripes rows
		// by middle level on nearest rows
		for r := 0; r < h; r++ {
			var n int
			var mr, mg, mb float64
			for ri := r - *lines; ri < r + *lines; ri++ {
				if ri < 0 || ri >= h {
					// continue
				} else {
					n += 1
					mr += rowmidlr[ri]
					mg += rowmidlg[ri]
					mb += rowmidlb[ri]
				}
			}
			mr /= float64(n)
			mg /= float64(n)
			mb /= float64(n)

		        if math.Abs(mr - rowmidlr[r]) <= *lvlm ||
			   math.Abs(mg - rowmidlg[r]) <= *lvlm ||
			   math.Abs(mb - rowmidlb[r]) <= *lvlm {
		                stripe[r] = true
			}
		}

		for r := 0; r < h; r++ {
			/*
			for c := 0; c < w; c++ {
				//wim.Set(c, r, im.At(c, r))
				cr, cg, cb, _ := im.At(c, r).RGBA()
				wim.Set(c, r, color.NRGBA{
					uint8(float64(cr>>8) - rowmidlr[r] + rowmmidlr[r]),
					uint8(float64(cg>>8) - rowmidlr[r] + rowmmidlr[r]),
					uint8(float64(cb>>8) - rowmidlr[r] + rowmmidlr[r]),
					255})
			}
			*/

			if stripe[r] {
				// get prev/next color
				rprev := r
				for ; stripe[rprev]; {
					rprev--
					if rprev == -1 {
						break
					}
				}
				rnext := r
				for ; stripe[rnext]; {
					rnext++
					if rnext == h {
						break
					}
				}
				if rprev == -1 {
					rprev = rnext
				}
				if rnext == h {
					rnext = rprev
				}

				for c := 0; c < w; c++ {
					rcprev, gcprev, bcprev, _ := im.At(c, rprev).RGBA()
					rcnext, gcnext, bcnext, _ := im.At(c, rnext).RGBA()

					wim.Set(c, r, color.NRGBA{uint8(float64(rcprev>>8) + (float64(rcnext>>8)-float64(rcprev>>8))*float64(r-rprev+1)/float64(rnext-rprev+1)),
						uint8(float64(gcprev>>8) + (float64(gcnext>>8)-float64(gcprev>>8))*float64(r-rprev+1)/float64(rnext-rprev+1)),
						uint8(float64(bcprev>>8) + (float64(bcnext>>8)-float64(bcprev>>8))*float64(r-rprev+1)/float64(rnext-rprev+1)),
						255})
				}

			} else {
				for c := 0; c < w; c++ {
					wim.Set(c, r, im.At(c, r))
				}
			}

			if r%100 == 0 {
				fmt.Printf(".")
			}
		}
		fmt.Printf(" DONE\n")
	}

	// save
	out, err := os.Create("out.png")
	defer out.Close()
	png.Encode(out, wim)
	fmt.Printf("write %s\n", out.Name())
}
