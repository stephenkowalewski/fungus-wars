package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stephenkowalewski/fungus-wars/internal/server_flags"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

func TestParsePiecesArg(t *testing.T) {
	var arg string
	var res []Piece
	var err error

	if !testing.Verbose() {
		serverLogFile := server_flags.Logfile{Logger: &serverlog}
		serverLogFile.Set(os.DevNull)
	}

	// test empty lists
	arg = `{}`
	res, err = parsePiecesArg(url.QueryEscape(arg))
	if err != nil {
		t.Error("Unexpected error parsing empty list", err)
	}
	if len(res) > 0 {
		t.Error("Expected result to be an empty list. Got:", res)
	}

	arg = `{"data":[]}`
	res, err = parsePiecesArg(url.QueryEscape(arg))
	if err != nil {
		t.Error("Unexpected error parsing empty list", err)
	}
	if len(res) > 0 {
		t.Error("Expected result to be an empty list. Got:", res)
	}

	// test that invalid weights and piece masks are ignored
	arg = `{"data": [
		{"mask": 33325056, "weight": 100 },
		{"mask": 29622272, "weight": 0   },
		{"mask": 25559040, "weight": -12 },
		{"mask": 13369344, "weight": 12.5},
		{"mask": 0,        "weight": 1   },
		{"mask": 33554433, "weight": 1   },
		{"mask": 21254144, "weight": 1   },
		{"mask": 17039360, "weight": 5   }
	]}`
	expected := []Piece{
		{PieceMask(33325056).generateRotations(), 100},
		{PieceMask(13369344).generateRotations(), 12.5},
		{PieceMask(21254144).generateRotations(), 1},
		{PieceMask(17039360).generateRotations(), 5},
	}
	res, err = parsePiecesArg(url.QueryEscape(arg))
	if err != nil {
		t.Error("Unexpected error parsing empty list", err)
	}
	if !slices.Equal(res, expected) {
		t.Errorf("Unexpected result. Got %v. Expected: %v.", res, expected)
	}
}

func TestGameWsHandler_PlayerInfo(t *testing.T) {
	if !testing.Verbose() {
		serverLogFile := server_flags.Logfile{Logger: &serverlog}
		serverLogFile.Set(os.DevNull)
	}

	// create a game with 2 players
	players := [maxPlayers]Player{newPlayer("PlayerOne"), newPlayer("PlayerTwo"), Player{}, Player{}}
	game, err := createGame(&Lobby{player: players}, nil)
	if err != nil {
		t.Fatal("Unexpected error from createGame:", err)
	}

	// connect as each player
	for _, player := range players {
		if player.id == uuid.Nil {
			continue
		}

		// Setup HTTP server
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			gameWsHandler(w, r)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		// Prepare cookies to authenticate as player one
		cookies := []*http.Cookie{
			{Name: "player-id", Value: player.id.String()},
			{Name: "player-name", Value: player.Name},
			{Name: "game-id", Value: game.uuid.String()},
		}

		// set the cookies in the request
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		options := &websocket.DialOptions{
			HTTPHeader: http.Header{},
		}
		for _, c := range cookies {
			options.HTTPHeader.Add("Cookie", fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		wsURL := "ws" + srv.URL[len("http"):] + "/ws"
		c, _, err := websocket.Dial(ctx, wsURL, options)
		if err != nil {
			t.Fatalf("Failed to dial websocket: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// The first message should be a "player_info"
		var msg Message
		err = wsjson.Read(ctx, c, &msg)
		if err != nil {
			t.Fatalf("Failed to read from websocket: %v", err)
		}
		if msg.Type != "player_info" {
			t.Fatalf("Expected Type=player_info message but got: %s", msg.Type)
		}

		// sanity check "player_info"
		var payload MessagePayloadPlayerInfo
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatalf("Cannot unmarshal payload as MessagePayloadPlayerInfo: %v", err)
		}

		if len(payload.Players) != 2 {
			t.Fatalf("Expected 2 players, got %d: %+v", len(payload.Players), payload.Players)
		}
		foundPlayer := false
		for _, payloadPlayer := range payload.Players {
			if payloadPlayer.Name == player.Name {
				foundPlayer = true
				break
			}
		}
		if !foundPlayer {
			t.Errorf("Expected player %s in players: %+v", player.Name, payload.Players)
		}
	}
}
