package onvif

import (
	"strings"
	"testing"
)

func TestProfileAdvertisesConfiguredEncoder(t *testing.T) {
	enc := Encoder{Encoding: "H265", Width: 3840, Height: 2160, FPS: 15, Bitrate: 4096}
	b := string(GetProfilesResponse([]string{"foyer"}, func(string) Encoder { return enc }))
	for _, want := range []string{
		`token="foyer"`, `<tt:Encoding>H265</tt:Encoding>`,
		`<tt:Width>3840</tt:Width><tt:Height>2160</tt:Height>`,
		`<tt:FrameRateLimit>15</tt:FrameRateLimit>`, `<tt:BitrateLimit>4096</tt:BitrateLimit>`,
	} {
		if !strings.Contains(b, want) {
			t.Fatalf("profile lacks %s:\n%s", want, b)
		}
	}
	if strings.Contains(b, "<tt:H264>") {
		t.Fatalf("H264 block must not appear for H265")
	}
	// defaults stay as before
	d := string(GetProfilesResponse([]string{"x"}, func(string) Encoder { return Encoder{} }))
	if !strings.Contains(d, `<tt:Encoding>H264</tt:Encoding>`) || !strings.Contains(d, `<tt:Width>1920</tt:Width>`) || !strings.Contains(d, "<tt:H264>") {
		t.Fatalf("defaults changed:\n%s", d)
	}
}

func TestProfilesPerStreamOverride(t *testing.T) {
	hq := Encoder{Encoding: "H265", Width: 3840, Height: 2160, FPS: 15, Bitrate: 4096}
	lq := Encoder{Encoding: "H265", Width: 640, Height: 360, FPS: 5, Bitrate: 300}
	f := func(name string) Encoder {
		if name == "foyer_lq" {
			return lq
		}
		return hq
	}
	b := string(GetProfilesResponse([]string{"foyer", "foyer_lq"}, f))
	i, j := strings.Index(b, `token="foyer"`), strings.Index(b, `token="foyer_lq"`)
	if i < 0 || j < 0 || i > j {
		t.Fatalf("profiles missing or out of order")
	}
	if !strings.Contains(b[i:j], "<tt:Width>3840</tt:Width>") || !strings.Contains(b[j:], "<tt:Width>640</tt:Width>") {
		t.Fatalf("per-stream resolution not applied:\n%s", b)
	}
}
