package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/stephenkowalewski/fungus-wars/internal/logging"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

// Message is used for data shared with the client
// Valid types are: ping, pong, player_info
// Payload is either a MessagePayload struct defined below or null
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// MessagePayloadPlayerInfo is the payload for messages where
// type == "player_info"
type MessagePayloadPlayerInfo struct {
	Players []Player `json:"players"`
	// Identity tells the requestor which Player they are (index into Players)
	Identity          int           `json:"identity"`
	WinLossDrawRecord []WinLossDraw `json:"win_loss_draw_record"`
}

// MessagePayloadGameAction is the payload for messages where
// type == "game_update"
type MessagePayloadGameAction struct {
	Action string `json:"action"`
}

// MessagePayloadButtonAction is the payload for messages where
// type == "button_update" or
// type == "button_info"
type MessagePayloadButtonAction struct {
	Active   []string `json:"active"`
	Inactive []string `json:"inactive"`
	Notify   []string `json:"notify"`
}

// MessagePayloadGameInfo is the payload for messages where
// type == "game_info"
type MessagePayloadGameInfo struct {
	Board           GameBoard `json:"board"`
	LastBoardUpdate []int     `json:"board_updates_to_animate"`
	Turn            int       `json:"turn"`
	NextPiece       Piece     `json:"next_piece"`
	Scores          []int     `json:"scores"`
	Bites           []int     `json:"bites"`
	Rerolls         []int     `json:"rerolls"`
	GameOver        bool      `json:"game_over"`
}

// MessagePayloadBoardUpdate is the payload for messages where
// type == "board_update"
type MessagePayloadBoardUpdate struct {
	Action string    `json:"action"`
	Index  int       `json:"index"`
	Mask   PieceMask `json:"mask"`
}

// MessagePayloadBoardUpdatePreview is the payload for messages where
// type == "board_update_preview" or
// type == "board_info_preview"
type MessagePayloadBoardUpdatePreview struct {
	Action string    `json:"action"`
	Owner  Cell      `json:"owner"`
	Index  int       `json:"index"`
	Mask   PieceMask `json:"mask"`
}

// MessagePayloadError is the payload for messages where
// type == "error"
type MessagePayloadError struct {
	Message string `json:"message"`
}

// gameArgCustomPieces is the payload for custom game pieces
type gameArgCustomPieces struct {
	Data []gameArgCustomPiece `json:"data"`
}
type gameArgCustomPiece struct {
	Mask   PieceMask `json:"mask"`
	Weight float64   `json:"weight"`
}

// getGamePlayerFromReq gets the player based on cookies and optionally updates the
// lastSeen field for that player.
// return values: game UUID, Player, error
func getGamePlayerFromReq(r *http.Request, updateLastSeen bool) (uuid.UUID, Player, error) {
	cookiePlayerId, err := getCookieWrapper(r, "player-id")
	if err != nil || cookiePlayerId == "" {
		return uuid.Nil, Player{}, fmt.Errorf("Required cookie missing or empty: player-id")
	}
	cookiePlayerName, err := getCookieWrapper(r, "player-name")
	if err != nil || cookiePlayerName == "" {
		return uuid.Nil, Player{}, fmt.Errorf("Required cookie missing or empty: player-name")
	}
	cookieGameId, err := getCookieWrapper(r, "game-id")
	if err != nil || cookieGameId == "" {
		return uuid.Nil, Player{}, fmt.Errorf("Required cookie missing or empty: game-id")
	}

	playerId, err := uuid.Parse(cookiePlayerId)
	if err != nil {
		return uuid.Nil, Player{}, err
	}
	gameId, err := uuid.Parse(cookieGameId)
	if err != nil {
		return uuid.Nil, Player{}, err
	}

	activeGameMutex.Lock()
	game, ok := activeGames[gameId]
	activeGameMutex.Unlock()
	if !ok {
		return uuid.Nil, Player{}, errors.New("Game not found: " + cookieGameId)
	}
	game.mu.Lock()
	defer game.mu.Unlock()

	for i := 0; i < game.playerCount; i++ {
		if game.players[i].id == playerId {
			if game.players[i].Name == cookiePlayerName {
				if updateLastSeen {
					game.players[i].lastSeen = time.Now()
				}
				return gameId, game.players[i], nil
			}
		}
	}

	return uuid.Nil, Player{}, errors.New("Player not found")
}

