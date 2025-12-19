package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type LobbyInfo struct {
	Members []Player `json:"members"`
	GameId  *string  `json:"game"` // nil or UUID of the game
}

// getLobbyPlayerFromReq gets the player based on cookies and optionally updates the
// lastSeen field for that player.
// return values: lobby name, Player, error
func getLobbyPlayerFromReq(r *http.Request, updateLastSeen bool) (string, Player, error) {
	idStr, err := getCookieWrapper(r, "player-id")
	if err != nil || idStr == "" {
		return "", Player{}, fmt.Errorf("Required cookie missing or empty: player-id")
	}
	pname, err := getCookieWrapper(r, "player-name")
	if err != nil || pname == "" {
		return "", Player{}, fmt.Errorf("Required cookie missing or empty: player-name")
	}
	lname, err := getCookieWrapper(r, "lobby-name")
	if err != nil || lname == "" {
		return "", Player{}, fmt.Errorf("Required cookie missing or empty: lobby-name")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return "", Player{}, err
	}

	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()
	lobby, ok := activeLobbies[lname]
	if !ok {
		return "", Player{}, errors.New("Lobby does not exist")
	}

	for i := 0; i < len(lobby.player); i++ {
		if lobby.player[i].id == id {
			if lobby.player[i].Name == pname {
				if updateLastSeen {
					lobby.player[i].lastSeen = time.Now()
				}
				return lname, lobby.player[i], nil
			}
		}
	}

	return lname, Player{}, errors.New("Player not found")
}

// lobbyJoinHandler adds a player to a lobby via joinLobby, sets cookies containing the lobby
// information, and redirects to a lobby landing page
func lobbyJoinHandler(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	lobby_name := strings.TrimSpace(args.Get("lobby"))
	if !lobbyAllowedNameRegex.MatchString(lobby_name) {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=invalid_lobby_name", http.StatusFound)
		return
	}

	player_name := strings.TrimSpace(args.Get("player-name"))
	if lobbyMemberReservedNameRegex.MatchString(player_name) {
		serverlog.Println("Unsetting reserved player name " + player_name)
		player_name = ""
	}

	player_color := strings.TrimSpace(args.Get("player-color"))

	player, err := joinLobby(lobby_name, player_name, player_color)
	if debug {
		serverlog.Println("lobbyJoinHandler:", activeLobbies)
	}
	if err == nil {
		w.Header().Add("Set-Cookie", fmt.Sprintf(`player-id=%s; path=/`, url.QueryEscape(player.id.String())))
		w.Header().Add("Set-Cookie", fmt.Sprintf(`player-name=%s; path=/`, url.QueryEscape(player.Name)))
		w.Header().Add("Set-Cookie", fmt.Sprintf(`lobby-name=%s; path=/`, url.QueryEscape(lobby_name)))
		http.Redirect(w, r, "/lobby", http.StatusFound)
	} else if err.Error() == "Lobby is full" {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=lobby_full", http.StatusFound)
	} else if err.Error() == "Game has already started" {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=game_started", http.StatusFound)
	} else if strings.HasPrefix(err.Error(), "Color parse error:") {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=invalid_color", http.StatusFound)
	} else if strings.HasPrefix(err.Error(), "Duplicate color:") {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=duplicate_player_color", http.StatusFound)
	} else if strings.HasPrefix(err.Error(), "Duplicate player name:") {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err=duplicate_player_name", http.StatusFound)
	} else {
		serverlog.Println(err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
	}
}

// lobbyLeaveHandler removes a player from the lobby and redirects back to the main page
func lobbyLeaveHandler(w http.ResponseWriter, r *http.Request) {
	lobby, player, err := getLobbyPlayerFromReq(r, false)
	if err != nil {
		serverlog.Println("lobbyLeaveHandler: Error from getLobbyPlayerFromReq: " + err.Error())
	}
	if err = leaveLobby(lobby, player.id); err != nil {
		serverlog.Println("lobbyLeaveHandler: Error from leaveLobby: " + err.Error())
	}
	clearCookies(w)

	args := r.URL.Query()
	userErrorCode := url.QueryEscape(args.Get("err"))
	if userErrorCode == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/static/error_pages/lobby.html?err="+userErrorCode, http.StatusFound)
	}
}

// lobbyListHandler returns the list of lobbies as json
func lobbyListHandler(w http.ResponseWriter, r *http.Request) {
	lobbyMutex.Lock()
	keys := make([]string, len(activeLobbies))
	i := 0
	for k := range activeLobbies {
		keys[i] = k
		i++
	}
	lobbyMutex.Unlock()

	lobbies := LobbyList{Lobbies: keys}
	j, err := json.Marshal(lobbies)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(j))
}

// lobbyInfoHandler returns the details of a lobby as json
func lobbyInfoHandler(w http.ResponseWriter, r *http.Request) {
	name, _, err := getLobbyPlayerFromReq(r, true)
	if err != nil {
		errMsg := err.Error()
		if debug {
			serverlog.Println("getLobbyPlayerFromReq had error: " + errMsg)
		}
		w.Header().Set("Content-Type", "application/json")
		userErrorCode := "unknown_error"
		if strings.HasPrefix(errMsg, "Required cookie missing or empty:") {
			userErrorCode = "missing_cookie"
		}
		fmt.Fprintf(w,
			`{"members":[],"error":true,"error_page":"/lobby/leave?err=%s"}`,
			userErrorCode,
		)
		return
	}

	info := LobbyInfo{}
	lobbyMutex.Lock()
	for k := range activeLobbies {
		if k == name {
			if activeLobbies[k].gameId != uuid.Nil {
				gameIdStr := activeLobbies[k].gameId.String()
				info.GameId = &gameIdStr
			}
			for _, u := range activeLobbies[k].player {
				if !u.lastSeen.IsZero() {
					info.Members = append(info.Members, u)
				}
			}
			break
		}
	}
	lobbyMutex.Unlock()
	j, err := json.Marshal(info)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(j))
}
