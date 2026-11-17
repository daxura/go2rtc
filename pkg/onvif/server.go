package onvif

import (
	"bytes"
	"regexp"
	"time"
)

const ServiceGetServiceCapabilities = "GetServiceCapabilities"

const (
	DeviceGetCapabilities          = "GetCapabilities"
	DeviceGetDeviceInformation     = "GetDeviceInformation"
	DeviceGetDiscoveryMode         = "GetDiscoveryMode"
	DeviceGetDNS                   = "GetDNS"
	DeviceGetHostname              = "GetHostname"
	DeviceGetNetworkDefaultGateway = "GetNetworkDefaultGateway"
	DeviceGetNetworkInterfaces     = "GetNetworkInterfaces"
	DeviceGetNetworkProtocols      = "GetNetworkProtocols"
	DeviceGetNTP                   = "GetNTP"
	DeviceGetScopes                = "GetScopes"
	DeviceGetServices              = "GetServices"
	DeviceGetSystemDateAndTime     = "GetSystemDateAndTime"
	DeviceSetSystemDateAndTime     = "SetSystemDateAndTime"
	DeviceSystemReboot             = "SystemReboot"
)

const (
	MediaGetAudioEncoderConfigurations       = "GetAudioEncoderConfigurations"
	MediaGetAudioSources                     = "GetAudioSources"
	MediaGetAudioSourceConfigurations        = "GetAudioSourceConfigurations"
	MediaGetProfile                          = "GetProfile"
	MediaGetProfiles                         = "GetProfiles"
	MediaGetSnapshotUri                      = "GetSnapshotUri"
	MediaGetStreamUri                        = "GetStreamUri"
	MediaGetVideoEncoderConfiguration        = "GetVideoEncoderConfiguration"
	MediaGetVideoEncoderConfigurations       = "GetVideoEncoderConfigurations"
	MediaGetVideoEncoderConfigurationOptions = "GetVideoEncoderConfigurationOptions"
	MediaGetVideoSources                     = "GetVideoSources"
	MediaGetVideoSourceConfiguration         = "GetVideoSourceConfiguration"
	MediaGetVideoSourceConfigurations        = "GetVideoSourceConfigurations"
)

func GetRequestAction(b []byte) string {
	// <soap-env:Body><ns0:GetCapabilities xmlns:ns0="http://www.onvif.org/ver10/device/wsdl">
	// <v:Body><GetSystemDateAndTime xmlns="http://www.onvif.org/ver10/device/wsdl" /></v:Body>
	re := regexp.MustCompile(`Body[^<]+<([^ />]+)`)
	m := re.FindSubmatch(b)
	if len(m) != 2 {
		return ""
	}
	if i := bytes.IndexByte(m[1], ':'); i > 0 {
		return string(m[1][i+1:])
	}
	return string(m[1])
}

func GetCapabilitiesResponse(host string) []byte {
	e := NewEnvelope()
	e.Appendf(`<tds:GetCapabilitiesResponse>
	<tds:Capabilities>
		<tt:Device>
			<tt:XAddr>http://%s/onvif/device_service</tt:XAddr>
		</tt:Device>
		<tt:Media>
			<tt:XAddr>http://%s/onvif/media_service</tt:XAddr>
			<tt:StreamingCapabilities>
				<tt:RTPMulticast>false</tt:RTPMulticast>
				<tt:RTP_TCP>false</tt:RTP_TCP>
				<tt:RTP_RTSP_TCP>true</tt:RTP_RTSP_TCP>
			</tt:StreamingCapabilities>
		</tt:Media>
	</tds:Capabilities>
</tds:GetCapabilitiesResponse>`, host, host)
	return e.Bytes()
}

func GetServicesResponse(host string) []byte {
	e := NewEnvelope()
	e.Appendf(`<tds:GetServicesResponse>
	<tds:Service>
		<tds:Namespace>http://www.onvif.org/ver10/device/wsdl</tds:Namespace>
		<tds:XAddr>http://%s/onvif/device_service</tds:XAddr>
		<tds:Version><tt:Major>2</tt:Major><tt:Minor>5</tt:Minor></tds:Version>
	</tds:Service>
	<tds:Service>
		<tds:Namespace>http://www.onvif.org/ver10/media/wsdl</tds:Namespace>
		<tds:XAddr>http://%s/onvif/media_service</tds:XAddr>
		<tds:Version><tt:Major>2</tt:Major><tt:Minor>5</tt:Minor></tds:Version>
	</tds:Service>
</tds:GetServicesResponse>`, host, host)
	return e.Bytes()
}

