package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/ncruces/zenity"
)

var (
	pixels []uint32
	pitch  int // Screen width in pixels

	mouse_x, mouse_y                  int
	mouse_px, mouse_py                int
	mouse_left, mouse_right           bool
	mouse_leftclick, mouse_rightclick bool

	vselected  *float32
	vcurbutton int = -1

	font Spriteset
	ld48 Spriteset

	firstframe      bool = true
	refresh_counter int  = 0
	drawcount       int  = 0
)

func ClearScreen(color uint32) {
	for i := range pixels {
		pixels[i] = color
	}
}

func DrawBar(sx, sy, w, h int, color uint32) {
	for y := sy; y < sy+h; y++ {
		if y < 0 || y >= 480 {
			continue
		}
		offset := y*pitch + sx
		for x := 0; x < w; x++ {
			if sx+x < 0 || sx+x >= 640 {
				continue
			}
			pixels[offset+x] = color
		}
	}
}

func DrawBox(sx, sy, w, h int, color uint32) {
	DrawBar(sx, sy, w, 1, color)
	DrawBar(sx, sy, 1, h, color)
	DrawBar(sx+w, sy, 1, h, color)
	DrawBar(sx, sy+h, w+1, 1, color)
}

func DrawSprite(sprites *Spriteset, sx, sy, i int, color uint32) {
	for y := 0; y < sprites.Height; y++ {
		if sy+y < 0 || sy+y >= 480 {
			continue
		}
		offset := (sy+y)*pitch + sx
		spoffset := y*sprites.Pitch + i*sprites.Width

		if color&0xFF000000 != 0 {
			for x := 0; x < sprites.Width; x++ {
				if sx+x < 0 || sx+x >= 640 {
					spoffset++
					continue
				}
				p := sprites.Data[spoffset]
				spoffset++
				if p != 0x300030 {
					pixels[offset+x] = p
				}
			}
		} else {
			for x := 0; x < sprites.Width; x++ {
				if sx+x < 0 || sx+x >= 640 {
					spoffset++
					continue
				}
				p := sprites.Data[spoffset]
				spoffset++
				if p != 0x300030 {
					pixels[offset+x] = color
				}
			}
		}
	}
}

func DrawText(sx, sy int, color uint32, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	for i, char := range text {
		DrawSprite(&font, sx+i*8, sy, int(char)-' ', color)
	}
}

func MouseInBox(x, y, w, h int) bool {
	if mouse_x >= x && mouse_x < x+w && mouse_y >= y && mouse_y < y+h {
		return true
	}
	return false
}

func Slider(x, y int, value *float32, bipolar bool, text string) {
	if MouseInBox(x, y, 100, 10) {
		if mouse_leftclick {
			vselected = value
		}
		if mouse_rightclick {
			*value = 0.0
		}
	}
	mv := float32(mouse_x - mouse_px)
	if vselected != value {
		mv = 0.0
	}
	if bipolar {
		*value += mv * 0.005
		if *value < -1.0 {
			*value = -1.0
		}
		if *value > 1.0 {
			*value = 1.0
		}
	} else {
		*value += mv * 0.0025
		if *value < 0.0 {
			*value = 0.0
		}
		if *value > 1.0 {
			*value = 1.0
		}
	}
	DrawBar(x-1, y, 102, 10, 0x000000)
	ival := int(*value * 99)
	if bipolar {
		ival = int(*value*49.5 + 49.5)
	}
	DrawBar(x, y+1, ival, 8, 0xF0C090)
	DrawBar(x+ival, y+1, 100-ival, 8, 0x807060)
	DrawBar(x+ival, y+1, 1, 8, 0xFFFFFF)
	if bipolar {
		DrawBar(x+50, y-1, 1, 3, 0x000000)
		DrawBar(x+50, y+8, 1, 3, 0x000000)
	}
	tcol := uint32(0x000000)
	if wave_type != 0 && (value == &p_duty || value == &p_duty_ramp) {
		tcol = 0x808080
	}
	DrawText(x-4-len(text)*8, y+1, tcol, text)
}

