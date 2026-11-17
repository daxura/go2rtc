package onvif

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlexxIT/go2rtc/internal/api"
	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/AlexxIT/go2rtc/internal/rtsp"
	"github.com/AlexxIT/go2rtc/internal/streams"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/onvif"
	"github.com/rs/zerolog"
)

func Init() {
	log = app.GetLogger("onvif")

	// What the ONVIF server advertises for its streams (go2rtc does not
	// transcode; clients like UniFi Protect size their pipeline from it).
	type encoderCfg struct {
		Encoding string `yaml:"encoding"`
		Width    int    `yaml:"width"`
		Height   int    `yaml:"height"`
		FPS      int    `yaml:"fps"`
		Bitrate  int    `yaml:"bitrate"`
		// audio track served with this stream; advertised only where set (no inheritance into `profiles`)
		Audio struct {
			Encoding   string `yaml:"encoding"`
			Bitrate    int    `yaml:"bitrate"`
			SampleRate int    `yaml:"sample_rate"`
		} `yaml:"audio"`
	}
	var cfg struct {
		Mod struct {
			encoderCfg   `yaml:",inline"`
			Profiles     map[string]encoderCfg `yaml:"profiles"` // per-stream overrides (e.g. a low-quality second profile)
			Manufacturer string                `yaml:"manufacturer"`
			Model        string                `yaml:"model"`
		} `yaml:"onvif"`
	}
	app.LoadConfig(&cfg)
	manufacturer, model = cfg.Mod.Manufacturer, cfg.Mod.Model
	if model == "" {
		model = "go2rtc"
	}
	toEnc := func(c encoderCfg) onvif.Encoder {
		return onvif.Encoder{
			Encoding: strings.ToUpper(c.Encoding), Width: c.Width, Height: c.Height, FPS: c.FPS, Bitrate: c.Bitrate,
			Audio: onvif.Audio{Encoding: strings.ToUpper(c.Audio.Encoding), Bitrate: c.Audio.Bitrate, SampleRate: c.Audio.SampleRate},
		}
	}
	base := toEnc(cfg.Mod.encoderCfg)
	overrides := map[string]onvif.Encoder{}
	for name, c := range cfg.Mod.Profiles {
		enc := toEnc(c)
		if enc.Encoding == "" {
			enc.Encoding = base.Encoding
		}
		overrides[name] = enc
	}
	encoder = func(name string) onvif.Encoder {
		if enc, ok := overrides[name]; ok {
			return enc
		}
		return base
	}

	streams.HandleFunc("onvif", streamOnvif)

	// ONVIF server on all suburls
	api.HandleFunc("/onvif/", onvifDeviceService)

	// ONVIF client autodiscovery
	api.HandleFunc("api/onvif", apiOnvif)
}

var log zerolog.Logger

func sortedNames() []string {
	names := streams.GetAllNames()
	sort.Strings(names)
	return names
}

var encoder onvif.EncoderFunc
var manufacturer, model string

func streamOnvif(rawURL string) (core.Producer, error) {
	client, err := onvif.NewClient(rawURL)
	if err != nil {
		return nil, err
	}

	uri, err := client.GetURI()
	if err != nil {
		return nil, err
	}

	// Append hash-based arguments to the retrieved URI
	if i := strings.IndexByte(rawURL, '#'); i > 0 {
		uri += rawURL[i:]
	}

	log.Debug().Msgf("[onvif] new uri=%s", uri)

	if err = streams.Validate(uri); err != nil {
		return nil, err
	}

	return streams.GetProducer(uri)
}

