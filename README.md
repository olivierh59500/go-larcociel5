# Larcociel 5 Go

A Go/Ebitengine conversion of **DMA Intro 5**, an Atari ST intro by
**Larcociel of DMA**, using Demo Construction Kit **v1.0.13**.

```sh
go run ./cmd/larcociel5
go run ./cmd/larcociel5 -mute
```

Space or Escape closes the intro. The 320 × 200 scene advances at 50 Hz and
combines three Batman music meters, a moving row-deformed ribbon, an illustrated
landscape and a color scrolltext receding into a perspective floor.

The native artwork, font colors, message, acceleration program and fifty-line
ribbon history are retained. The movement test compares a 149-frame 68000
checkpoint, including the complete row history and motion counters. DCK owns
the retained row/sprite batches, literal scrolling layout and YM playback.
The palette scroller samples DCK's bounded glyph window, then uploads only a
16 × 100 color bank using the original integer perspective recurrence.
The fixed indexed floor supplies the projected column outlines. Drawing uses
no GPU pixel readback; the original YM6 soundtrack is embedded.

```sh
go test ./internal/source
go vet ./...
go run ./cmd/larcociel5 -capture captures/preview -frame 800 -frames 1
go run ./cmd/video
```

DCK's video exporter writes a three-minute H.264/AAC MP4, a PNG poster and a
JSON report to `recordings/`, at 640 × 400 and 50 fps. The website uses a
VP9/Opus WebM copy. Graphics and music share one simulation clock.

Original production: [Demozoo](https://demozoo.org/productions/79464/).
