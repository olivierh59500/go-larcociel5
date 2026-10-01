package source

import (
	"encoding/json"
	"github.com/olivierh59500/go-larcociel5/assets"
	"os"
	"reflect"
	"testing"
)

func TestRibbonMatchesNativeCheckpoint(t *testing.T) {
	m, e := assets.Files.ReadFile("original/initial-motion.bin")
	if e != nil {
		t.Fatal(e)
	}
	h, e := assets.Files.ReadFile("original/initial-history.bin")
	if e != nil {
		t.Fatal(e)
	}
	c, e := NewClock(m, h)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("testdata/native-ribbon-149.json")
	if e != nil {
		t.Fatal(e)
	}
	var want struct {
		Steps, Counter, Hold             int
		Velocity, Acceleration, Position [2]int16
		History                          [50]int16
	}
	if e = json.Unmarshal(b, &want); e != nil {
		t.Fatal(e)
	}
	for range want.Steps {
		c.Step()
	}
	if [2]int16{c.X, c.Y} != want.Position || [2]int16{c.VX, c.VY} != want.Velocity || [2]int16{c.AX, c.AY} != want.Acceleration || c.Counter != want.Counter || c.Hold != want.Hold || !reflect.DeepEqual(c.History, want.History) {
		t.Fatalf("ribbon differs from native: position=(%d,%d) velocity=(%d,%d) hold=%d counter=%d", c.X, c.Y, c.VX, c.VY, c.Hold, c.Counter)
	}
}
