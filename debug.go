package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/stephenkowalewski/fungus-wars/internal/logging"

	"github.com/google/uuid"
)

func addDebugEndpoints() {
	http.Handle("/debug",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(debugHandlerListGames))))
	http.Handle("/debug/join-existing-game",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(debugHandlerJoinExistingGame))))
}

// debugHandlerListGames lists out all active games and provides buttons to join
// as an existing player.
func debugHandlerListGames(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<!DOCTYPE html><head></head><body>`)

	activeGameMutex.Lock()
	defer activeGameMutex.Unlock()

	if len(activeGames) == 0 {
		fmt.Fprint(w, `<p>No active games</p></body>`)
		return
	}

	keys := make([]uuid.UUID, 0, len(activeGames))
	for k := range activeGames {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return uuidLess(keys[i], keys[j])
	})

	for _, gameId := range keys {
		fmt.Fprintf(w, `<p>Game: %v (%s)</p>`, gameId, activeGames[gameId].fromLobby)
		for i := 0; i < activeGames[gameId].playerCount; i++ {
			player := activeGames[gameId].players[i]
			fmt.Fprintf(w, `<button onclick="window.location.replace('/debug/join-existing-game?game-id=%v&player-id=%v&player-name=%v');">%v</button>`, gameId, player.id, url.QueryEscape(player.Name), player.Name)
		}
		fmt.Fprintln(w, `<br>`)
	}
	fmt.Fprintln(w, `</body>`)
}

func debugHandlerJoinExistingGame(w http.ResponseWriter, r *http.Request) {
	gameId := r.URL.Query().Get("game-id")
	playerId := r.URL.Query().Get("player-id")
	playerName := r.URL.Query().Get("player-name")
	if gameId == "" || playerId == "" || playerName == "" {
		http.Error(w, "400 bad request", http.StatusBadRequest)
	}

	// clear lobby-name
	w.Header().Add("Set-Cookie", `lobby-name=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`)
	// add headers
	w.Header().Add("Set-Cookie", fmt.Sprintf(`player-id=%s; path=/`, playerId))
	w.Header().Add("Set-Cookie", fmt.Sprintf(`player-name=%s; path=/`, playerName))
	w.Header().Add("Set-Cookie", fmt.Sprintf(`game-id=%s; path=/`, gameId))

	http.Redirect(w, r, "/game", http.StatusFound)
}

// createGameWithStaticUUIDs creates games with static UUIDs. Player UUIDs
// are incremented off from the last section of the game UUID.
// For game 00000000-0000-0000-0002-000000000000,
// Player 1 is 00000000-0000-0000-0002-000000000001,
// Player 2 is 00000000-0000-0000-0002-000000000002, etc
// Intended to be called before the http server starts up, so does not concern itself
// with locking or lobby name collisions.
func createGameWithStaticUUIDs(gameUuid uuid.UUID, numPlayers int) {
	if numPlayers < 0 || numPlayers > maxPlayers {
		serverlog.Fatalf("Invalid numPlayers arg to createGameWithStaticUUIDs(): %d", numPlayers)
		return
	}
	playerColors := []string{"#ff0000", "#00ff00", "#0000ff", "#ffff00"}
	lobbyName := "createGameWithStaticUUIDs"

	gameUuidSplit := strings.Split(gameUuid.String(), "-")
	gameUuidLastValue, err := strconv.ParseUint(gameUuidSplit[4], 16, 64)
	if err != nil {
		serverlog.Fatal(err)
	}
	delete(activeLobbies, lobbyName)
	for i := 0; i < numPlayers; i++ {
		playerUuidLastValue := (gameUuidLastValue + uint64(i) + 1) % 0x1000000000000
		playerUuidStr := fmt.Sprintf("%s-%s-%s-%s-%012x",
			gameUuidSplit[0],
			gameUuidSplit[1],
			gameUuidSplit[2],
			gameUuidSplit[3],
			playerUuidLastValue,
		)
		gameUuidSplit[4] = fmt.Sprintf("%012x", playerUuidLastValue)
		_, _ = joinLobby(lobbyName, "", playerColors[i])
		activeLobbies[lobbyName].player[i].id = uuid.Must(uuid.Parse(playerUuidStr))
	}

	game, _ := createGame(activeLobbies[lobbyName],
		map[string]any{
			"size":                  15,
			"starting_bites":        12,
			"starting_rerolls":      99,
			"new_bites_freq_factor": 4.0,
		})
	activeGames[gameUuid] = activeGames[game.uuid]
	delete(activeGames, game.uuid)
	delete(activeLobbies, lobbyName)
}

// Compare UUIDs byte-by-byte
func uuidLess(a, b uuid.UUID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return false
}
