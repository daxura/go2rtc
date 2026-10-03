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
