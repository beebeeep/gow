package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/gm"
	"gitlab.com/gomidi/midi/v2/smf"

	_ "gitlab.com/gomidi/midi/v2/drivers/portmididrv" // autoregisters driver
)

const (
	SR       = 44100
	BPS      = 16
	DURATION = 2
)

func collatz(n uint64) ([]uint64, uint64) {
	var (
		max uint64 = 0
		r          = make([]uint64, 0, 4)
	)
	for n != 1 {
		if n > max {
			max = n
		}
		if n%2 == 0 {
			r = append(r, n)
			n = n / 2
			continue
		}
		r = append(r, n)
		n = 3*n + 1
	}
	return r, max
}

func freq(f float64) float64 {
	return 2.0 * math.Pi * f / float64(SR)
}

func main() {
	n, err := strconv.ParseUint(os.Args[1], 10, 0)
	if err != nil {
		log.Fatal(err)
	}
	numbers, max := collatz(n)
	// wav(numbers, max)
	genmidi(numbers, max)
}

func genmidi(numbers []uint64, max uint64) {

	defer midi.CloseDriver()

	fmt.Printf("outports:\n" + midi.GetOutPorts().String() + "\n")

	out, err := midi.FindOutPort("qsynth")
	if err != nil {
		fmt.Printf("can't find qsynth")
		return
	}

	// create a SMF
	rd := bytes.NewReader(mkSMF())

	// read and play it
	smf.ReadTracksFrom(rd).Do(func(ev smf.TrackEvent) {
		fmt.Printf("track %v @%vms %s\n", ev.TrackNo, ev.AbsMicroSeconds/1000, ev.Message)
	}).Play(out)
}

// makes a SMF and returns the bytes
func mkSMF() []byte {
	var (
		bf    bytes.Buffer
		clock = smf.MetricTicks(96) // resolution: 96 ticks per quarternote 960 is also common
		tr    smf.Track
	)

	// first track must have tempo and meter informations
	tr.Add(0, smf.MetaMeter(3, 4))
	tr.Add(0, smf.MetaTempo(140))
	tr.Add(0, smf.MetaInstrument("Brass"))
	tr.Add(0, midi.ProgramChange(0, gm.Instr_BrassSection.Value()))
	tr.Add(0, midi.NoteOn(0, midi.Ab(3), 120))
	tr.Add(clock.Ticks8th(), midi.NoteOn(0, midi.C(4), 120))
	// duration: a quarter note (96 ticks in our case)
	tr.Add(clock.Ticks4th()*2, midi.NoteOff(0, midi.Ab(3)))
	tr.Add(0, midi.NoteOff(0, midi.C(4)))
	tr.Close(0)

	// create the SMF and add the tracks
	s := smf.New()
	s.TimeFormat = clock
	s.Add(tr)
	s.WriteTo(&bf)
	return bf.Bytes()
}

func wav(numbers []uint64, max uint64) {

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
	var (
		dataSizePos, _        = f.Seek(0, io.SeekCurrent)
		dataSize       uint32 = 0
	)
	binary.Write(f, binary.LittleEndian, dataSize) // data size to be written later

	var (
		maxh   = 1500
		minh   = 300
		fscale = float64(maxh-minh) / float64(max)
		i      = math.Pow(2.0, 7.0/12.0)
	)
	fmt.Println(i)
	for _, n := range numbers {
		// hz := 300 + float64(n)*fscale
		hz := 300 + math.Log2(float64(n)*fscale)*float64(maxh-minh)/math.Log2(float64(maxh))
		if hz < 300 {
			continue
		}
		fmt.Printf("%.2f\n", hz)
		for t := float64(0); t < 0.1*SR; t += 1.0 {
			// v := (math.Sin(t*freq(hz)) + math.Sin(i*t*freq(hz))) / 2.0
			v := math.Sin(t * freq(hz))
			sv := int16(32768 * v)
			binary.Write(f, binary.LittleEndian, sv)
			dataSize += 2
		}
	}

	// k := 2.0 * math.Pi * 320.0 / float64(SR)
	// for t := float64(0); t < DURATION*SR; t += 1.0 {
	// 	v := (math.Sin(t*k) + math.Sin(2.0*t*k)) / 2.0
	// 	sv := int16(32768 * v)
	// 	binary.Write(f, binary.LittleEndian, sv)
	// }

	size, _ := f.Seek(0, io.SeekCurrent)
	f.Seek(4, io.SeekStart)
	binary.Write(f, binary.LittleEndian, uint32(size-8))
	f.Seek(dataSizePos, io.SeekStart)
	binary.Write(f, binary.LittleEndian, dataSize)

	f.Close()
}
