package main

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

type Spriteset struct {
	Data   []uint32
	Width  int
	Height int
	Pitch  int
}

func LoadTGA(filename string) (*Spriteset, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	header := make([]byte, 18)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, err
	}

	idLength := int(header[0])
	width := int(binary.LittleEndian.Uint16(header[12:14]))
	height := int(binary.LittleEndian.Uint16(header[14:16]))
	bits := int(header[16])
	// imageDescriptor := header[17]

	if bits != 32 && bits != 24 {
		return nil, errors.New("only 24-bit and 32-bit TGA supported")
	}

	// Skip ID
	if idLength > 0 {
		if _, err := file.Seek(int64(idLength), io.SeekCurrent); err != nil {
			return nil, err
		}
	}

	data := make([]uint32, width*height)
	bytesPerPixel := bits / 8
	pixelBuf := make([]byte, bytesPerPixel)

	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			if _, err := io.ReadFull(file, pixelBuf); err != nil {
				return nil, err
			}
			
			var pixel uint32
			b := uint32(pixelBuf[0])
			g := uint32(pixelBuf[1])
			r := uint32(pixelBuf[2])
			
			// C++: pixel = B | (G<<8) | (R<<16) -> 0x00RRGGBB
			pixel = b | (g << 8) | (r << 16)
			
			if bytesPerPixel == 4 {
				a := uint32(pixelBuf[3])
				pixel |= (a << 24)
			}
			
			data[y*width+x] = pixel
		}
	}

	return &Spriteset{
		Data:   data,
		Width:  width,
		Height: height,
		Pitch:  width,
	}, nil
}
