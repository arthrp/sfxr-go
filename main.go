package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	PI = 3.14159265
)

// Global sound parameters
var (
	wave_type int

	p_base_freq  float32
	p_freq_limit float32
	p_freq_ramp  float32
	p_freq_dramp float32
	p_duty       float32
	p_duty_ramp  float32

	p_vib_strength float32
	p_vib_speed    float32
	p_vib_delay    float32

	p_env_attack  float32
	p_env_sustain float32
	p_env_decay   float32
	p_env_punch   float32

	filter_on       bool
	p_lpf_resonance float32
	p_lpf_freq      float32
	p_lpf_ramp      float32
	p_hpf_freq      float32
	p_hpf_ramp      float32

	p_pha_offset float32
	p_pha_ramp   float32

	p_repeat_speed float32

	p_arp_speed float32
	p_arp_mod   float32

	master_vol float32 = 0.05
	sound_vol  float32 = 0.5
)

// Synthesis state
var (
	playing_sample bool
	phase          int
	fperiod        float64
	fmaxperiod     float64
	fslide         float64
	fdslide        float64
	period         int
	square_duty    float32
	square_slide   float32
	env_stage      int
	env_time       int
	env_length     [3]int
	env_vol        float32
	fphase         float32
	fdphase        float32
	iphase         int
	phaser_buffer  [1024]float32
	ipp            int
	noise_buffer   [32]float32
	fltp           float32
	fltdp          float32
	fltw           float32
	fltw_d         float32
	fltdmp         float32
	fltphp         float32
	flthp          float32
	flthp_d        float32
	vib_phase      float32
	vib_speed      float32
	vib_amp        float32
	rep_time       int
	rep_limit      int
	arp_time       int
	arp_limit      int
	arp_mod        float64

	wav_bits int = 16
	wav_freq int = 44100

	file_sampleswritten int
	filesample          float32 = 0.0
	fileacc             int     = 0

	mute_stream bool
)

type Category struct {
	Name string
}

var categories = []Category{
	{"PICKUP/COIN"},
	{"LASER/SHOOT"},
	{"EXPLOSION"},
	{"POWERUP"},
	{"HIT/HURT"},
	{"JUMP"},
	{"BLIP/SELECT"},
}

func rnd(n int) int {
	return rand.Intn(n + 1)
}

func frnd(rangeVal float32) float32 {
	return float32(rand.Intn(10000)) / 10000 * rangeVal
}

func ResetParams() {
	wave_type = 0

	p_base_freq = 0.3
	p_freq_limit = 0.0
	p_freq_ramp = 0.0
	p_freq_dramp = 0.0
	p_duty = 0.0
	p_duty_ramp = 0.0

	p_vib_strength = 0.0
	p_vib_speed = 0.0
	p_vib_delay = 0.0

	p_env_attack = 0.0
	p_env_sustain = 0.3
	p_env_decay = 0.4
	p_env_punch = 0.0

	filter_on = false
	p_lpf_resonance = 0.0
	p_lpf_freq = 1.0
	p_lpf_ramp = 0.0
	p_hpf_freq = 0.0
	p_hpf_ramp = 0.0

	p_pha_offset = 0.0
	p_pha_ramp = 0.0

	p_repeat_speed = 0.0

	p_arp_speed = 0.0
	p_arp_mod = 0.0
}

