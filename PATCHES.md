# Local patches

Branch `blueiris-hevc-noap` = upstream tag `v1.9.14` + the commits below.
Rebuild: `go build -o go2rtc-noap .` (binary is git-ignored).
Upgrade: `git fetch upstream-tag && git rebase <new tag>`; re-run the build; re-verify Blue Iris shows the HEVC streams.

## h265: send VPS/SPS/PPS as single NAL units (no aggregation packets)

`pkg/h265/rtp.go`: `RTPPay` now constructs `&Payloader{SkipAggregation: true}`.

Why: Blue Iris's H.265 RTSP receiver cannot decode RTP aggregation packets (NAL type 48).
With the stock payloader every keyframe's VPS/SPS/PPS arrive bundled in one AP; Blue Iris reads the
stream but never decodes a frame ("No signal" / "network retry"). Cameras send parameter sets as single
NAL units, which is what this patch reproduces. Pair it with `rtsp: pkt_size: 1400` in the config.
Verified 2026-09-25 against Blue Iris 5 with a Eufy S350 4K HEVC stream; see
`~/dev/ops/systems/eufy.md` and the eufy-relay repo.

## aac: reject impossible ADTS frame sizes instead of panicking

`pkg/aac/producer.go`: the raw ADTS reader validates the sync word and that the declared frame size exceeds the
header size before subtracting; a corrupt header now ends the producer with an error (go2rtc restarts the exec
source) instead of underflowing uint16 and reading a 65,535-byte "frame".
`pkg/aac/rtp.go`: `RTPPay` drops empty or >65,531-byte units (its buffer size is computed in uint16).

Why: with a raw ADTS `exec:` source, one malformed audio header crashed the whole process
(`index out of range [1] with length 1` in `aac.RTPPay`), taking every stream down (2026-10-06 03:13).