func GetSystemDateAndTimeResponse() []byte {
	loc := time.Now()
	utc := loc.UTC()

	e := NewEnvelope()
	e.Appendf(`<tds:GetSystemDateAndTimeResponse>
	<tds:SystemDateAndTime>
		<tt:DateTimeType>NTP</tt:DateTimeType>
		<tt:DaylightSavings>true</tt:DaylightSavings>
		<tt:TimeZone>
			<tt:TZ>%s</tt:TZ>
		</tt:TimeZone>
		<tt:UTCDateTime>
			<tt:Time><tt:Hour>%d</tt:Hour><tt:Minute>%d</tt:Minute><tt:Second>%d</tt:Second></tt:Time>
			<tt:Date><tt:Year>%d</tt:Year><tt:Month>%d</tt:Month><tt:Day>%d</tt:Day></tt:Date>
		</tt:UTCDateTime>
		<tt:LocalDateTime>
			<tt:Time><tt:Hour>%d</tt:Hour><tt:Minute>%d</tt:Minute><tt:Second>%d</tt:Second></tt:Time>
			<tt:Date><tt:Year>%d</tt:Year><tt:Month>%d</tt:Month><tt:Day>%d</tt:Day></tt:Date>
		</tt:LocalDateTime>
	</tds:SystemDateAndTime>
</tds:GetSystemDateAndTimeResponse>`,
		GetPosixTZ(loc),
		utc.Hour(), utc.Minute(), utc.Second(), utc.Year(), utc.Month(), utc.Day(),
		loc.Hour(), loc.Minute(), loc.Second(), loc.Year(), loc.Month(), loc.Day(),
	)
	return e.Bytes()
}

func GetDeviceInformationResponse(manuf, model, firmware, serial string) []byte {
	e := NewEnvelope()
	e.Appendf(`<tds:GetDeviceInformationResponse>
	<tds:Manufacturer>%s</tds:Manufacturer>
	<tds:Model>%s</tds:Model>
	<tds:FirmwareVersion>%s</tds:FirmwareVersion>
	<tds:SerialNumber>%s</tds:SerialNumber>
	<tds:HardwareId>1.00</tds:HardwareId>
</tds:GetDeviceInformationResponse>`, manuf, model, firmware, serial)
	return e.Bytes()
}

// Encoder describes the (single) video encoder the ONVIF server advertises.
// go2rtc does not transcode, so this is metadata about the stream it serves;
// clients such as UniFi Protect size their decode pipeline from it.
type Encoder struct {
	Encoding string // H264 or H265
	Width    int
	Height   int
	FPS      int
	Bitrate  int // kbps
	Audio    Audio
}

// Audio describes the audio track served next to the video (zero value = none).
// Clients such as UniFi Protect take "the camera has a microphone" from it.
type Audio struct {
	Encoding   string // G711, G726 or AAC
	Bitrate    int    // kbps
	SampleRate int    // kHz
}

var DefaultEncoder = Encoder{Encoding: "H264", Width: 1920, Height: 1080, FPS: 30, Bitrate: 8192}

func (enc Encoder) orDefault() Encoder {
	if enc.Encoding == "" {
		enc.Encoding = DefaultEncoder.Encoding
	}
	if enc.Width == 0 || enc.Height == 0 {
		enc.Width, enc.Height = DefaultEncoder.Width, DefaultEncoder.Height
	}
	if enc.FPS == 0 {
		enc.FPS = DefaultEncoder.FPS
	}
	if enc.Bitrate == 0 {
		enc.Bitrate = DefaultEncoder.Bitrate
	}
	return enc
}

// EncoderFunc returns the encoder to advertise for a stream name.
type EncoderFunc func(name string) Encoder

func GetProfilesResponse(names []string, enc EncoderFunc) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetProfilesResponse>`)
	for _, name := range names {
		appendProfile(e, "Profiles", name, enc(name))
	}
	e.Append(`</trt:GetProfilesResponse>`)
	return e.Bytes()
}

func GetProfileResponse(name string, enc EncoderFunc) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetProfileResponse>`)
	appendProfile(e, "Profile", name, enc(name))
	e.Append(`</trt:GetProfileResponse>`)
	return e.Bytes()
}

