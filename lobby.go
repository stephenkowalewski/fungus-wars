package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const lobbyMemberIdleTimeout time.Duration = time.Duration(15) * time.Minute

// Lobby represents a group of players waiting to start a game.
// Once gameId is set, the lobby is closed.
type Lobby struct {
	player [maxPlayers]Player
	name   string
	gameId uuid.UUID
}

func (l *Lobby) String() string {
	var sb strings.Builder

	sb.WriteString("lobby {\n")
	if l.gameId != uuid.Nil {
		sb.WriteString(fmt.Sprintf("  name: %s\n", l.name))
		sb.WriteString(fmt.Sprintf("  gameId: %s\n", l.gameId.String()))
	}
	for i := 0; i < len(l.player); i++ {
		sb.WriteString("  ")
		sb.WriteString(l.player[i].String())
		sb.WriteString("\n")
	}
	sb.WriteString("}")

	return sb.String()
}

var activeLobbies = map[string]*Lobby{}
var lobbyMutex sync.Mutex

var lobbyAllowedNameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
var lobbyMemberReservedNameRegex = regexp.MustCompile(`(?i)^player\s*[1234]$`)

var reservedColors []*RGB = []*RGB{
	{[3]uint8{0x11, 0x11, 0x11}}, // common.css body background-color
}

// joinLobby adds player `pname` to lobby `lname`. Lobby is created if it doesn't exist.
// If `pcolor` is an empty string, a random color will be generated.
// Errors if lobby is full, or if the player or color name is in use by another member.
func joinLobby(lname string, pname string, pcolor string) (Player, error) {
	pcolorRGB := RGB{}
	avoidRGBPlayer := make([]*RGB, 0, maxPlayers)

	// put this somewhere below

	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()

	lobby, ok := activeLobbies[lname]
	var freeSlot int = -1
	if ok {
		if lobby.gameId != uuid.Nil {
			return Player{}, errors.New("Game has already started")
		}
		// scan Lobby for the next spot
		for i := 0; i < len(lobby.player); i++ {
			if lobby.player[i].lastSeen.IsZero() {
				freeSlot = i
				break
			}
		}
		if freeSlot < 0 {
			return Player{}, errors.New("Lobby is full")
		}
	} else {
		// create a lobby
		freeSlot = 0
		lobby = &Lobby{name: lname}
		activeLobbies[lname] = lobby
	}

	// set pname based on slot if empty
	if pname == "" {
		pname = fmt.Sprintf("Player %d", freeSlot+1)
	}

	if pcolor != "" {
		err := pcolorRGB.Parse(pcolor)
		if err != nil {
			return Player{}, errors.New("Color parse error: " + err.Error())
		}
		for _, r := range reservedColors {
			if r.IsNearDuplicate(&pcolorRGB, defaultRGBTolerance) {
				return Player{}, fmt.Errorf("Duplicate color: %v (%v is reserved)", &pcolorRGB, r)
			}
		}
	}

	// scan Lobby to ensure `pname` and `pcolor` are not in use.
	// build avoidRGBPlayer
	for i := 0; i < len(lobby.player); i++ {
		if lobby.player[i].lastSeen.IsZero() {
			continue
		}
		if lobby.player[i].Name == pname {
			return Player{}, errors.New("Duplicate player name: " + pname)
		}
		if pcolor != "" && lobby.player[i].Color.IsNearDuplicate(&pcolorRGB, defaultRGBTolerance) {
			return Player{}, fmt.Errorf("Duplicate color: %v (player %d has %v)",
				&pcolorRGB,
				i,
				&lobby.player[i].Color)
		}
		avoidRGBPlayer = append(avoidRGBPlayer, &lobby.player[i].Color)
	}
	if pcolor == "" {
		pcolorRGB.RandomizeAvoidingDuplicates(defaultRGBTolerance, avoidRGBPlayer, reservedColors)
	}

	// create user
	lobby.player[freeSlot] = newPlayer(pname)
	lobby.player[freeSlot].Color = pcolorRGB
	return lobby.player[freeSlot], nil
}

// leaveLobby removes player with uuid `id` from lobby `lname`
func leaveLobby(lname string, id uuid.UUID) error {
	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()

	lobby, ok := activeLobbies[lname]
	if !ok {
		return errors.New("leaveLobby: lobby not found: " + lname)
	}
	for i := 0; i < len(lobby.player); i++ {
		if id == lobby.player[i].id {
			lobby.player[i] = Player{}
			return nil
		}
	}
	return fmt.Errorf(`leaveLobby: player not found. Lobby: "%s". UUID: %s`, lname, id)
}

// cleanUpLobbies removes inactive players from all lobbies and removes empty lobbies
func cleanUpLobbies(serverlog *log.Logger, debug bool) {
	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()

	for k := range activeLobbies {
		player_count := 0
		for i, u := range activeLobbies[k].player {
			if u.lastSeen.IsZero() {
				continue
			}
			if time.Since(u.lastSeen) > lobbyMemberIdleTimeout {
				serverlog.Println("cleanUpLobbies(): Purging inactive player " + u.Name)
				activeLobbies[k].player[i] = Player{}
				continue
			}
			player_count++
		}
		if player_count == 0 {
			serverlog.Println("cleanUpLobbies(): Deleting empty lobby " + k)
			delete(activeLobbies, k)
		}
	}
}

func cleanUpLobbiesBackgroundTask(serverlog *log.Logger, debug bool) {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		cleanUpLobbies(serverlog, debug)
	}
}

type LobbyList struct {
	Lobbies []string `json:"lobbies"`
}
