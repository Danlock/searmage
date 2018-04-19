package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
	"github.com/otiai10/gosseract"
	"github.com/pkg/errors"
)

type OCRParser struct {
	gosserC            *gosseract.Client
	preProcessedImages []string
}

var tmpPath = os.TempDir() + "/regex-img/"

const EDGE_DETECTION_RADIUS = 5.0
const MIN_IMG_WIDTH = 1080

func NewOCRParser(imgPath string) (*OCRParser, error) {
	img, err := imgio.Open(imgPath)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read in image!")
	}

	// img = effect.EdgeDetection(img, EDGE_DETECTION_RADIUS)
	img = effect.Grayscale(img)
	img = transform.Resize(img, img.Bounds().Dx()*2, img.Bounds().Dy()*2, transform.NearestNeighbor)
	preProcImgFilePath := tmpPath + strconv.Itoa(int(time.Now().UnixNano()))
	if err := os.MkdirAll(filepath.Dir(preProcImgFilePath), os.ModePerm); err != nil {
		return nil, errors.WithStack(err)
	}

	var imgFormat imgio.Format
	if filepath.Ext(imgPath) == ".png" {
		imgFormat = imgio.PNG
		preProcImgFilePath += ".png"
	} else {
		imgFormat = imgio.JPEG
		preProcImgFilePath += ".jpg"
	}

	if err := imgio.Save(preProcImgFilePath, img, imgFormat); err != nil {
		return nil, errors.WithStack(err)
	}

	return &OCRParser{gosseract.NewClient(), []string{preProcImgFilePath}}, nil
}

func (ocrP *OCRParser) Close() {
	// os.RemoveAll(tmpPath)
	if ocrP != nil && ocrP.gosserC != nil {
		ocrP.gosserC.Close()
	}
}

func (ocrP *OCRParser) ScanImages() (string, error) {
	fmt.Println("Starting to scan images!")
	ocrCLI := gosseract.NewClient().SetPageSegMode(gosseract.PSM_SINGLE_BLOCK)
	defer ocrCLI.Close()
	ocrCLI.SetImage(ocrP.preProcessedImages[0])

	for i := gosseract.PSM_AUTO; i <= gosseract.PSM_SPARSE_TEXT_OSD; i++ {
		ocrCLI = ocrCLI.SetPageSegMode(i)
		text, err := ocrCLI.Text()
		if err != nil {
			return "", errors.WithStack(err)
		}

		fmt.Printf("\nFound text using psm mode %d:\n%s\n", i, text)
	}

	return "text", nil
}