func appendProfile(e *Envelope, tag, name string, enc Encoder) {
	// go2rtc name = ONVIF Profile Name = ONVIF Profile token
	e.Appendf(`<trt:%s token="%s" fixed="true">`, tag, name)
	e.Appendf(`<tt:Name>%s</tt:Name>`, name)
	appendVideoSourceConfiguration(e, "VideoSourceConfiguration", name)
	if enc.Audio.Encoding != "" {
		appendAudioSourceConfiguration(e, "tt:AudioSourceConfiguration")
	}
	appendVideoEncoderConfiguration(e, "VideoEncoderConfiguration", enc)
	if enc.Audio.Encoding != "" {
		appendAudioEncoderConfiguration(e, "tt:AudioEncoderConfiguration", enc.Audio)
	}
	e.Appendf(`</trt:%s>`, tag)
}

// tag is the qualified element name: tt:… inside a profile, trt:Configurations in a list response
func appendAudioSourceConfiguration(e *Envelope, tag string) {
	e.Appendf(`<%s token="asc"><tt:Name>ASC</tt:Name><tt:UseCount>1</tt:UseCount><tt:SourceToken>audio</tt:SourceToken></%s>`, tag, tag)
}

func appendAudioEncoderConfiguration(e *Envelope, tag string, a Audio) {
	e.Appendf(`<%s token="aec"><tt:Name>AEC</tt:Name><tt:UseCount>1</tt:UseCount><tt:Encoding>%s</tt:Encoding><tt:Bitrate>%d</tt:Bitrate><tt:SampleRate>%d</tt:SampleRate><tt:SessionTimeout>PT10S</tt:SessionTimeout></%s>`,
		tag, a.Encoding, a.Bitrate, a.SampleRate, tag)
}

func GetAudioSourcesResponse() []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetAudioSourcesResponse><trt:AudioSources token="audio"><tt:Channels>1</tt:Channels></trt:AudioSources></trt:GetAudioSourcesResponse>`)
	return e.Bytes()
}

func GetAudioSourceConfigurationsResponse() []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetAudioSourceConfigurationsResponse>`)
	appendAudioSourceConfiguration(e, "trt:Configurations")
	e.Append(`</trt:GetAudioSourceConfigurationsResponse>`)
	return e.Bytes()
}

func GetAudioEncoderConfigurationsResponse(a Audio) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetAudioEncoderConfigurationsResponse>`)
	appendAudioEncoderConfiguration(e, "trt:Configurations", a)
	e.Append(`</trt:GetAudioEncoderConfigurationsResponse>`)
	return e.Bytes()
}

func GetVideoSourcesResponse(names []string, encf EncoderFunc) []byte {
	// go2rtc name = ONVIF VideoSource token
	e := NewEnvelope()
	e.Append(`<trt:GetVideoSourcesResponse>`)
	for _, name := range names {
		enc := encf(name).orDefault()
		e.Appendf(`<trt:VideoSources token="%s">
	<tt:Framerate>30.000000</tt:Framerate>
	<tt:Resolution><tt:Width>%d</tt:Width><tt:Height>%d</tt:Height></tt:Resolution>
</trt:VideoSources>`, name, enc.Width, enc.Height)
	}
	e.Append(`</trt:GetVideoSourcesResponse>`)
	return e.Bytes()
}

func GetVideoSourceConfigurationsResponse(names []string) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetVideoSourceConfigurationsResponse>`)
	for _, name := range names {
		appendVideoSourceConfiguration(e, "Configurations", name)
	}
	e.Append(`</trt:GetVideoSourceConfigurationsResponse>`)
	return e.Bytes()
}

func GetVideoSourceConfigurationResponse(name string) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetVideoSourceConfigurationResponse>`)
	appendVideoSourceConfiguration(e, "Configuration", name)
	e.Append(`</trt:GetVideoSourceConfigurationResponse>`)
	return e.Bytes()
}

func appendVideoSourceConfiguration(e *Envelope, tag, name string) {
	// go2rtc name = ONVIF VideoSourceConfiguration token
	e.Appendf(`<tt:%s token="%s" fixed="true">
	<tt:Name>VSC</tt:Name>
	<tt:SourceToken>%s</tt:SourceToken>
	<tt:Bounds x="0" y="0" width="1920" height="1080"></tt:Bounds>