func ResetSample(restart bool) {
	if !restart {
		phase = 0
	}
	fperiod = 100.0 / (float64(p_base_freq*p_base_freq) + 0.001)
	period = int(fperiod)
	fmaxperiod = 100.0 / (float64(p_freq_limit*p_freq_limit) + 0.001)
	fslide = 1.0 - math.Pow(float64(p_freq_ramp), 3.0)*0.01
	fdslide = -math.Pow(float64(p_freq_dramp), 3.0) * 0.000001
	square_duty = 0.5 - p_duty*0.5
	square_slide = -p_duty_ramp * 0.00005
	if p_arp_mod >= 0.0 {
		arp_mod = 1.0 - math.Pow(float64(p_arp_mod), 2.0)*0.9
	} else {
		arp_mod = 1.0 + math.Pow(float64(p_arp_mod), 2.0)*10.0
	}
	arp_time = 0
	arp_limit = int(math.Pow(float64(1.0-p_arp_speed), 2.0)*20000 + 32)
	if p_arp_speed == 1.0 {
		arp_limit = 0
	}
	if !restart {
		// reset filter
		fltp = 0.0
		fltdp = 0.0
		fltw = float32(math.Pow(float64(p_lpf_freq), 3.0) * 0.1)
		fltw_d = 1.0 + p_lpf_ramp*0.0001
		fltdmp = 5.0 / (1.0 + float32(math.Pow(float64(p_lpf_resonance), 2.0))*20.0) * (0.01 + fltw)
		if fltdmp > 0.8 {
			fltdmp = 0.8
		}
		fltphp = 0.0
		flthp = float32(math.Pow(float64(p_hpf_freq), 2.0) * 0.1)
		flthp_d = 1.0 + p_hpf_ramp*0.0003
		// reset vibrato
		vib_phase = 0.0
		vib_speed = float32(math.Pow(float64(p_vib_speed), 2.0) * 0.01)
		vib_amp = p_vib_strength * 0.5
		// reset envelope
		env_vol = 0.0
		env_stage = 0
		env_time = 0
		env_length[0] = int(p_env_attack * p_env_attack * 100000.0)
		env_length[1] = int(p_env_sustain * p_env_sustain * 100000.0)
		env_length[2] = int(p_env_decay * p_env_decay * 100000.0)

		fphase = float32(math.Pow(float64(p_pha_offset), 2.0) * 1020.0)
		if p_pha_offset < 0.0 {
			fphase = -fphase
		}
		fdphase = float32(math.Pow(float64(p_pha_ramp), 2.0) * 1.0)
		if p_pha_ramp < 0.0 {
			fdphase = -fdphase
		}
		iphase = int(math.Abs(float64(fphase)))
		ipp = 0
		for i := 0; i < 1024; i++ {
			phaser_buffer[i] = 0.0
		}

		for i := 0; i < 32; i++ {
			noise_buffer[i] = frnd(2.0) - 1.0
		}

		rep_time = 0
		rep_limit = int(math.Pow(float64(1.0-p_repeat_speed), 2.0)*20000 + 32)
		if p_repeat_speed == 0.0 {
			rep_limit = 0
		}
	}
}

func PlaySample() {
	ResetSample(false)
	playing_sample = true
}

