package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stephenkowalewski/fungus-wars/internal/server_flags"

	"github.com/google/uuid"
)

type TestOrBench interface {
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
	Fatal(args ...any)
	Error(args ...any)
	Log(args ...any)
}

// joinLobbyWrapper reduces boilerplate code in test cases where joinLobby is expected to succeed
func joinLobbyWrapper(t TestOrBench, lname string, pname string, pcolor string) Player {
	before := time.Now()
	lmem, err := joinLobby(lname, pname, pcolor)
	after := time.Now()
	if err != nil {
		t.Fatalf("joinLobby(%v, %v, '') failed: %s", lname, pname, err)
	}

	if pname == "" {
		// ensure we got back a reserved name
		if !lobbyMemberReservedNameRegex.MatchString(lmem.Name) {
			t.Fatalf("joinLobby was called with an empty player name, but the name we got back (%s) doesn't match lobbyMemberReservedNameRegex", lmem.Name)
		}
	} else {
		// ensure we got back our requested name
		if lmem.Name != pname {
			t.Fatalf("joinLobby set unexpected name for player. Expected %s. Got %s.", pname, lmem.Name)
		}
	}

	if lmem.id == uuid.Nil {
		t.Fatalf("joinLobby returned a user with a blank UUID")
	}

	if lmem.lastSeen.Before(before) || lmem.lastSeen.After(after) {
		t.Fatalf("lastSeen timestamp is incorrect")
	}

	return lmem
}

// checkLobbySize ensures that `lobby` has exactly `n` active players.
func checkLobbySize(t *testing.T, lobby *Lobby, n int) {
	count := 0
	for i := 0; i < len(lobby.player); i++ {
		if time.Since(lobby.player[i].lastSeen) <= lobbyMemberIdleTimeout {
			count++
		}
	}
	if count != n {
		t.Fatalf("Lobby has size %d. Expected %d.", count, n)
	}
}

