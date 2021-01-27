package sample

import (
	"math/rand"
	"time"

	"gitlab.techschool.pcbook/pb"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomKeyboardLayout() pb.Keyboard_Layout {
	switch rand.Intn(3) {
	case 1:
		return pb.Keyboard_QWERTY
	case 2:
		return pb.Keyboard_QWERTZ
	case 3:
		return pb.Keyboard_AZERTY
	}

	return pb.Keyboard_UNKNOWN
}

func randomBool() bool {
	return rand.Intn(2) == 1
}

func randomCPUBrand() string {
	return randomStrimFromSet("Intel", "AMD")
}

func randomStrimFromSet(a ...string) string {
	n := len(a)
	if n == 0 {
		return ""
	}

	return a[rand.Intn(n)]
}

func randomCPUName(brand string) string {
	if brand == "Intel" {
		return randomStrimFromSet(
			"Xeon E-2203M",
			"Core i9-9800HK",
			"Core u709750H",
			"Core i5-9400F",
		)
	}
	return randomStrimFromSet(
		"Ryzen 7 PRO 2700U",
		"Ryzen 5 PRO 3500U",
		"Ryzen 3 PRO 3200GE",
	)
}

func randomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func randomFloat64(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randomFloat32(min, max float32) float32 {
	return min + rand.Float32()*(max-min)
}

func randomGPUBrnad() string {
	return randomStrimFromSet("NVIDIA", "AMD")
}

func randomGPUName(brand string) string {
	if brand == "NVIDIA" {
		return randomStrimFromSet(
			"RTX 2060",
			"RTX 2070",
			"GTX 1660-Ti",
			"GTX 1060",
		)
	}

	return randomStrimFromSet(
		"RX 590",
		"RX 580",
		"RX 5700-XT",
		"RX Vega-56",
	)
}

func randomScreenPanel() pb.Screen_Panel {
	if rand.Intn(2) == 1 {
		return pb.Screen_IPS
	}

	return pb.Screen_OLED
}

func randomScreenResolution() *pb.Screen_Resolution {
	height := randomInt(1080, 4320)
	width := height * 16 / 9
	return &pb.Screen_Resolution{
		Height: uint32(height),
		Width:  uint32(width),
	}
}

func randomLaptopbrand() string {
	return randomStrimFromSet("Apple", "Dell", "Lenovo")
}

func randomLaptopName(brand string) string {
	switch brand {
	case "Apple":
		return randomStrimFromSet("Macbook Air", "Macbook pro")
	case "Dell":
		return randomStrimFromSet("Latitude", "Vostro", "Xps", "Alienware")
	case "Lenovo":
		return randomStrimFromSet("Thinkpad x1", "Thinkpad P1", "Thinkpad PS3")
	}
	return ""
}