</tt:%s>`, tag, name, name, tag)
}

func GetVideoEncoderConfigurationsResponse(enc Encoder) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetVideoEncoderConfigurationsResponse>`)
	appendVideoEncoderConfiguration(e, "VideoEncoderConfigurations", enc)
	e.Append(`</trt:GetVideoEncoderConfigurationsResponse>`)
	return e.Bytes()
}

func GetVideoEncoderConfigurationResponse(enc Encoder) []byte {
	e := NewEnvelope()
	e.Append(`<trt:GetVideoEncoderConfigurationResponse>`)
	appendVideoEncoderConfiguration(e, "VideoEncoderConfiguration", enc)
	e.Append(`</trt:GetVideoEncoderConfigurationResponse>`)
	return e.Bytes()
}

func appendVideoEncoderConfiguration(e *Envelope, tag string, enc Encoder) {
	enc = enc.orDefault()
	// empty `RateControl` important for UniFi Protect
	e.Appendf(`<tt:%s token="vec">
		<tt:Name>VEC</tt:Name>
        <tt:UseCount>1</tt:UseCount>
		<tt:Encoding>%s</tt:Encoding>
		<tt:Resolution><tt:Width>%d</tt:Width><tt:Height>%d</tt:Height></tt:Resolution>
        <tt:Quality>0</tt:Quality>
		<tt:RateControl><tt:FrameRateLimit>%d</tt:FrameRateLimit><tt:EncodingInterval>1</tt:EncodingInterval><tt:BitrateLimit>%d</tt:BitrateLimit></tt:RateControl>`,
		tag, enc.Encoding, enc.Width, enc.Height, enc.FPS, enc.Bitrate)
	if enc.Encoding == "H264" {
		e.Append(`
        <tt:H264><tt:GovLength>10</tt:GovLength><tt:H264Profile>Main</tt:H264Profile></tt:H264>`)
	}
	e.Appendf(`
        <tt:SessionTimeout>PT10S</tt:SessionTimeout>
	</tt:%s>`, tag)
}

func GetVideoEncoderConfigurationOptionsResponse(enc Encoder) []byte {
	enc = enc.orDefault()
	e := NewEnvelope()
	e.Appendf(`<trt:GetVideoEncoderConfigurationOptionsResponse>
   <trt:Options>
       <tt:QualityRange><tt:Min>1</tt:Min><tt:Max>6</tt:Max></tt:QualityRange>
	   <tt:%[1]s>
		   <tt:ResolutionsAvailable><tt:Width>%[2]d</tt:Width><tt:Height>%[3]d</tt:Height></tt:ResolutionsAvailable>
		   <tt:GovLengthRange><tt:Min>0</tt:Min><tt:Max>100</tt:Max></tt:GovLengthRange>
		   <tt:FrameRateRange><tt:Min>1</tt:Min><tt:Max>%[4]d</tt:Max></tt:FrameRateRange>
		   <tt:EncodingIntervalRange><tt:Min>1</tt:Min><tt:Max>100</tt:Max></tt:EncodingIntervalRange>
           <tt:%[1]sProfilesSupported>Main</tt:%[1]sProfilesSupported>
	   </tt:%[1]s>
   </trt:Options>
</trt:GetVideoEncoderConfigurationOptionsResponse>`, enc.Encoding, enc.Width, enc.Height, enc.FPS)
	return e.Bytes()
}

func GetStreamUriResponse(uri string) []byte {
	e := NewEnvelope()
	e.Appendf(`<trt:GetStreamUriResponse><trt:MediaUri><tt:Uri>%s</tt:Uri></trt:MediaUri></trt:GetStreamUriResponse>`, uri)
	return e.Bytes()
}

func GetSnapshotUriResponse(uri string) []byte {
	e := NewEnvelope()
	e.Appendf(`<trt:GetSnapshotUriResponse><trt:MediaUri><tt:Uri>%s</tt:Uri></trt:MediaUri></trt:GetSnapshotUriResponse>`, uri)
	return e.Bytes()
}

func StaticResponse(operation string) []byte {
	switch operation {
	case DeviceGetSystemDateAndTime:
		return GetSystemDateAndTimeResponse()
	case MediaGetVideoEncoderConfiguration:
		return GetVideoEncoderConfigurationResponse(DefaultEncoder)
	case MediaGetVideoEncoderConfigurations:
		return GetVideoEncoderConfigurationsResponse(DefaultEncoder)
	}

	e := NewEnvelope()
	e.Append(responses[operation])
	return e.Bytes()
}

