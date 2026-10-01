// Package demo assembles palette perspective, an authored ribbon and YM logos.
package demo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"io"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	playback "github.com/olivierh59500/democonstructionkit/sound/ebiten"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/go-larcociel5/assets"
	"github.com/olivierh59500/go-larcociel5/internal/source"
)

const Width, Height, FPS = 320, 200, 50

type Game struct {
	clock                                    *source.Clock
	images                                   []*ebiten.Image
	batch                                    *sprites.ImageSlots
	slots                                    [50]sprites.ImageSlot
	background, body, bats, dummy, colorBank *ebiten.Image
	scroll                                   *scrolling.Scrolling
	font                                     [30]*ebiten.Image
	fontColors                               []byte
	message                                  []byte
	stencil                                  [272 * 15 * 4]byte
	colors                                   [100 * 16 * 4]byte
	player                                   *playback.Player
	visual                                   *sound.Stream
	pcm                                      [960 * 8]byte
	shader                                   *ebiten.Shader
	palette                                  [64]float32
	raster                                   [64 * 4]float32
	uniforms                                 map[string]any
	closed                                   bool
}

func resource(name string) ([]byte, error) { return assets.Files.ReadFile("original/" + name) }
func rgb(dst []float32, w uint16) {
	dst[0] = float32(w>>8&7) * 34 / 255
	dst[1] = float32(w>>4&7) * 34 / 255
	dst[2] = float32(w&7) * 34 / 255
	dst[3] = 1
}

