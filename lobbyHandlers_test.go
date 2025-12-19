package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stephenkowalewski/fungus-wars/internal/server_flags"
)

func addPlayerCookies(t *testing.T, req *http.Request, player Player, lobbyName string) {
	cookies, err := http.ParseCookie(
		fmt.Sprintf(
			`player-id=%s; player-name=%s; lobby-name=%s`,
			url.QueryEscape(player.id.String()),
			url.QueryEscape(player.Name),
			url.QueryEscape(lobbyName)))
	if err != nil || len(cookies) != 3 {
		t.Fatal("Failed to generate test - ParseCookie failed or is mising cookies. Err:", err)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
}

func TestGetPlayerFromReq(t *testing.T) {
	var req *http.Request
	var lname string
	var p1, p2, p3 Player
	var err error
	var cookies []*http.Cookie

	if !testing.Verbose() {
		serverLogFile := server_flags.Logfile{Logger: &serverlog}
		serverLogFile.Set(os.DevNull)
	}

	// create lobby with 3 players and delete player 2 to create a gap
	lname = "TestGetPlayerFromReq"
	p1 = joinLobbyWrapper(t, lname, "p1", "")
	p2 = joinLobbyWrapper(t, lname, "p2", "")
	p3 = joinLobbyWrapper(t, lname, "p3", "")
	err = leaveLobby(lname, p2.id)
	if err != nil {
		t.Fatal("leaveLobby returned unexpected error:", err)
	}

	// test valid players
	for i, p := range []Player{p1, {}, p3} { // dummy p2 so we can use i as an index
		if p.lastSeen.IsZero() {
			continue
		}
		req = httptest.NewRequest("GET", "/valid", strings.NewReader(""))
		addPlayerCookies(t, req, p, lname)
		for _, updateLastSeen := range []bool{true, false} {
			lastSeenBefore := activeLobbies[lname].player[i].lastSeen
			time.Sleep(1 * time.Microsecond)
			lob, lobMem, err := getLobbyPlayerFromReq(req, updateLastSeen)
			if err != nil {
				t.Fatal("getLobbyPlayerFromReq failed:", err)
			}
			if lob != lname {
				t.Fatalf(`getLobbyPlayerFromReq returned unexpected lobby name "%s". Expected "%s".`, lob, lname)
			}
			if lobMem != activeLobbies[lname].player[i] {
				t.Fatalf("getLobbyPlayerFromReq return value does not match global state")
			}
			if updateLastSeen {
				if !lastSeenBefore.Before(activeLobbies[lname].player[i].lastSeen) {
					t.Fatalf("lastSeen was not updated for %s. %v is not after %v.", p.Name, activeLobbies[lname].player[i].lastSeen, p.lastSeen)
				}
			} else {
				if activeLobbies[lname].player[i].lastSeen != lastSeenBefore {
					t.Fatalf("lastSeen was updated for %s, but an update was not requested. %v != %v.", p.Name, activeLobbies[lname].player[i].lastSeen, lastSeenBefore)
				}
			}
		}
	}

	// test missing player
	for _, p := range []Player{p2} {
		req = httptest.NewRequest("GET", "/invalid", strings.NewReader(""))
		cookies, err = http.ParseCookie(fmt.Sprintf(`player-id=%s; player-name=%s; lobby-name=%s`, p.id.String(), p.Name, lname))
		if err != nil || len(cookies) != 3 {
			t.Fatal("Failed to generate test - ParseCookie failed or is mising cookies. Err:", err)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		for _, updateLastSeen := range []bool{true, false} {
			_, _, err := getLobbyPlayerFromReq(req, updateLastSeen)
			if err == nil || err.Error() != "Player not found" {
				t.Fatal("getLobbyPlayerFromReq did not respond with 'Player not found' for missing player:", p.Name)
			}
		}
	}

	// test incomplete request - missing player-id
	for _, p := range []Player{p1, p2, p3} {
		req = httptest.NewRequest("GET", "/foo", strings.NewReader(""))
		cookies, err = http.ParseCookie(fmt.Sprintf(`player-name=%s; lobby-name=%s`, p.Name, lname))
		if err != nil || len(cookies) != 2 {
			t.Fatal("Failed to generate test - ParseCookie failed or is mising cookies. Err:", err)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		for _, updateLastSeen := range []bool{true, false} {
			_, _, err = getLobbyPlayerFromReq(req, updateLastSeen)
			if err == nil || err.Error() != "Required cookie missing or empty: player-id" {
				t.Fatal("getLobbyPlayerFromReq succeeded despite a missing player-id Cookie")
			}
		}
	}

	// test incomplete request - missing player-name
	for _, p := range []Player{p1, p2, p3} {
		req = httptest.NewRequest("GET", "/bar", strings.NewReader(""))
		cookies, err = http.ParseCookie(fmt.Sprintf(`player-id=%s; lobby-name=%s`, p.id.String(), lname))
		if err != nil || len(cookies) != 2 {
			t.Fatal("Failed to generate test - ParseCookie failed or is mising cookies. Err:", err)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		for _, updateLastSeen := range []bool{true, false} {
			_, _, err = getLobbyPlayerFromReq(req, updateLastSeen)
			if err == nil || err.Error() != "Required cookie missing or empty: player-name" {
				t.Fatal("getLobbyPlayerFromReq succeeded despite a missing player-name Cookie")
			}
		}
	}

	// test missing lobby
	for _, p := range []Player{p1, p2, p3} {
		req = httptest.NewRequest("GET", "/baz", strings.NewReader(""))
		cookies, err = http.ParseCookie(fmt.Sprintf(`player-id=%s; player-name=%s; lobby-name="INVALID"`, p.id.String(), p.Name))
		if err != nil || len(cookies) != 3 {
			t.Fatal("Failed to generate test - ParseCookie failed or is mising cookies. Err:", err)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		for _, updateLastSeen := range []bool{true, false} {
			_, _, err = getLobbyPlayerFromReq(req, updateLastSeen)
			if err == nil || err.Error() != "Lobby does not exist" {
				t.Fatal("getLobbyPlayerFromReq succeeded despite invalid lobby-name")
			}
		}
	}

	// create lobby with special characters in player names
	lname = "TestGetPlayerFromReqSpecialChars"
	p1 = joinLobbyWrapper(t, lname, "José", "")
	p2 = joinLobbyWrapper(t, lname, "Name with spaces", "")

	// test valid players
	for i, p := range []Player{p1, p2} {
		if p.lastSeen.IsZero() {
			continue
		}
		req = httptest.NewRequest("GET", "/valid", strings.NewReader(""))
		addPlayerCookies(t, req, p, lname)
		for _, updateLastSeen := range []bool{true, false} {
			lastSeenBefore := activeLobbies[lname].player[i].lastSeen
			time.Sleep(1 * time.Microsecond)
			lob, lobMem, err := getLobbyPlayerFromReq(req, updateLastSeen)
			if err != nil {
				t.Fatal("getLobbyPlayerFromReq failed:", err)
			}
			if lob != lname {
				t.Fatalf(`getLobbyPlayerFromReq returned unexpected lobby name "%s". Expected "%s".`, lob, lname)
			}
			if lobMem != activeLobbies[lname].player[i] {
				t.Fatalf("getLobbyPlayerFromReq return value does not match global state")
			}
			if updateLastSeen {
				if !lastSeenBefore.Before(activeLobbies[lname].player[i].lastSeen) {
					t.Fatalf("lastSeen was not updated for %s. %v is not after %v.", p.Name, activeLobbies[lname].player[i].lastSeen, p.lastSeen)
				}
			} else {
				if activeLobbies[lname].player[i].lastSeen != lastSeenBefore {
					t.Fatalf("lastSeen was updated for %s, but an update was not requested. %v != %v.", p.Name, activeLobbies[lname].player[i].lastSeen, lastSeenBefore)
				}
			}
		}
	}

}

func TestLobbyInfoHandler(t *testing.T) {
	var lname string = "TestLobbyInfoHandler"
	var err error

	if !testing.Verbose() {
		serverLogFile := server_flags.Logfile{Logger: &serverlog}
		serverLogFile.Set(os.DevNull)
	}

	// create lobby with 3 players and delete player 2 to create a gap
	p1 := joinLobbyWrapper(t, lname, "p1", "#ff0000")
	p2 := joinLobbyWrapper(t, lname, "p2", "#00ff00")
	_ = joinLobbyWrapper(t, lname, "p3", "#0000ff")
	err = leaveLobby(lname, p2.id)
	if err != nil {
		t.Fatal("leaveLobby returned unexpected error:", err)
	}

	// generate a test request
	req := httptest.NewRequest("GET", "/lobby/get", strings.NewReader(""))
	addPlayerCookies(t, req, p1, lname)
	w := httptest.NewRecorder()

	// call lobbyInfoHandler and get the result
	lobbyInfoHandler(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	t.Log("lobbyInfoHandler response body was:", string(body))

	// verify the response body
	expected := `{"members":[{"name":"p1","color":"#ff0000"},{"name":"p3","color":"#0000ff"}],"game":null}`
	var expectedMap, resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedMap); err != nil {
		t.Fatalf("Failed to unmarshal expected: %v", err)
	}
	if err := json.Unmarshal(body, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if !reflect.DeepEqual(expectedMap, resultMap) {
		t.Errorf("JSON does not match.\nExpected: %v\nActual:   %v", expectedMap, resultMap)
	}
}
