// Package source retains the ribbon's native signed-word trail and pauses.
package source

import (
	"encoding/binary"
	"fmt"
)

type Clock struct {
	Tick, Counter, Hold  int
	X, Y, VX, VY, AX, AY int16
	History              [50]int16
}

func NewClock(motion, history []byte) (*Clock, error) {
	if len(motion) != 12 || len(history) != 100 {
		return nil, fmt.Errorf("source: incomplete ribbon data")
	}
	w := func(at int) int16 { return int16(binary.BigEndian.Uint16(motion[at:])) }
	c := &Clock{VX: w(0), AX: w(2), X: w(4), VY: w(6), AY: w(8), Y: w(10), Hold: 62}
	for i := range c.History {
		c.History[i] = int16(binary.BigEndian.Uint16(history[i*2:]))
	}
	return c, nil
}
func (c *Clock) Step() {
	c.Tick++
	if c.Tick%320 == 0 {
		c.Hold = 61
	}
	copy(c.History[:49], c.History[1:])
	if c.Hold > 0 {
		c.Hold--
	} else {
		c.X += c.VX
		c.Y += c.VY
		c.Counter++
		if c.Counter == 5 {
			c.Counter = 0
			c.VX += c.AX
			c.VY += c.AY
			if c.VX == 6 || c.VX == -6 {
				c.AX = -c.AX
			}
			if c.VY == 3 || c.VY == -3 {
				c.AY = -c.AY
			}
		}
	}
	c.History[49] = c.X
}