func Button(x, y int, highlight bool, text string, id int) bool {
	color1 := uint32(0x000000)
	color2 := uint32(0xA09088)
	color3 := uint32(0x000000)
	hover := MouseInBox(x, y, 100, 17)
	if hover && mouse_leftclick {
		vcurbutton = id
	}
	current := (vcurbutton == id)
	if highlight {
		color1 = 0x000000
		color2 = 0x988070
		color3 = 0xFFF0E0
	}
	if current && hover {
		color1 = 0xA09088
		color2 = 0xFFF0E0
		color3 = 0xA09088
	}
	DrawBar(x-1, y-1, 102, 19, color1)
	DrawBar(x, y, 100, 17, color2)
	DrawText(x+5, y+5, color3, text)
	if current && hover && !mouse_left {
		return true
	}
	return false
}

func DrawScreen() {
	redraw := true
	if !firstframe && mouse_x-mouse_px == 0 && mouse_y-mouse_py == 0 && !mouse_left && !mouse_right {
		redraw = false
	}
	if !mouse_left {
		if vselected != nil || vcurbutton > -1 {
			redraw = true
			refresh_counter = 2
		}
		vselected = nil
	}
	if refresh_counter > 0 {
		refresh_counter--
		redraw = true
	}

	if playing_sample {
		redraw = true
	}

	drawcount++
	if drawcount > 20 {
		redraw = true
		drawcount = 0
	}

	if !redraw {
		return
	}

	firstframe = false

	ClearScreen(0xC0B090)

	DrawText(10, 10, 0x504030, "GENERATOR")
	for i := 0; i < 7; i++ {
		if Button(5, 35+i*30, false, categories[i].Name, 300+i) {
			switch i {
			case 0: // pickup/coin
				ResetParams()
				p_base_freq = 0.4 + frnd(0.5)
				p_env_attack = 0.0
				p_env_sustain = frnd(0.1)
				p_env_decay = 0.1 + frnd(0.4)
				p_env_punch = 0.3 + frnd(0.3)
				if rnd(1) != 0 {
					p_arp_speed = 0.5 + frnd(0.2)
					p_arp_mod = 0.2 + frnd(0.4)
				}
			case 1: // laser/shoot
				ResetParams()
				wave_type = rnd(2)
				if wave_type == 2 && rnd(1) != 0 {
					wave_type = rnd(1)
				}
				p_base_freq = 0.5 + frnd(0.5)
				p_freq_limit = p_base_freq - 0.2 - frnd(0.6)
				if p_freq_limit < 0.2 {
					p_freq_limit = 0.2
				}
				p_freq_ramp = -0.15 - frnd(0.2)
				if rnd(2) == 0 {
					p_base_freq = 0.3 + frnd(0.6)
					p_freq_limit = frnd(0.1)
					p_freq_ramp = -0.35 - frnd(0.3)
				}
				if rnd(1) != 0 {
					p_duty = frnd(0.5)
					p_duty_ramp = frnd(0.2)
				} else {
					p_duty = 0.4 + frnd(0.5)
					p_duty_ramp = -frnd(0.7)
				}
				p_env_attack = 0.0
				p_env_sustain = 0.1 + frnd(0.2)
				p_env_decay = frnd(0.4)
				if rnd(1) != 0 {
					p_env_punch = frnd(0.3)
				}
				if rnd(2) == 0 {
					p_pha_offset = frnd(0.2)
					p_pha_ramp = -frnd(0.2)
				}
				if rnd(1) != 0 {
					p_hpf_freq = frnd(0.3)
				}
			case 2: // explosion
				ResetParams()
				wave_type = 3
				if rnd(1) != 0 {
					p_base_freq = 0.1 + frnd(0.4)
					p_freq_ramp = -0.1 + frnd(0.4)
				} else {
					p_base_freq = 0.2 + frnd(0.7)
					p_freq_ramp = -0.2 - frnd(0.2)
				}
				p_base_freq *= p_base_freq
				if rnd(4) == 0 {
					p_freq_ramp = 0.0
				}
				if rnd(2) == 0 {
					p_repeat_speed = 0.3 + frnd(0.5)
				}
				p_env_attack = 0.0
				p_env_sustain = 0.1 + frnd(0.3)
				p_env_decay = frnd(0.5)
				if rnd(1) == 0 {
					p_pha_offset = -0.3 + frnd(0.9)
					p_pha_ramp = -frnd(0.3)
				}
				p_env_punch = 0.2 + frnd(0.6)
				if rnd(1) != 0 {
					p_vib_strength = frnd(0.7)
					p_vib_speed = frnd(0.6)
				}
				if rnd(2) == 0 {
					p_arp_speed = 0.6 + frnd(0.3)
					p_arp_mod = 0.8 - frnd(1.6)
				}
			case 3: // powerup
				ResetParams()
				if rnd(1) != 0 {
					wave_type = 1
				} else {
					p_duty = frnd(0.6)
				}
				if rnd(1) != 0 {
					p_base_freq = 0.2 + frnd(0.3)
					p_freq_ramp = 0.1 + frnd(0.4)
					p_repeat_speed = 0.4 + frnd(0.4)
				} else {
					p_base_freq = 0.2 + frnd(0.3)
					p_freq_ramp = 0.05 + frnd(0.2)
					if rnd(1) != 0 {
						p_vib_strength = frnd(0.7)
						p_vib_speed = frnd(0.6)
					}
				}
				p_env_attack = 0.0
				p_env_sustain = frnd(0.4)
				p_env_decay = 0.1 + frnd(0.4)
			case 4: // hit/hurt
				ResetParams()
				wave_type = rnd(2)
				if wave_type == 2 {
					wave_type = 3
				}
				if wave_type == 0 {
					p_duty = frnd(0.6)
				}
				p_base_freq = 0.2 + frnd(0.6)
				p_freq_ramp = -0.3 - frnd(0.4)
				p_env_attack = 0.0
				p_env_sustain = frnd(0.1)
				p_env_decay = 0.1 + frnd(0.2)
				if rnd(1) != 0 {
					p_hpf_freq = frnd(0.3)
				}
			case 5: // jump
				ResetParams()
				wave_type = 0
				p_duty = frnd(0.6)
				p_base_freq = 0.3 + frnd(0.3)
				p_freq_ramp = 0.1 + frnd(0.2)
				p_env_attack = 0.0
				p_env_sustain = 0.1 + frnd(0.3)
				p_env_decay = 0.1 + frnd(0.2)
				if rnd(1) != 0 {
					p_hpf_freq = frnd(0.3)
				}
				if rnd(1) != 0 {
					p_lpf_freq = 1.0 - frnd(0.6)
				}
			case 6: // blip/select
				ResetParams()
				wave_type = rnd(1)
				if wave_type == 0 {
					p_duty = frnd(0.6)
				}
				p_base_freq = 0.2 + frnd(0.4)
				p_env_attack = 0.0
				p_env_sustain = 0.1 + frnd(0.1)
				p_env_decay = frnd(0.2)
				p_hpf_freq = 0.1
			}
			PlaySample()
		}
	}

	DrawBar(110, 0, 2, 480, 0x000000)
	DrawText(120, 10, 0x504030, "MANUAL SETTINGS")
	DrawSprite(&ld48, 8, 440, 0, 0xB0A080)

	if Button(130, 30, wave_type == 0, "SQUAREWAVE", 10) {
		wave_type = 0
	}
	if Button(250, 30, wave_type == 1, "SAWTOOTH", 11) {
		wave_type = 1
	}
	if Button(370, 30, wave_type == 2, "SINEWAVE", 12) {
		wave_type = 2
	}
	if Button(490, 30, wave_type == 3, "NOISE", 13) {
		wave_type = 3
	}

	do_play := false

	DrawBar(5-1-1, 412-1-1, 102+2, 19+2, 0x000000)
	if Button(5, 412, false, "RANDOMIZE", 40) {
		p_base_freq = float32(math.Pow(float64(frnd(2.0)-1.0), 2.0))
		if rnd(1) != 0 {
			p_base_freq = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0)) + 0.5
		}
		p_freq_limit = 0.0
		p_freq_ramp = float32(math.Pow(float64(frnd(2.0)-1.0), 5.0))
		if p_base_freq > 0.7 && p_freq_ramp > 0.2 {
			p_freq_ramp = -p_freq_ramp
		}
		if p_base_freq < 0.2 && p_freq_ramp < -0.05 {
			p_freq_ramp = -p_freq_ramp
		}
		p_freq_dramp = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		p_duty = frnd(2.0) - 1.0
		p_duty_ramp = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		p_vib_strength = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		p_vib_speed = frnd(2.0) - 1.0
		p_vib_delay = frnd(2.0) - 1.0
		p_env_attack = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		p_env_sustain = float32(math.Pow(float64(frnd(2.0)-1.0), 2.0))
		p_env_decay = frnd(2.0) - 1.0
		p_env_punch = float32(math.Pow(float64(frnd(0.8)), 2.0))
		if p_env_attack+p_env_sustain+p_env_decay < 0.2 {
			p_env_sustain += 0.2 + frnd(0.3)
			p_env_decay += 0.2 + frnd(0.3)
		}
		p_lpf_resonance = frnd(2.0) - 1.0
		p_lpf_freq = 1.0 - float32(math.Pow(float64(frnd(1.0)), 3.0))
		p_lpf_ramp = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		if p_lpf_freq < 0.1 && p_lpf_ramp < -0.05 {
			p_lpf_ramp = -p_lpf_ramp
		}
		p_hpf_freq = float32(math.Pow(float64(frnd(1.0)), 5.0))
		p_hpf_ramp = float32(math.Pow(float64(frnd(2.0)-1.0), 5.0))
		p_pha_offset = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		p_pha_ramp = float32(math.Pow(float64(frnd(2.0)-1.0), 3.0))
		p_repeat_speed = frnd(2.0) - 1.0
		p_arp_speed = frnd(2.0) - 1.0
		p_arp_mod = frnd(2.0) - 1.0
		do_play = true
	}

	if Button(5, 382, false, "MUTATE", 30) {
		if rnd(1) != 0 {
			p_base_freq += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_freq_ramp += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_freq_dramp += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_duty += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_duty_ramp += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_vib_strength += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_vib_speed += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_vib_delay += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_env_attack += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_env_sustain += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_env_decay += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_env_punch += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_lpf_resonance += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_lpf_freq += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_lpf_ramp += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_hpf_freq += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_hpf_ramp += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_pha_offset += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_pha_ramp += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_repeat_speed += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_arp_speed += frnd(0.1) - 0.05
		}
		if rnd(1) != 0 {
			p_arp_mod += frnd(0.1) - 0.05
		}
		do_play = true
	}

	DrawText(515, 170, 0x000000, "VOLUME")
	DrawBar(490-1-1+60, 180-1+5, 70, 2, 0x000000)
	DrawBar(490-1-1+60+68, 180-1+5, 2, 205, 0x000000)
	DrawBar(490-1-1+60, 180-1, 42+2, 10+2, 0xFF0000)
	Slider(490, 180, &sound_vol, false, " ")
	if Button(490, 200, false, "PLAY SOUND", 20) {
		PlaySample()
	}

	if Button(490, 290, false, "LOAD SOUND", 14) {
		filename, err := zenity.SelectFile(
			zenity.Title("Load sound settings"),
			zenity.FileFilter{Name: "CFG files", Patterns: []string{"*.cfg"}},
		)
		if err == nil && filename != "" {
			ResetParams()
			LoadSettings(filename)
			PlaySample()
		}
	}
	if Button(490, 320, false, "SAVE SOUND", 15) {
		filename, err := zenity.SelectFileSave(
			zenity.Title("Save sound settings"),
			zenity.FileFilter{Name: "CFG files", Patterns: []string{"*.cfg"}},
		)
		if err == nil && filename != "" {
			SaveSettings(filename)
		}
	}

	DrawBar(490-1-1+60, 380-1+9, 70, 2, 0x000000)
	DrawBar(490-1-2, 380-1-2, 102+4, 19+4, 0x000000)
	if Button(490, 380, false, "EXPORT .WAV", 16) {
		filename, err := zenity.SelectFileSave(
			zenity.Title("Export WAV"),
			zenity.FileFilter{Name: "WAV files", Patterns: []string{"*.wav"}},
		)
		if err == nil && filename != "" {
			if !strings.HasSuffix(strings.ToLower(filename), ".wav") {
				filename += ".wav"
			}
			if ExportWAV(filename) {
				fmt.Printf("Exported to %s\n", filename)
			} else {
				fmt.Printf("Export failed\n")
			}
		}
	}

	str := fmt.Sprintf("%d HZ", wav_freq)
	if Button(490, 410, false, str, 18) {
		if wav_freq == 44100 {
			wav_freq = 22050
		} else {
			wav_freq = 44100
		}
	}
	str = fmt.Sprintf("%d-BIT", wav_bits)
	if Button(490, 440, false, str, 19) {
		if wav_bits == 16 {
			wav_bits = 8
		} else {
			wav_bits = 16
		}
	}

	ypos := 4
	xpos := 350

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_env_attack, false, "ATTACK TIME")
	ypos++
	Slider(xpos, ypos*18, &p_env_sustain, false, "SUSTAIN TIME")
	ypos++
	Slider(xpos, ypos*18, &p_env_punch, false, "SUSTAIN PUNCH")
	ypos++
	Slider(xpos, ypos*18, &p_env_decay, false, "DECAY TIME")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_base_freq, false, "START FREQUENCY")
	ypos++
	Slider(xpos, ypos*18, &p_freq_limit, false, "MIN FREQUENCY")
	ypos++
	Slider(xpos, ypos*18, &p_freq_ramp, true, "SLIDE")
	ypos++
	Slider(xpos, ypos*18, &p_freq_dramp, true, "DELTA SLIDE")
	ypos++

	Slider(xpos, ypos*18, &p_vib_strength, false, "VIBRATO DEPTH")
	ypos++
	Slider(xpos, ypos*18, &p_vib_speed, false, "VIBRATO SPEED")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_arp_mod, true, "CHANGE AMOUNT")
	ypos++
	Slider(xpos, ypos*18, &p_arp_speed, false, "CHANGE SPEED")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_duty, false, "SQUARE DUTY")
	ypos++
	Slider(xpos, ypos*18, &p_duty_ramp, true, "DUTY SWEEP")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_repeat_speed, false, "REPEAT SPEED")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_pha_offset, true, "PHASER OFFSET")
	ypos++
	Slider(xpos, ypos*18, &p_pha_ramp, true, "PHASER SWEEP")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	Slider(xpos, ypos*18, &p_lpf_freq, false, "LP FILTER CUTOFF")
	ypos++
	Slider(xpos, ypos*18, &p_lpf_ramp, true, "LP FILTER CUTOFF SWEEP")
	ypos++
	Slider(xpos, ypos*18, &p_lpf_resonance, false, "LP FILTER RESONANCE")
	ypos++
	Slider(xpos, ypos*18, &p_hpf_freq, false, "HP FILTER CUTOFF")
	ypos++
	Slider(xpos, ypos*18, &p_hpf_ramp, true, "HP FILTER CUTOFF SWEEP")
	ypos++

	DrawBar(xpos-190, ypos*18-5, 300, 2, 0x000000)

	DrawBar(xpos-190, 4*18-5, 1, (ypos-4)*18, 0x000000)
	DrawBar(xpos-190+299, 4*18-5, 1, (ypos-4)*18, 0x000000)

	if do_play {
		PlaySample()
	}

	if !mouse_left {
		vcurbutton = -1
	}
}
