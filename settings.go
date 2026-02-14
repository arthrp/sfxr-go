package main

import (
	"encoding/binary"
	"os"
)

func LoadSettings(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	var version int32
	binary.Read(file, binary.LittleEndian, &version)
	if version != 100 && version != 101 && version != 102 {
		return false
	}

	var wt int32
	binary.Read(file, binary.LittleEndian, &wt)
	wave_type = int(wt)

	sound_vol = 0.5
	if version == 102 {
		binary.Read(file, binary.LittleEndian, &sound_vol)
	}

	binary.Read(file, binary.LittleEndian, &p_base_freq)
	binary.Read(file, binary.LittleEndian, &p_freq_limit)
	binary.Read(file, binary.LittleEndian, &p_freq_ramp)
	if version >= 101 {
		binary.Read(file, binary.LittleEndian, &p_freq_dramp)
	}
	binary.Read(file, binary.LittleEndian, &p_duty)
	binary.Read(file, binary.LittleEndian, &p_duty_ramp)

	binary.Read(file, binary.LittleEndian, &p_vib_strength)
	binary.Read(file, binary.LittleEndian, &p_vib_speed)
	binary.Read(file, binary.LittleEndian, &p_vib_delay)

	binary.Read(file, binary.LittleEndian, &p_env_attack)
	binary.Read(file, binary.LittleEndian, &p_env_sustain)
	binary.Read(file, binary.LittleEndian, &p_env_decay)
	binary.Read(file, binary.LittleEndian, &p_env_punch)

	binary.Read(file, binary.LittleEndian, &filter_on)
	binary.Read(file, binary.LittleEndian, &p_lpf_resonance)
	binary.Read(file, binary.LittleEndian, &p_lpf_freq)
	binary.Read(file, binary.LittleEndian, &p_lpf_ramp)
	binary.Read(file, binary.LittleEndian, &p_hpf_freq)
	binary.Read(file, binary.LittleEndian, &p_hpf_ramp)

	binary.Read(file, binary.LittleEndian, &p_pha_offset)
	binary.Read(file, binary.LittleEndian, &p_pha_ramp)

	binary.Read(file, binary.LittleEndian, &p_repeat_speed)

	if version >= 101 {
		binary.Read(file, binary.LittleEndian, &p_arp_speed)
		binary.Read(file, binary.LittleEndian, &p_arp_mod)
	}

	return true
}

func SaveSettings(filename string) bool {
	file, err := os.Create(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	version := int32(102)
	binary.Write(file, binary.LittleEndian, version)

	binary.Write(file, binary.LittleEndian, int32(wave_type))

	binary.Write(file, binary.LittleEndian, sound_vol)

	binary.Write(file, binary.LittleEndian, p_base_freq)
	binary.Write(file, binary.LittleEndian, p_freq_limit)
	binary.Write(file, binary.LittleEndian, p_freq_ramp)
	binary.Write(file, binary.LittleEndian, p_freq_dramp)
	binary.Write(file, binary.LittleEndian, p_duty)
	binary.Write(file, binary.LittleEndian, p_duty_ramp)

	binary.Write(file, binary.LittleEndian, p_vib_strength)
	binary.Write(file, binary.LittleEndian, p_vib_speed)
	binary.Write(file, binary.LittleEndian, p_vib_delay)

	binary.Write(file, binary.LittleEndian, p_env_attack)
	binary.Write(file, binary.LittleEndian, p_env_sustain)
	binary.Write(file, binary.LittleEndian, p_env_decay)
	binary.Write(file, binary.LittleEndian, p_env_punch)

	binary.Write(file, binary.LittleEndian, filter_on)
	binary.Write(file, binary.LittleEndian, p_lpf_resonance)
	binary.Write(file, binary.LittleEndian, p_lpf_freq)
	binary.Write(file, binary.LittleEndian, p_lpf_ramp)
	binary.Write(file, binary.LittleEndian, p_hpf_freq)
	binary.Write(file, binary.LittleEndian, p_hpf_ramp)

	binary.Write(file, binary.LittleEndian, p_pha_offset)
	binary.Write(file, binary.LittleEndian, p_pha_ramp)

	binary.Write(file, binary.LittleEndian, p_repeat_speed)

	binary.Write(file, binary.LittleEndian, p_arp_speed)
	binary.Write(file, binary.LittleEndian, p_arp_mod)

	return true
}