func SynthSample(length int, buffer []float32, file *os.File) {
	for i := 0; i < length; i++ {
		if !playing_sample {
			break
		}

		rep_time++
		if rep_limit != 0 && rep_time >= rep_limit {
			rep_time = 0
			ResetSample(true)
		}

		// frequency envelopes/arpeggios
		arp_time++
		if arp_limit != 0 && arp_time >= arp_limit {
			arp_limit = 0
			fperiod *= arp_mod
		}
		fslide += fdslide
		fperiod *= fslide
		if fperiod > fmaxperiod {
			fperiod = fmaxperiod
			if p_freq_limit > 0.0 {
				playing_sample = false
			}
		}
		rfperiod := fperiod
		if vib_amp > 0.0 {
			vib_phase += vib_speed
			rfperiod = fperiod * (1.0 + math.Sin(float64(vib_phase))*float64(vib_amp))
		}
		period = int(rfperiod)
		if period < 8 {
			period = 8
		}
		square_duty += square_slide
		if square_duty < 0.0 {
			square_duty = 0.0
		}
		if square_duty > 0.5 {
			square_duty = 0.5
		}
		// volume envelope
		env_time++
		if env_time > env_length[env_stage] {
			env_time = 0
			env_stage++
			if env_stage == 3 {
				playing_sample = false
			}
		}
		if env_stage == 0 {
			env_vol = float32(env_time) / float32(env_length[0])
		}
		if env_stage == 1 {
			env_vol = 1.0 + float32(math.Pow(1.0-float64(env_time)/float64(env_length[1]), 1.0))*2.0*p_env_punch
		}
		if env_stage == 2 {
			env_vol = 1.0 - float32(env_time)/float32(env_length[2])
		}

		// phaser step
		fphase += fdphase
		iphase = int(math.Abs(float64(fphase)))
		if iphase > 1023 {
			iphase = 1023
		}

		if flthp_d != 0.0 {
			flthp *= flthp_d
			if flthp < 0.00001 {
				flthp = 0.00001
			}
			if flthp > 0.1 {
				flthp = 0.1
			}
		}

		ssample := float32(0.0)
		for si := 0; si < 8; si++ { // 8x supersampling
			sample := float32(0.0)
			phase++
			if phase >= period {
				// phase = 0
				phase %= period
				if wave_type == 3 {
					for i := 0; i < 32; i++ {
						noise_buffer[i] = frnd(2.0) - 1.0
					}
				}
			}
			// base waveform
			fp := float32(phase) / float32(period)
			switch wave_type {
			case 0: // square
				if fp < square_duty {
					sample = 0.5
				} else {
					sample = -0.5
				}
			case 1: // sawtooth
				sample = 1.0 - fp*2
			case 2: // sine
				sample = float32(math.Sin(float64(fp) * 2 * PI))
			case 3: // noise
				sample = noise_buffer[phase*32/period]
			}
			// lp filter
			pp := fltp
			fltw *= fltw_d
			if fltw < 0.0 {
				fltw = 0.0
			}
			if fltw > 0.1 {
				fltw = 0.1
			}
			if p_lpf_freq != 1.0 {
				fltdp += (sample - fltp) * fltw
				fltdp -= fltdp * fltdmp
			} else {
				fltp = sample
				fltdp = 0.0
			}
			fltp += fltdp
			// hp filter
			fltphp += fltp - pp
			fltphp -= fltphp * flthp
			sample = fltphp
			// phaser
			phaser_buffer[ipp&1023] = sample
			sample += phaser_buffer[(ipp-iphase+1024)&1023]
			ipp = (ipp + 1) & 1023
			// final accumulation and envelope application
			ssample += sample * env_vol
		}
		ssample = ssample / 8 * master_vol

		ssample *= 2.0 * sound_vol

		if buffer != nil {
			if ssample > 1.0 {
				ssample = 1.0
			}
			if ssample < -1.0 {
				ssample = -1.0
			}
			buffer[i] = ssample
		}
		if file != nil {
			// quantize depending on format
			// accumulate/count to accomodate variable sample rate?
			ssample *= 4.0 // arbitrary gain to get reasonable output volume...
			if ssample > 1.0 {
				ssample = 1.0
			}
			if ssample < -1.0 {
				ssample = -1.0
			}
			filesample += ssample
			fileacc++
			if wav_freq == 44100 || fileacc == 2 {
				filesample /= float32(fileacc)
				fileacc = 0
				if wav_bits == 16 {
					isample := int16(filesample * 32000)
					binary.Write(file, binary.LittleEndian, isample)
				} else {
					isample := uint8(filesample*127 + 128)
					binary.Write(file, binary.LittleEndian, isample)
				}
				filesample = 0.0
			}
			file_sampleswritten++
		}
	}
}

