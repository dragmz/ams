package ams

import (
	"bytes"
	"image"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/pkg/errors"
	"golang.design/x/clipboard"
)

func ReadWcFromClipboard() (string, error) {
	err := clipboard.Init()
	if err != nil {
		return "", errors.Wrap(err, "failed to initialize clipboard")
	}

	bs := clipboard.Read(clipboard.FmtText)

	return string(bs), nil
}

func ReadQrFromClipboard() (string, error) {
	err := clipboard.Init()
	if err != nil {
		return "", errors.Wrap(err, "failed to initialize clipboard")
	}

	bs := clipboard.Read(clipboard.FmtImage)
	r := bytes.NewReader(bs)

	img, _, err := image.Decode(r)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode image")
	}

	// prepare BinaryBitmap
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)

	// decode image
	qrr := qrcode.NewQRCodeReader()
	result, err := qrr.Decode(bmp, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode QR code")
	}

	return result.GetText(), nil
}
