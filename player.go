package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const maxPlayers int = 4

// Player represents an active user in a Lobby.
// Check for the zero value of lastSeen to see if a
// Player has been initialized.
type Player struct {
	Name     string `json:"name"`
	Color    RGB    `json:"color"`
	id       uuid.UUID
	lastSeen time.Time
}

func (m *Player) String() string {
	if m.lastSeen.IsZero() {
		return "<empty>"
	} else {
		return fmt.Sprintf("%-8s %s %s %s", m.Name, &m.Color, m.lastSeen, m.id)
	}
}

func newPlayer(name string) Player {
	return Player{
		Name:     name,
		id:       uuid.New(),
		lastSeen: time.Now(),
	}
}

type RGB struct {
	rgb [3]uint8
}

func (m *RGB) String() string {
	return fmt.Sprintf("#%02x%02x%02x", m.rgb[0], m.rgb[1], m.rgb[2])
}

func (m *RGB) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *RGB) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return m.Parse(s)
}

// Parse sets RGB values based on a hex string of the form
// - #RGB
// - #RRGGBB
// - RGB
// - RRGGBB
func (m *RGB) Parse(s string) error {
	if len(s) == 0 {
		return errors.New("Invalid RGB string")
	}
	if s[0:1] == "#" {
		s = s[1:]
	}
	if len(s) == 3 {
		var full_rgb []rune = make([]rune, 0, 6)
		for _, r := range s {
			full_rgb = append(full_rgb, r, r)
		}
		s = string(full_rgb)
	}
	if len(s) != 6 {
		return errors.New("Invalid RGB string")
	}
	for i := range m.rgb {
		val, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8)
		if err != nil {
			return errors.New("Invalid RGB string")
		}
		m.rgb[i] = uint8(val)
	}
	return nil
}

const defaultRGBTolerance int = 24

// IsNearDuplicate returns true if to RGB have the same rgb values
// within a given tolerance.
func (m *RGB) IsNearDuplicate(n *RGB, tolerance int) bool {
	var diffSum int
	for i := range m.rgb {
		if m.rgb[i] >= n.rgb[i] {
			diffSum += int(m.rgb[i]) - int(n.rgb[i])
		} else {
			diffSum += int(n.rgb[i]) - int(m.rgb[i])
		}
	}
	return diffSum <= tolerance
}

// Randomize sets the rgb values randomly
func (m *RGB) Randomize(n *RGB) {
	for i := range m.rgb {
		m.rgb[i] = uint8(rand.UintN(256))
	}
}

// Randomize, but avoid duplicates. Ensures that at least one color is at least `tolerance` away
// from everything in `avoid`
func (m *RGB) RandomizeAvoidingDuplicates(tolerance int, avoid ...[]*RGB) error {
	// pick a color to be constrained by tolerance
	// if there is no available color, will try the next color
	startColor := rand.IntN(len(m.rgb))
	var constrainedColor int
	var found bool
	for i := 0; i < len(m.rgb) && !found; i++ {
		constrainedColor = (startColor + i) % len(m.rgb)

		// determine availability for constrainedColor
		var mask [256]bool
		for _, arg := range avoid {
			for _, a := range arg {
				avoidIndexStart := max(int(a.rgb[constrainedColor])-tolerance, 0)
				avoidIndexEnd := min(int(a.rgb[constrainedColor])+tolerance, len(mask)-1)
				for avoidIndex := avoidIndexStart; avoidIndex <= avoidIndexEnd; avoidIndex++ {
					mask[avoidIndex] = true
				}
			}
		}
		availableIndexes := make([]int, 0, 256)
		for mIndex, b := range mask {
			if !b {
				availableIndexes = append(availableIndexes, mIndex)
			}
		}
		if len(availableIndexes) == 0 {
			continue
		}

		// choose a color
		m.rgb[constrainedColor] = uint8(availableIndexes[rand.IntN(len(availableIndexes))])
		found = true
	}

	if !found {
		return errors.New("No available colors match the tolerance constraint")
	}

	// set the remaining colors
	for i := 1; i < len(m.rgb); i++ {
		m.rgb[(constrainedColor+i)%len(m.rgb)] = uint8(rand.UintN(256))
	}

	return nil
}