func NewGame(mute bool) (_ *Game, err error) {
	g := &Game{}
	defer func() {
		if err != nil {
			g.Close()
		}
	}()
	m, err := resource("initial-motion.bin")
	if err != nil {
		return nil, err
	}
	h, err := resource("initial-history.bin")
	if err != nil {
		return nil, err
	}
	g.clock, err = source.NewClock(m, h)
	if err != nil {
		return nil, err
	}
	load := func(name string) (*ebiten.Image, error) {
		b, e := resource(name)
		if e != nil {
			return nil, e
		}
		im, _, e := image.Decode(bytes.NewReader(b))
		if e != nil {
			return nil, e
		}
		texture := ebiten.NewImageFromImage(im)
		g.images = append(g.images, texture)
		return texture, nil
	}
	g.background, err = load("background.png")
	if err != nil {
		return nil, err
	}
	if _, err = load("ribbon.png"); err != nil {
		return nil, err
	}
	for i := 0; i < 15; i++ {
		if _, err = load(fmt.Sprintf("bat-%d.png", i)); err != nil {
			return nil, err
		}
	}
	atlas, err := load("font.png")
	if err != nil {
		return nil, err
	}
	for i := range g.font {
		g.font[i] = atlas.SubImage(image.Rect(i*15, 0, i*15+15, 17)).(*ebiten.Image)
	}
	g.fontColors, err = resource("font-colors.bin")
	if err != nil {
		return nil, err
	}
	g.message, err = resource("message.txt")
	if err != nil {
		return nil, err
	}
	g.body = ebiten.NewImage(Width, Height)
	g.bats = ebiten.NewImage(Width, Height)
	g.dummy = ebiten.NewImage(15, 272)
	g.colorBank = ebiten.NewImage(16, 100)
	g.batch, err = sprites.NewImageSlots(sprites.ImageSlotsConfig{Images: g.images, MaxSlots: 50})
	if err != nil {
		return nil, err
	}
	g.scroll, err = scrolling.New(scrolling.Config{Vertical: true, GlyphWindow: &scrolling.GlyphWindowConfig{Count: 16, Advance: 17, Glyph: func(slot int) scrolling.Glyph {
		position := max(0, g.clock.Tick-1)
		current := position / 17 % len(g.message)
		character := byte(' ')
		if current+slot < len(g.message) {
			character = g.message[current+slot]
		}
		glyph := glyphIndex(character)
		return scrolling.Glyph{Image: g.font[glyph], Rune: rune(character), Y: -float64(position % 17), Advance: 17}
	}}})
	if err != nil {
		return nil, err
	}
	pal, err := resource("palette.bin")
	if err != nil {
		return nil, err
	}
	for i := 0; i < 16; i++ {
		rgb(g.palette[i*4:], binary.BigEndian.Uint16(pal[i*2:]))
	}
	rast, err := resource("raster.bin")
	if err != nil {
		return nil, err
	}
	for i := 0; i < 64; i++ {
		w := uint16(0x700)
		if i < len(rast)/2 {
			w = binary.BigEndian.Uint16(rast[i*2:])
		}
		rgb(g.raster[i*4:], w)
	}
	g.uniforms = map[string]any{"Palette": g.palette[:], "Raster": g.raster[:]}
	g.shader, err = ebiten.NewShader([]byte(paletteShader))
	if err != nil {
		return nil, err
	}
	music, err := resource("music.ym")
	if err != nil {
		return nil, err
	}
	g.visual, err = sound.Open("music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
	if err != nil {
		return nil, err
	}
	if !mute {
		g.player, err = playback.Open(nil, "music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
		if err != nil {
			return nil, err
		}
		g.player.Play()
	}
	return g, nil
}
func glyphIndex(character byte) int {
	if character == ' ' {
		return 0
	}
	if character == '.' {
		return 27
	}
	if character == ',' {
		return 28
	}
	if character == '\'' {
		return 29
	}
	return max(0, min(26, int(character)-64))
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	g.clock.Step()
	if _, err := io.ReadFull(g.visual, g.pcm[:]); err != nil {
		return err
	}
	g.body.Clear()
	for row := range g.slots {
		g.slots[row] = sprites.ImageSlot{Image: 1, Source: image.Rect(0, row, 112, row+1), X: float64(g.clock.History[row]), Y: float64(g.clock.Y) + float64(row)}
	}
	if err := g.batch.SetSlots(g.slots[:]); err != nil {
		return err
	}
	g.batch.Draw(g.body)
	g.bats.Clear()
	if regs, ok := g.visual.YMRegisters(); ok {
		for i, reg := range []int{8, 10, 9} {
			volume := max(1, int(regs[reg]&15))
			g.slots[i] = sprites.ImageSlot{Image: 2 + 15 - volume, X: float64(i * 128)}
		}
		if err := g.batch.SetSlots(g.slots[:3]); err != nil {
			return err
		}
		g.batch.Draw(g.bats)
	}
	clear(g.stencil[:])
	state := scrolling.IdentityState()
	state.Paint = func(_ *ebiten.Image, sample scrolling.Sample, _ ebiten.DrawImageOptions) {
		glyph := glyphIndex(byte(sample.Glyph.Rune))
		top := int(sample.Y)
		for row := 0; row < 17; row++ {
			y := top + row
			if y < 0 || y >= 272 {
				continue
			}
			for col := 0; col < 15; col++ {
				w := binary.BigEndian.Uint16(g.fontColors[(glyph*255+row*15+col)*2:])
				at := (y*15 + col) * 4
				g.stencil[at] = byte(w>>8&7) * 34
				g.stencil[at+1] = byte(w>>4&7) * 34
				g.stencil[at+2] = byte(w&7) * 34
				g.stencil[at+3] = 255
			}
		}
	}
	g.scroll.DrawAt(g.dummy, state)
	if err := g.scroll.Err(); err != nil {
		return err
	}
	clear(g.colors[:])
	output, phase, step := 0, 0, 20
	for row := 0; output < 100 && row < 272; row++ {
		phase += step
		if phase < 80 {
			continue
		}
		phase -= 80
		step++
		copy(g.colors[(output*16+1)*4:(output*16+16)*4], g.stencil[row*15*4:(row+1)*15*4])
		output++
	}
	g.colorBank.WritePixels(g.colors[:])
	return nil
}
func (g *Game) Draw(dst *ebiten.Image) {
	op := ebiten.DrawRectShaderOptions{Images: [4]*ebiten.Image{g.background, g.body, g.bats, g.colorBank}, Uniforms: g.uniforms, Blend: ebiten.BlendCopy}
	v := [...]ebiten.Vertex{{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}, {DstX: Width, DstY: 0, SrcX: Width, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}, {DstX: 0, DstY: Height, SrcX: 0, SrcY: Height, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}, {DstX: Width, DstY: Height, SrcX: Width, SrcY: Height, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}}
	dst.DrawTrianglesShader(v[:], quadIndices[:], g.shader, &ebiten.DrawTrianglesShaderOptions{Images: op.Images, Uniforms: g.uniforms, Blend: ebiten.BlendCopy})
}
func (*Game) Layout(int, int) (int, int) { return Width, Height }
func (g *Game) Tick() int                { return g.clock.Tick }
func (g *Game) Close() {
	if g == nil || g.closed {
		return
	}
	g.closed = true
	if g.player != nil {
		g.player.Close()
	}
	if g.visual != nil {
		g.visual.Close()
	}
	if g.scroll != nil {
		g.scroll.Close()
	}
	if g.batch != nil {
		g.batch.Close()
	}
	if g.shader != nil {
		g.shader.Deallocate()
	}
	for _, im := range append(g.images, g.body, g.bats, g.dummy, g.colorBank) {
		if im != nil {
			im.Deallocate()
		}
	}
}

const paletteShader = `//kage:unit pixels
package main
var Palette [16]vec4
var Raster [64]vec4
func Fragment(position vec4,source vec2,color vec4)vec4{
 p:=source-imageSrc0Origin();i:=int(clamp(floor(imageSrc0At(source).r*15+.5),0,15));ribbon:=imageSrc1At(source);bat:=imageSrc2At(source)
 if ribbon.a>0{i=(i/4)*4+int(floor(ribbon.r*15+.5))};if p.y<32{i=i%4+int(floor(bat.r*15+.5))}
 if p.y>=100&&i>0{return imageSrc3At(imageSrc0Origin()+vec2(float(i)+.5,p.y-100))*color}
 if i==4&&p.y<64{return Raster[int(clamp(p.y,0,63))]*color};return Palette[i]*color
}
`

var quadIndices = [...]uint16{0, 1, 2, 1, 3, 2}
