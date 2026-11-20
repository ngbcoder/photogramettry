package ingest

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ngbcoder/photogramettry/pkg/models"
	"github.com/rwcarlsen/goexif/exif"
	"gonum.org/v1/gonum/mat"
)

// SupportedExtensions lists image extensions handled by ingest.
var SupportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

// IngestImage reads a single image and builds an ImageFrame.
func IngestImage(path string, id int) (models.ImageFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		return models.ImageFrame{}, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return models.ImageFrame{}, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return models.ImageFrame{}, err
	}

	frame := models.ImageFrame{
		ID:     id,
		Path:   path,
		Width:  cfg.Width,
		Height: cfg.Height,
	}

	x, err := exif.Decode(f)
	if err == nil && x != nil {
		focalMM := 0.0
		if tag, err := x.Get("FocalLength"); err == nil {
			if r, err := tag.Rat(0); err == nil {
				if v, ok := r.Float64(); ok {
					focalMM = v
				}
			}
		}
		model := ""
		if tag, err := x.Get("Model"); err == nil {
			if v, err := tag.StringVal(); err == nil {
				model = v
			}
		}
		sensorW := SensorWidth("", model)
		if focalMM > 0 && sensorW > 0 {
			frame.Intrinsics.FocalLengthX = focalMM / sensorW * float64(cfg.Width)
			frame.Intrinsics.FocalLengthY = frame.Intrinsics.FocalLengthX
		} else {
			frame.Intrinsics.FocalLengthX = float64(cfg.Width)
			frame.Intrinsics.FocalLengthY = float64(cfg.Height)
		}
	} else {
		frame.Intrinsics.FocalLengthX = float64(cfg.Width)
		frame.Intrinsics.FocalLengthY = float64(cfg.Height)
	}

	frame.Intrinsics.PrincipalX = float64(cfg.Width) / 2.0
	frame.Intrinsics.PrincipalY = float64(cfg.Height) / 2.0
	if frame.Intrinsics.KMatrix == nil {
		k := makeK(frame.Intrinsics.FocalLengthX, frame.Intrinsics.FocalLengthY,
			frame.Intrinsics.PrincipalX, frame.Intrinsics.PrincipalY)
		frame.Intrinsics.KMatrix = k
	}
	return frame, nil
}

func makeK(fx, fy, cx, cy float64) *mat.Dense {
	k := mat.NewDense(3, 3, []float64{
		fx, 0, cx,
		0, fy, cy,
		0, 0, 1,
	})
	return k
}

// IngestDir walks a directory and ingests all supported images.
func IngestDir(dir string) ([]models.ImageFrame, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if SupportedExtensions[ext] {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(paths)
	frames := make([]models.ImageFrame, 0, len(paths))
	for i, p := range paths {
		fr, err := IngestImage(p, i)
		if err != nil {
			continue
		}
		frames = append(frames, fr)
	}
	return frames, nil
}