// handle converting the pieces url arg to the format expected by createGame
func parsePiecesArg(arg string) ([]Piece, error) {
	var unmarshalled gameArgCustomPieces
	var pieces []Piece

	jstring, err := url.QueryUnescape(arg)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jstring), &unmarshalled)
	if err != nil {
		return nil, err
	}

	for _, p := range unmarshalled.Data {
		if p.Weight <= 0 {
			if debug {
				serverlog.Printf("Skipping custom piece with zero or negative weight: %v", p)
			}
			continue
		}
		if p.Mask == 0 || p.Mask&^pieceMaskFullMask != 0 {
			if debug {
				serverlog.Printf("Skipping custom piece with invalid mask: peice: %v mask: %b", p, p.Mask)
			}
			continue
		}
		pieces = append(pieces, Piece{p.Mask.generateRotations(), p.Weight})
	}
	return pieces, nil
}

// createGameHandler creates a new game for the players in the lobby with the requestor.
// It redirects the player to the join game endpoint.
func createGameHandler(w http.ResponseWriter, r *http.Request) {
	createGameOpts := map[string]any{}

	// get lobby
	lobbyName, _, err := getLobbyPlayerFromReq(r, true)
	if err != nil {
		if debug {
			serverlog.Println("getLobbyPlayerFromReq had error: " + err.Error())
		}
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// lock activeLobbies
	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()
	lobby, ok := activeLobbies[lobbyName]
	if !ok {
		if debug {
			serverlog.Println("lobby not found: " + lobbyName)
		}
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// add url args to createGame options
	for _, boolArg := range []string{
		"randomize_start_positions",
		"has_bonus_bite_cells",
	} {
		if s := r.URL.Query().Get(boolArg); s != "" {
			if parsed, err := strconv.ParseBool(s); err == nil {
				createGameOpts[boolArg] = parsed
			} else {
				serverlog.Printf("Failed to parse boolean URL arg %s with value %s.\n", boolArg, s)
			}
		}
	}
	for _, intArg := range []string{
		"size",
		"starting_bites",
		"starting_rerolls",
		"bonus_reroll_cells",
		"capture_mode",
	} {
		if s := r.URL.Query().Get(intArg); s != "" {
			if parsed, err := strconv.Atoi(s); err == nil {
				createGameOpts[intArg] = parsed
			} else {
				serverlog.Printf("Failed to parse integer URL arg %s with value %s.\n", intArg, s)
			}
		}
	}
	for _, floatArg := range []string{"new_bites_freq_factor"} {
		if s := r.URL.Query().Get(floatArg); s != "" {
			if parsed, err := strconv.ParseFloat(s, 64); err == nil {
				createGameOpts[floatArg] = parsed
			} else {
				serverlog.Printf("Failed to parse floating point URL arg %s with value %s.\n", floatArg, s)
			}
		}
	}
	// custom game pieces need to be unmarshalled and converted to []Piece
	if j := r.URL.Query().Get("pieces"); j != "" {
		pieces, err := parsePiecesArg(j)
		if err == nil {
			createGameOpts["pieces"] = pieces
		} else {
			serverlog.Printf("Failed to parse pieces URL arg with value %s: %v\n", j, err)
		}
	}

	game, err := createGame(lobby, createGameOpts)
	if err != nil {
		if err.Error() == "A game requires at least two players" {
			http.Redirect(w, r, "/static/error_pages/lobby.html?err=not_enough_players", http.StatusFound)
			return
		}
		if debug {
			serverlog.Printf("createGame(%s) had error: %s\n", lobbyName, err.Error())
		}
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return

	}
	lobby.gameId = game.uuid
	serverlog.Println("Created game with UUID:", game.uuid)
	if debug {
		serverlog.Println(game.String())
	}

	http.Redirect(w, r, "/game/join", http.StatusFound)
}

// joinGameHandler sets a cookie and redirects to the main game page
func joinGameHandler(w http.ResponseWriter, r *http.Request) {
	// get the gameId from the player's lobby
	lobbyName, _, err := getLobbyPlayerFromReq(r, true)
	if err != nil {
		// Player is not in a lobby, redirect to error page
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=not_in_lobby", http.StatusFound)
		return
	}

	lobbyMutex.Lock()
	lobby, ok := activeLobbies[lobbyName]
	lobbyMutex.Unlock()

	if !ok {
		// This shouldn't happen if getLobbyPlayerFromReq succeeded
		serverlog.Printf("joinGameHandler: could not find lobby '%s'", lobbyName)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	if lobby.gameId == uuid.Nil {
		// Game has not started yet, send back to lobby
		http.Redirect(w, r, "/lobby", http.StatusFound)
		return
	}

	// set game-id cookie, clear lobby cookie
	w.Header().Add("Set-Cookie", fmt.Sprintf(`game-id=%s; path=/`, lobby.gameId.String()))
	w.Header().Add("Set-Cookie", `lobby-name=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`)

	// redirect to the game page
	http.Redirect(w, r, "/game", http.StatusFound)
}

// leaveGameHandler clears cookies and redirects to the main game page
func leaveGameHandler(w http.ResponseWriter, r *http.Request) {
	clearCookies(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// gameWsHandler sets up a WebSocket and dispatches commands
func gameWsHandler(w http.ResponseWriter, r *http.Request) {
	gameId, player, err := getGamePlayerFromReq(r, true)
	if err != nil {
		http.Error(w, "403 forbidden", http.StatusForbidden)
		return
	}

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		serverlog.Println(err)
		return
	}
	defer c.CloseNow()

	// add connection to game wsConns
	identity := -1
	activeGameMutex.Lock()
	thisGame := activeGames[gameId]
	activeGameMutex.Unlock()
	thisGame.mu.Lock()
	for i := 0; i < thisGame.playerCount; i++ {
		if thisGame.players[i].id == player.id {
			identity = i
			break
		}
	}
	if identity >= 0 {
		if thisGame.wsConns[identity] != nil {
			_ = thisGame.wsConns[identity].Close(websocket.StatusGoingAway, "connection replaced")
		}
		thisGame.wsConns[identity] = c
	}
	thisGame.mu.Unlock()
	if identity < 0 {
		serverlog.Printf("Player %v is not in game %v", player.id, gameId)
		c.Close(websocket.StatusInternalError, "send error")
		return
	}

	// Ping setup for WebSocket keep-alive
	cancel := gameWsStartPingPong(c)
	defer cancel()

	// Send initial messages on connection
	err = gameWsSendPlayerInfo(c, thisGame, player, false)
	if err != nil {
		serverlog.Printf("Failed to send player_info to client: %v\n", err)
		c.Close(websocket.StatusInternalError, "send error")
		return
	}
	err = gameWsSendGameInfo(c, thisGame, false)
	if err != nil {
		serverlog.Printf("Failed to send game_info to client: %v\n", err)
		c.Close(websocket.StatusInternalError, "send error")
		return
	}

	// Wait for client messages
	for {
		msg, err := gameWsReadMessage(c)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			serverlog.Printf("gameWsReadMessage failed for %v (game: %v, player: %v): %v", r.RemoteAddr, gameId, player.id, err)
			c.Close(websocket.StatusInternalError, "read error")
			return
		}

		if msg.Type == "pong" {
			// Do nothing. If the client stops sending these, gameWsReadMessage will timeout and fail
			continue
		}

		switch msg.Type {
		case "board_update_preview":
			if debug {
				logging.LogWebSocket(accesslog, r, msg.Type)
			}
			gameWsHandleBoardUpdatePreview(c, thisGame, player, msg)
		case "board_update":
			logging.LogWebSocket(accesslog, r, msg.Type)
			gameWsHandleBoardUpdate(c, thisGame, player, msg)
		case "button_update":
			logging.LogWebSocket(accesslog, r, msg.Type)
			gameWsHandleButtonAction(c, thisGame, player, msg)
		case "game_update":
			logging.LogWebSocket(accesslog, r, msg.Type)
			gameWsHandleGameAction(c, thisGame, player, msg)
		default:
			serverlog.Println("unimplemented:", msg.Type)
		}
	}
}

// gameWsStartPingPong sends "ping" messages to the client. The client should respond with "pong"
// messages to keep the WebSocket connection alive.
func gameWsStartPingPong(conn *websocket.Conn) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pingMsg := Message{
					Type: "ping",
				}
				pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
				err := wsjson.Write(pingCtx, conn, pingMsg)
				pingCancel()
				if err != nil {
					serverlog.Println("Ping write error:", err)
					return // stop goroutine on error
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return cancel
}

// gameWsSendPlayerInfo sends the client a message of type "player_info" for game UUID
func gameWsSendPlayerInfo(conn *websocket.Conn, game *Game, whoami Player, hasLock bool) error {
	identity := -1

	if !hasLock {
		game.mu.Lock()
	}
	players := make([]Player, game.playerCount)
	for i := 0; i < game.playerCount; i++ {
		players[i] = game.players[i]
		if players[i].id == whoami.id {
			identity = i
		}
	}
	if !hasLock {
		game.mu.Unlock()
	}

	if identity < 0 {
		return fmt.Errorf("Player not found in game. Player=%v, game=%s", whoami.id, game.shortDesc())
	}

	payload := MessagePayloadPlayerInfo{
		Players:           players,
		Identity:          identity,
		WinLossDrawRecord: game.winLossDrawRecord[:game.playerCount],
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := Message{
		Type:    "player_info",
		Payload: payloadBytes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return wsjson.Write(ctx, conn, msg)
}

// send "player_info" update to all connected players for game
func gameWsBroadcastPlayerInfo(game *Game) {
	var wg sync.WaitGroup
	game.mu.Lock()
	defer game.mu.Unlock()
	for i := 0; i < game.playerCount; i++ {
		if game.wsConns[i] != nil {
			wg.Add(1)
			go func(connIndex int) {
				gameWsSendPlayerInfo(game.wsConns[connIndex], game, game.players[i], true)
				defer wg.Done()
			}(i)
		}
	}
	wg.Wait() // wait for go routines to complete before releasing game.mu lock
}

// gameWsSendGameInfo sends the client a message of type "game_info" for game
func gameWsSendGameInfo(conn *websocket.Conn, game *Game, hasLock bool) error {
	if !hasLock {
		game.mu.Lock()
	}
	payload := MessagePayloadGameInfo{
		Board:           game.board,
		LastBoardUpdate: game.lastBoardUpdate,
		Turn:            game.turn,
		NextPiece:       game.nextPiece,
		Scores:          game.scores[:game.playerCount],
		Bites:           game.bites[:game.playerCount],
		Rerolls:         game.rerolls[:game.playerCount],
		GameOver:        game.isOver,
	}
	if !hasLock {
		game.mu.Unlock()
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := Message{
		Type:    "game_info",
		Payload: payloadBytes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return wsjson.Write(ctx, conn, msg)
}

// send "game_info" update to all connected players for game
func gameWsBroadcastGameInfo(game *Game) {
	var wg sync.WaitGroup
	game.mu.Lock()
	defer game.mu.Unlock()
	for i := 0; i < game.playerCount; i++ {
		if game.wsConns[i] != nil {
			wg.Add(1)
			go func(connIndex int) {
				gameWsSendGameInfo(game.wsConns[connIndex], game, true)
				defer wg.Done()
			}(i)
		}
	}
	wg.Wait() // wait for go routines to complete before releasing game.mu lock
}

// gameWsSendError sends the client a message of type "error"
func gameWsSendError(conn *websocket.Conn, message string) error {
	payload := MessagePayloadError{
		Message: message,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		emsg := "Unexpected json.Marshal error in gameWsSendError(): " + err.Error()
		serverlog.Println(emsg)
		return errors.New(emsg)
	}
	msg := Message{
		Type:    "error",
		Payload: payloadBytes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return wsjson.Write(ctx, conn, msg)
}

// gameWsReadMessage reads a Message from the client. Unpacks Message.Type but not Message.Payload.
func gameWsReadMessage(conn *websocket.Conn) (*Message, error) {
	readTimeout := 10 * time.Second
	if debug {
		readTimeout *= 10
	}
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()

	var msg Message
	err := wsjson.Read(ctx, conn, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// gameWsHandleBoardUpdate processes a Player move (type: "board_update").
// If it's a legal move, the update is sent to all players. Otherwise, an error
// (type "error")
func gameWsHandleBoardUpdate(conn *websocket.Conn, game *Game, whoami Player, msg *Message) {

	handleError := func(serverMessage, clientMessage string) {
		serverlog.Println(serverMessage + ": " + clientMessage)
		_ = gameWsSendError(conn, clientMessage)
		err := gameWsSendGameInfo(conn, game, false) // reset the board
		if err != nil {
			serverlog.Printf("Failed to send error message to client: %v\n", err)
			conn.Close(websocket.StatusInternalError, "send error")
		}
	}

	var payload MessagePayloadBoardUpdate
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		handleError(
			fmt.Sprintf("Invalid payload for msg type %s: %v", msg.Type, string(msg.Payload)),
			"Invalid game update",
		)
		return
	}

	_, owner := game.getTurnInfo(whoami)
	playerIndex := int(owner) - 1
	preCapturePreview := MessagePayloadBoardUpdatePreview{
		Action: payload.Action,
		Owner:  owner,
		Index:  payload.Index,
		Mask:   payload.Mask,
	}

	if debug {
		serverlog.Printf("***%v gameboard before move*** Index: %d, Action: %s\nMask:\n%s\nBoard:\n%s",
			game.uuid,
			payload.Index,
			payload.Action,
			payload.Mask.String2D(),
			game.board.String2D(),
		)
	}

	switch payload.Action {
	case "place_piece":
		err = game.placePiece(whoami, payload.Index, payload.Mask)
		if err != nil {
			handleError(
				fmt.Sprintf("placePiece failed. Player=%v %v", whoami.id, game.shortDesc()),
				err.Error(),
			)
			return
		}
		gameWsBroadcastBoardUpdatePreview(game, &preCapturePreview, []int{playerIndex})
	case "place_bite":
		err = game.placeBite(whoami, payload.Index, payload.Mask)
		if err != nil {
			handleError(
				fmt.Sprintf("placeBite failed. Player=%v %v", whoami.id, game.shortDesc()),
				err.Error(),
			)
			return
		}
		gameWsBroadcastBoardUpdatePreview(game, &preCapturePreview, []int{playerIndex})
	default:
		serverlog.Println("Skipping unexpected board_update action: " + payload.Action)
		return
	}

	if debug {
		serverlog.Printf("***%v gameboard after move*** Index: %d, Action: %s\nMask:\n%s\nBoard:\n%s",
			game.uuid,
			payload.Index,
			payload.Action,
			payload.Mask.String2D(),
			game.board.String2D(),
		)
	}

	// send updates to all connected players of game
	if game.isOver {
		gameWsBroadcastPlayerInfo(game)
	}
	gameWsBroadcastGameInfo(game)
	game.clearLastBoardUpdate()
}

// gameWsSendBoardUpdatePreview sends the client a message of type "board_update_preview"
func gameWsSendBoardUpdatePreview(conn *websocket.Conn, update *MessagePayloadBoardUpdatePreview) error {
	payloadBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	msg := Message{
		Type:    "board_info_preview",
		Payload: payloadBytes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return wsjson.Write(ctx, conn, msg)
}

// send "board_update_preview" update to all connected players for game, except for players
// in skip
func gameWsBroadcastBoardUpdatePreview(game *Game, update *MessagePayloadBoardUpdatePreview, skip []int) {
	var wg sync.WaitGroup
	game.mu.Lock()
	defer game.mu.Unlock()
	for i := 0; i < game.playerCount; i++ {
		doSkip := false
		for _, s := range skip {
			if i == s {
				doSkip = true
				break
			}
		}
		if doSkip {
			continue
		}
		if game.wsConns[i] != nil {
			wg.Add(1)
			go func(connIndex int) {
				gameWsSendBoardUpdatePreview(game.wsConns[i], update)
				defer wg.Done()
			}(i)
		}
	}
	wg.Wait() // wait for go routines to complete before releasing game.mu lock
}

// gameWsHandleBoardUpdatePreview sends a preview of a player's move to all players, except the
// one sending the update
func gameWsHandleBoardUpdatePreview(conn *websocket.Conn, game *Game, whoami Player, msg *Message) {
	var payload MessagePayloadBoardUpdatePreview
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		serverlog.Printf("Invalid payload for msg type %s: %v", msg.Type, msg.Payload)
		return
	}

	// send board_update_preview to all other connected players of game
	gameWsBroadcastBoardUpdatePreview(game, &payload, []int{game.turn})
}

func gameWsHandleGameAction(conn *websocket.Conn, game *Game, whoami Player, msg *Message) {

	handleError := func(serverMessage, clientMessage string) {
		serverlog.Println(serverMessage + ": " + clientMessage)
		_ = gameWsSendError(conn, clientMessage)
		err := gameWsSendGameInfo(conn, game, false) // reset the board
		if err != nil {
			serverlog.Printf("Failed to send error message to client: %v\n", err)
			conn.Close(websocket.StatusInternalError, "send error")
		}
	}

	var payload MessagePayloadGameAction
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		serverlog.Printf("Invalid payload for msg type %s: %v", msg.Type, string(msg.Payload))
		return
	}

	switch payload.Action {
	case "skip_turn":
		err = game.skipTurn(whoami)
		if err != nil {
			handleError(
				fmt.Sprintf("skipTurn failed. Player=%v %v", whoami.id, game.shortDesc()),
				err.Error(),
			)
			return
		}
	case "reroll":
		err = game.reroll(whoami)
		if err != nil {
			handleError(
				fmt.Sprintf("reroll failed. Player=%v %v", whoami.id, game.shortDesc()),
				err.Error(),
			)
			return
		}
	case "forfeit_game":
		game.forfeitGame(whoami)
	case "reset_game":
		game.resetGame()
		gameWsBroadcastPlayerInfo(game)
	default:
		serverlog.Println("gameWsHandleGameAction: Skipping unexpected action: " + payload.Action)
		return
	}

	// send game_info to all connected players of game
	gameWsBroadcastGameInfo(game)
	game.clearLastBoardUpdate()
}

// gameWsSendButtonInfo sends the client a message of type "button_update"
func gameWsSendButtonInfo(conn *websocket.Conn, update *MessagePayloadButtonAction) error {
	payloadBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	msg := Message{
		Type:    "button_info",
		Payload: payloadBytes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return wsjson.Write(ctx, conn, msg)
}

// send "button_update" update to all connected players for game, except for the player
// whose turn it is
func gameWsBroadcastButtonInfo(game *Game, update *MessagePayloadButtonAction) {
	var wg sync.WaitGroup
	game.mu.Lock()
	defer game.mu.Unlock()
	for i := 0; i < game.playerCount; i++ {
		if i == game.turn {
			continue
		}
		if game.wsConns[i] != nil {
			wg.Add(1)
			go func(connIndex int) {
				gameWsSendButtonInfo(game.wsConns[i], update)
				defer wg.Done()
			}(i)
		}
	}
	wg.Wait() // wait for go routines to complete before releasing game.mu lock
}

func gameWsHandleButtonAction(conn *websocket.Conn, game *Game, whoami Player, msg *Message) {
	var payload MessagePayloadButtonAction
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		serverlog.Printf("Invalid payload for msg type %s: %v", msg.Type, string(msg.Payload))
		return
	}

	gameWsBroadcastButtonInfo(game, &payload)
}

// vim:nowrap