// TestLobbyOperations tests adding and removing players from lobbies
func TestLobbyOperations(t *testing.T) {
	var err error

	if !testing.Verbose() {
		serverLogFile := server_flags.Logfile{Logger: &serverlog}
		serverLogFile.Set(os.DevNull)
	}

	// create full lobby with default names
	for i := 0; i < maxPlayers; i++ {
		lmem := joinLobbyWrapper(t, "test_lobby", "", "")
		expectedName := fmt.Sprintf("Player %d", i+1)
		if lmem.Name != expectedName {
			t.Fatalf(`unexpected player name after adding player %d to "test_lobby". Expected %s. Got %s.`, i, expectedName, lmem.Name)
		}
	}
	// try to add a player to a full lobby
	_, err = joinLobby("test_lobby", "", "")
	if err == nil {
		t.Fatalf("Adding a player to a full lobby succeeded")
	}
	// adding to a different lobby should still work
	_ = joinLobbyWrapper(t, "test_lobby2", "", "")

	// verify that activeLobbies contains test_lobby and test_lobby2, but not test_lobby3
	lobby, ok := activeLobbies["test_lobby"]
	if !ok {
		t.Fatal(`"test_lobby" is not in activeLobbies`)
	}
	checkLobbySize(t, lobby, maxPlayers)
	lobby, ok = activeLobbies["test_lobby2"]
	if !ok {
		t.Fatal(`"test_lobby2" is not in activeLobbies`)
	}
	checkLobbySize(t, lobby, 1)
	_, ok = activeLobbies["test_lobby3"]
	if ok {
		t.Fatal(`"test_lobby3" is in activeLobbies. Expected it to be missing.`)
	}

	// expire "Player 1" add "Player One"
	activeLobbies["test_lobby"].player[0].lastSeen = time.Now().Add(-1 * (lobbyMemberIdleTimeout + time.Second))
	cleanUpLobbies(serverlog, true)
	_ = joinLobbyWrapper(t, "test_lobby", "Player One", "")
	// expire "Player 3" and add "Player 2" fails with Duplicate player name
	activeLobbies["test_lobby"].player[2].lastSeen = time.Now().Add(-1 * (lobbyMemberIdleTimeout + time.Second))
	cleanUpLobbies(serverlog, true)
	t.Log("after adding Player One: " + activeLobbies["test_lobby"].String())
	_, err = joinLobby("test_lobby", "Player 2", "")
	if err == nil || !strings.HasPrefix(err.Error(), "Duplicate player name:") {
		t.Fatalf("Adding duplicate Player 2 did not fail as expected")
	}
	t.Log("after failing to add Player 2: " + activeLobbies["test_lobby"].String())
	// add "Player 4" fails with Duplicate player name
	// the lobby has space in the formerly "Player 3" slot since the previous operation failed
	_, err = joinLobby("test_lobby", "Player 4", "")
	if err == nil || !strings.HasPrefix(err.Error(), "Duplicate player name:") {
		t.Fatalf("Adding duplicate Player 4 did not fail as expected")
	}

	// leaveLobby tests
	t.Log("before leaveLobby: " + activeLobbies["test_lobby"].String())
	// bad lobby
	err = leaveLobby("test_lobby3", activeLobbies["test_lobby"].player[0].id)
	if err == nil || !strings.HasPrefix(err.Error(), "leaveLobby: lobby not found:") {
		t.Fatal("leaveLobby - Expected lobby not found error. Got", err)
	}
	// uuid from a different lobby
	err = leaveLobby("test_lobby", activeLobbies["test_lobby2"].player[0].id)
	if err == nil || !strings.HasPrefix(err.Error(), "leaveLobby: player not found.") {
		t.Fatal("leaveLobby - expected player not found error for bad UUID. Got", err)
	}
	// success
	err = leaveLobby("test_lobby", activeLobbies["test_lobby"].player[0].id)
	if err != nil {
		t.Fatal("leaveLobby returned unexpected error: ", err)
	}
	t.Log("after leaveLobby: " + activeLobbies["test_lobby"].String())

	// Verify joinLobby() pcolor arg and color-related errors
	lmem := joinLobbyWrapper(t, "color_lobby", "", "#abcdef")
	expectedRGB := RGB{[3]uint8{0xab, 0xcd, 0xef}}
	if !reflect.DeepEqual(lmem.Color, expectedRGB) {
		t.Fatalf("Player 1 did not get the requested color: %v. Got %v.", expectedRGB, lmem.Color)
	}
	// add another player color allowed by defaultRGBTolerance
	lmem = joinLobbyWrapper(t, "color_lobby", "", "#abc")
	expectedRGB = RGB{[3]uint8{0xaa, 0xbb, 0xcc}}
	if !reflect.DeepEqual(lmem.Color, expectedRGB) {
		t.Fatalf("Player 2 did not get the requested color: %v. Got %v.", expectedRGB, lmem.Color)
	}
	// check for Duplicate player color error
	_, err = joinLobby("color_lobby", "", "#acbcc0") // with defaultRGBTolerance of PLayer 2
	if err == nil || !strings.HasPrefix(err.Error(), "Duplicate color:") {
		t.Fatal("joinLobby - expected duplicate color error. Got", err)
	}
	// add a reserved color
	_, err = joinLobby("color_lobby", "", reservedColors[0].String())
	if err == nil || !strings.HasPrefix(err.Error(), "Duplicate color:") {
		t.Fatal("joinLobby - expected duplicate color error. Got", err)
	}
	// check for Color parse error
	_, err = joinLobby("color_lobby", "", "#abcd")
	if err == nil || !strings.HasPrefix(err.Error(), "Color parse error:") {
		t.Fatal("joinLobby - expected color parse error. Got", err)
	}

	// check for Game has already started error
	gameStartedLobby := "game_started_lobby"
	_ = joinLobbyWrapper(t, gameStartedLobby, "p1", "")
	gameUUID := uuid.New()
	activeLobbies[gameStartedLobby].gameId = gameUUID
	_, err = joinLobby(gameStartedLobby, "p2", "")
	if err == nil || err.Error() != "Game has already started" {
		t.Fatal("joinLobby - expected 'Game has already started' error. Got", err)
	}
}