func onvifDeviceService(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	operation := onvif.GetRequestAction(b)
	if operation == "" {
		http.Error(w, "malformed request body", http.StatusBadRequest)
		return
	}

	log.Trace().Msgf("[onvif] server request %s %s:\n%s", r.Method, r.RequestURI, b)

	switch operation {
	case onvif.ServiceGetServiceCapabilities, // important for Hass
		onvif.DeviceGetNetworkInterfaces, // important for Hass
		onvif.DeviceGetSystemDateAndTime, // important for Hass
		onvif.DeviceSetSystemDateAndTime, // return just OK
		onvif.DeviceGetDiscoveryMode,
		onvif.DeviceGetDNS,
		onvif.DeviceGetHostname,
		onvif.DeviceGetNetworkDefaultGateway,
		onvif.DeviceGetNetworkProtocols,
		onvif.DeviceGetNTP,
		onvif.DeviceGetScopes:
		b = onvif.StaticResponse(operation)

	case onvif.MediaGetAudioEncoderConfigurations,
		onvif.MediaGetAudioSources,
		onvif.MediaGetAudioSourceConfigurations:
		audio := encoder("").Audio
		switch {
		case audio.Encoding == "":
			b = onvif.StaticResponse(operation) // empty lists
		case operation == onvif.MediaGetAudioSources:
			b = onvif.GetAudioSourcesResponse()
		case operation == onvif.MediaGetAudioSourceConfigurations:
			b = onvif.GetAudioSourceConfigurationsResponse()
		default:
			b = onvif.GetAudioEncoderConfigurationsResponse(audio)
		}

	case onvif.DeviceGetCapabilities:
		// important for Hass: Media section
		b = onvif.GetCapabilitiesResponse(r.Host)

	case onvif.DeviceGetServices:
		b = onvif.GetServicesResponse(r.Host)

	case onvif.DeviceGetDeviceInformation:
		// important for Hass: SerialNumber (unique server ID)
		b = onvif.GetDeviceInformationResponse(manufacturer, model, app.Version, r.Host)

	case onvif.DeviceSystemReboot:
		b = onvif.StaticResponse(operation)

		time.AfterFunc(time.Second, func() {
			os.Exit(0)
		})

	case onvif.MediaGetVideoSources:
		b = onvif.GetVideoSourcesResponse(sortedNames(), encoder)

	case onvif.MediaGetProfiles:
		// important for Hass: codec, width, height
		// sorted: clients such as UniFi Protect take the first profile as HQ, the second as LQ
		b = onvif.GetProfilesResponse(sortedNames(), encoder)

	case onvif.MediaGetProfile:
		token := onvif.FindTagValue(b, "ProfileToken")
		b = onvif.GetProfileResponse(token, encoder)

	case onvif.MediaGetVideoEncoderConfiguration:
		b = onvif.GetVideoEncoderConfigurationResponse(encoder(""))

	case onvif.MediaGetVideoEncoderConfigurations:
		b = onvif.GetVideoEncoderConfigurationsResponse(encoder(""))

	case onvif.MediaGetVideoEncoderConfigurationOptions:
		b = onvif.GetVideoEncoderConfigurationOptionsResponse(encoder(""))

	case onvif.MediaGetVideoSourceConfigurations:
		// important for Happytime Onvif Client
		b = onvif.GetVideoSourceConfigurationsResponse(sortedNames())

	case onvif.MediaGetVideoSourceConfiguration:
		token := onvif.FindTagValue(b, "ConfigurationToken")
		b = onvif.GetVideoSourceConfigurationResponse(token)

	case onvif.MediaGetStreamUri:
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host // in case of Host without port
		}

		uri := "rtsp://" + host + ":" + rtsp.Port + "/" + onvif.FindTagValue(b, "ProfileToken")
		b = onvif.GetStreamUriResponse(uri)

	case onvif.MediaGetSnapshotUri:
		uri := "http://" + r.Host + "/api/frame.jpeg?src=" + onvif.FindTagValue(b, "ProfileToken")
		b = onvif.GetSnapshotUriResponse(uri)

	default:
		http.Error(w, "unsupported operation", http.StatusBadRequest)
		log.Warn().Msgf("[onvif] unsupported operation: %s", operation)
		log.Debug().Msgf("[onvif] unsupported request:\n%s", b)
		return
	}

	log.Trace().Msgf("[onvif] server response:\n%s", b)

	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	if _, err = w.Write(b); err != nil {
		log.Error().Err(err).Caller().Send()
	}
}

func apiOnvif(w http.ResponseWriter, r *http.Request) {
	src := r.URL.Query().Get("src")

	var items []*api.Source

	if src == "" {
		devices, err := onvif.DiscoveryStreamingDevices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, device := range devices {
			u, err := url.Parse(device.URL)
			if err != nil {
				log.Warn().Str("url", device.URL).Msg("[onvif] broken")
				continue
			}

			if u.Scheme != "http" {
				log.Warn().Str("url", device.URL).Msg("[onvif] unsupported")
				continue
			}

			u.Scheme = "onvif"
			u.User = url.UserPassword("user", "pass")

			if u.Path == onvif.PathDevice {
				u.Path = ""
			}

			items = append(items, &api.Source{
				Name: u.Host,
				URL:  u.String(),
				Info: device.Name + " " + device.Hardware,
			})
		}
	} else {
		client, err := onvif.NewClient(src)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if l := log.Trace(); l.Enabled() {
			b, _ := client.MediaRequest(onvif.MediaGetProfiles)
			l.Msgf("[onvif] src=%s profiles:\n%s", src, b)
		}

		name, err := client.GetName()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokens, err := client.GetProfilesTokens()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for i, token := range tokens {
			items = append(items, &api.Source{
				Name: name + " stream" + strconv.Itoa(i),
				URL:  src + "?subtype=" + token,
			})
		}

		if len(tokens) > 0 && client.HasSnapshots() {
			items = append(items, &api.Source{
				Name: name + " snapshot",
				URL:  src + "?subtype=" + tokens[0] + "&snapshot",
			})
		}
	}

	api.ResponseSources(w, items)
}