func ExportWAV(filename string) bool {
	foutput, err := os.Create(filename)
	if err != nil {
		return false
	}
	defer foutput.Close()

	// write wav header
	foutput.Write([]byte("RIFF"))
	binary.Write(foutput, binary.LittleEndian, uint32(0)) // remaining file size
	foutput.Write([]byte("WAVE"))

	foutput.Write([]byte("fmt "))
	binary.Write(foutput, binary.LittleEndian, uint32(16))                  // chunk size
	binary.Write(foutput, binary.LittleEndian, uint16(1))                   // compression code
	binary.Write(foutput, binary.LittleEndian, uint16(1))                   // channels
	binary.Write(foutput, binary.LittleEndian, uint32(wav_freq))            // sample rate
	binary.Write(foutput, binary.LittleEndian, uint32(wav_freq*wav_bits/8)) // bytes/sec
	binary.Write(foutput, binary.LittleEndian, uint16(wav_bits/8))          // block align
	binary.Write(foutput, binary.LittleEndian, uint16(wav_bits))            // bits per sample

	foutput.Write([]byte("data"))
	binary.Write(foutput, binary.LittleEndian, uint32(0)) // chunk size

	foutstream_datasize, _ := foutput.Seek(0, 1)

	// write sample data
	mute_stream = true
	file_sampleswritten = 0
	filesample = 0.0
	fileacc = 0
	PlaySample()

	// Safety limit: ~10 seconds at 44.1kHz to prevent infinite loop from edge cases
	const maxSamples = 44100 * 10
	for playing_sample && file_sampleswritten < maxSamples {
		SynthSample(256, nil, foutput)
	}
	playing_sample = false // ensure we don't leave playback stuck
	mute_stream = false

	// seek back to header and write size info
	foutput.Seek(4, 0)
	binary.Write(foutput, binary.LittleEndian, uint32(int(foutstream_datasize)-4+file_sampleswritten*wav_bits/8))
	foutput.Seek(foutstream_datasize-4, 0)
	binary.Write(foutput, binary.LittleEndian, uint32(file_sampleswritten*wav_bits/8))

	return true
}

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("sfxr-go", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		640, 480, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_STREAMING, 640, 480)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()

	// Audio Setup
	spec := &sdl.AudioSpec{
		Freq:     44100,
		Format:   sdl.AUDIO_S16SYS,
		Channels: 1,
		Samples:  512,
	}

	deviceID, err := sdl.OpenAudioDevice("", false, spec, nil, 0)
	if err != nil {
		panic(err)
	}
	sdl.PauseAudioDevice(deviceID, false)
	defer sdl.CloseAudioDevice(deviceID)

	// Load assets
	f, err := LoadTGA("../font.tga")
	if err != nil {
		f, err = LoadTGA("font.tga")
		if err != nil {
			fmt.Println("Error loading font.tga", err)
			return
		}
	}
	font = *f
	font.Width = font.Height

	l, err := LoadTGA("../ld48.tga")
	if err != nil {
		l, err = LoadTGA("ld48.tga")
		if err != nil {
			fmt.Println("Error loading ld48.tga", err)
			return
		}
	}
	ld48 = *l
	ld48.Width = ld48.Pitch // Fix width from C++ code logic

	// Initialize pixels buffer
	pitch = 640
	pixels = make([]uint32, 640*480)

	ResetParams()

	running := true
	for running {
		// Audio buffering
		if playing_sample {
			queued := sdl.GetQueuedAudioSize(deviceID)
			if queued < 4096 {
				n := 1024
				fbuf := make([]float32, n)
				SynthSample(n, fbuf, nil)

				byteBuffer := make([]byte, n*2)
				for i := 0; i < n; i++ {
					f := fbuf[i]
					if f < -1.0 {
						f = -1.0
					}
					if f > 1.0 {
						f = 1.0
					}
					val := int16(f * 32767)

					byteBuffer[i*2] = byte(val & 0xFF)
					byteBuffer[i*2+1] = byte((val >> 8) & 0xFF)
				}
				sdl.QueueAudio(deviceID, byteBuffer)
			}
		}

		// Input handling
		mouse_px = mouse_x
		mouse_py = mouse_y

		x, y, state := sdl.GetMouseState()
		mouse_x = int(x)
		mouse_y = int(y)
		mouse_left = (state & sdl.ButtonLMask()) != 0
		mouse_right = (state & sdl.ButtonRMask()) != 0

		// Reset click states
		mouse_leftclick = false
		mouse_rightclick = false

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseButtonEvent:
				if t.Type == sdl.MOUSEBUTTONDOWN {
					if t.Button == sdl.BUTTON_LEFT {
						mouse_leftclick = true
						mouse_left = true
					}
					if t.Button == sdl.BUTTON_RIGHT {
						mouse_rightclick = true
						mouse_right = true
					}
				}
			case *sdl.KeyboardEvent:
				if t.Type == sdl.KEYDOWN {
					if t.Keysym.Sym == sdl.K_SPACE || t.Keysym.Sym == sdl.K_RETURN {
						PlaySample()
					}
				}
			}
		}

		// Draw UI
		DrawScreen()

		// Update texture
		texture.Update(nil, unsafe.Pointer(&pixels[0]), 640*4)

		renderer.Copy(texture, nil, nil)
		renderer.Present()
		sdl.Delay(10) // Approx 100fps
	}
}