var responses = map[string]string{
	ServiceGetServiceCapabilities: `<trt:GetServiceCapabilitiesResponse>
	<trt:Capabilities SnapshotUri="true" Rotation="false" VideoSourceMode="false" OSD="false" TemporaryOSDText="false" EXICompression="false">
		<trt:StreamingCapabilities RTPMulticast="false" RTP_TCP="false" RTP_RTSP_TCP="true" NonAggregateControl="false" NoRTSPStreaming="false" />
	</trt:Capabilities>
</trt:GetServiceCapabilitiesResponse>`,

	DeviceGetDiscoveryMode:         `<tds:GetDiscoveryModeResponse><tds:DiscoveryMode>Discoverable</tds:DiscoveryMode></tds:GetDiscoveryModeResponse>`,
	DeviceGetDNS:                   `<tds:GetDNSResponse><tds:DNSInformation /></tds:GetDNSResponse>`,
	DeviceGetHostname:              `<tds:GetHostnameResponse><tds:HostnameInformation /></tds:GetHostnameResponse>`,
	DeviceGetNetworkDefaultGateway: `<tds:GetNetworkDefaultGatewayResponse><tds:NetworkGateway /></tds:GetNetworkDefaultGatewayResponse>`,
	DeviceGetNTP:                   `<tds:GetNTPResponse><tds:NTPInformation /></tds:GetNTPResponse>`,
	DeviceSetSystemDateAndTime:     `<tds:SetSystemDateAndTimeResponse />`,
	DeviceSystemReboot:             `<tds:SystemRebootResponse><tds:Message>OK</tds:Message></tds:SystemRebootResponse>`,

	DeviceGetNetworkInterfaces: `<tds:GetNetworkInterfacesResponse />`,
	DeviceGetNetworkProtocols:  `<tds:GetNetworkProtocolsResponse />`,
	DeviceGetScopes: `<tds:GetScopesResponse>
	<tds:Scopes><tt:ScopeDef>Fixed</tt:ScopeDef><tt:ScopeItem>onvif://www.onvif.org/name/go2rtc</tt:ScopeItem></tds:Scopes>
	<tds:Scopes><tt:ScopeDef>Fixed</tt:ScopeDef><tt:ScopeItem>onvif://www.onvif.org/location/github</tt:ScopeItem></tds:Scopes>
	<tds:Scopes><tt:ScopeDef>Fixed</tt:ScopeDef><tt:ScopeItem>onvif://www.onvif.org/Profile/Streaming</tt:ScopeItem></tds:Scopes>
	<tds:Scopes><tt:ScopeDef>Fixed</tt:ScopeDef><tt:ScopeItem>onvif://www.onvif.org/type/Network_Video_Transmitter</tt:ScopeItem></tds:Scopes>
</tds:GetScopesResponse>`,

	MediaGetAudioEncoderConfigurations: `<trt:GetAudioEncoderConfigurationsResponse />`,
	MediaGetAudioSources:               `<trt:GetAudioSourcesResponse />`,
	MediaGetAudioSourceConfigurations:  `<trt:GetAudioSourceConfigurationsResponse />`,

	MediaGetVideoEncoderConfigurationOptions: `<trt:GetVideoEncoderConfigurationOptionsResponse>
   <trt:Options>
       <tt:QualityRange><tt:Min>1</tt:Min><tt:Max>6</tt:Max></tt:QualityRange>
	   <tt:H264>
		   <tt:ResolutionsAvailable><tt:Width>1920</tt:Width><tt:Height>1080</tt:Height></tt:ResolutionsAvailable>
		   <tt:GovLengthRange><tt:Min>0</tt:Min><tt:Max>100</tt:Max></tt:GovLengthRange>
		   <tt:FrameRateRange><tt:Min>1</tt:Min><tt:Max>30</tt:Max></tt:FrameRateRange>
		   <tt:EncodingIntervalRange><tt:Min>1</tt:Min><tt:Max>100</tt:Max></tt:EncodingIntervalRange>
           <tt:H264ProfilesSupported>Main</tt:H264ProfilesSupported>
	   </tt:H264>
   </trt:Options>
</trt:GetVideoEncoderConfigurationOptionsResponse>`,
}
