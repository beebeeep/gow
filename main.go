package main

import (
	"encoding/binary"
	"io"
	"log"
	"math"
	"os"
)

const (
	SR       = 44100
	BPS      = 16
	DURATION = 2
)

func main() {
	f, err := os.OpenFile("out.wav", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}

	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(0)) // file size to be written later
	f.Write([]byte("WAVEfmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))       // length of chunk
	binary.Write(f, binary.LittleEndian, uint16(1))        // Type (PCM)
	binary.Write(f, binary.LittleEndian, uint16(1))        // number of channels
	binary.Write(f, binary.LittleEndian, uint32(SR))       // sample rate
	binary.Write(f, binary.LittleEndian, uint32(SR*BPS/8)) // byte rate
	binary.Write(f, binary.LittleEndian, uint16(BPS/8))    // block align
	binary.Write(f, binary.LittleEndian, uint16(BPS))      // bits per sample
	f.Write([]byte("data"))

	k := 2.0 * math.Pi * 320.0 / float64(SR)
	for t := float64(0); t < DURATION*SR; t += 1.0 {
		v := (math.Sin(t*k) + math.Sin(2.0*t*k)) / 2.0
		sv := int16(32768 * v)
		binary.Write(f, binary.LittleEndian, sv)
	}

	size, _ := f.Seek(0, io.SeekCurrent)
	f.Seek(4, io.SeekStart)
	binary.Write(f, binary.LittleEndian, uint32(size-8))

	f.Close()
}
