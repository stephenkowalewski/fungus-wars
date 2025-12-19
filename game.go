package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const gbDefaultSize = 20
const gbMinSize = pieceMaskMaxLength
const gbMaxSize = 128
const gbStartOffsetDivisor = 5

const gbDefaultRandomizeStartPos = false
const gbDefaultStartBites = 4
const gbDefaultStartRerolls = 3
const gbDefaultHasBonusBiteCells = true
const gbBonusBiteAward = 3
const gbDefaultBonusRerollCells = 3
const gbBonusRerollAward = 1
const gbDefaultNewBiteFreqFactor = 1.0

const (
	gameModeCaptureFromPiece = iota
	gameModeCaptureAnywhereCurrentPlayer
	gameModeCaptureAnywhereAllPlayers
	gameModeCaptureMax // For input validation. Not a capture mode.
)

var activeGames = map[uuid.UUID]*Game{}
var activeGameMutex sync.Mutex
var activeGameMaxAge = 24 * time.Hour

type Game struct {
	mu                        sync.Mutex
	playerCount               int
	turn                      int // 0 == Player 1, etc
	players                   [maxPlayers]Player
	wsConns                   [maxPlayers]*websocket.Conn // for broadcasting messages
	scores                    [maxPlayers]int             // a score of 0 indicates that the player lost the game
	bites                     [maxPlayers]int
	rerolls                   [maxPlayers]int
	newCellsForBites          [maxPlayers]int // track each players' progress towards additional bites
	newCellsForBitesThreshold int             // placing or capturing this many pieces grants a bite, -1 disables this
	startBites                int
	startRerolls              int
	bonusBiteCells            bool
	bonusRerollCells          int
	board                     GameBoard
	rowCount                  int
	colCount                  int
	lastBoardUpdate           []int
	pieces                    []Piece
	nextPiece                 Piece
	captureMode               int
	randomizeStartPos         bool
	created                   time.Time
	fromLobby                 string
	isOver                    bool
	winLossDrawRecord         [maxPlayers]WinLossDraw
	uuid                      uuid.UUID
}

func (g *Game) shortDesc() string {
	state := "Active"
	if g.isOver {
		state = "Completed"
	}
	return fmt.Sprintf("%s Game(%d players, from lobby %s at %v)",
		state, g.playerCount, g.fromLobby, g.created,
	)
}

func (g *Game) String() string {
	var b strings.Builder
	b.WriteString(g.shortDesc())
	b.WriteRune('\n')

	b.WriteString("uuid: ")
	b.WriteString(fmt.Sprintf("%v\n", g.uuid))

	b.WriteString("turn: ")
	b.WriteString(fmt.Sprintf("%d\n", g.turn))

	b.WriteString("players:\n")
	for i := 0; i < g.playerCount; i++ {
		b.WriteString(fmt.Sprintf("- slot: %d\n", i))
		b.WriteString(fmt.Sprintf("  player: %v\n", &g.players[i]))
		b.WriteString(fmt.Sprintf("  score: %d\n", g.scores[i]))
		b.WriteString(fmt.Sprintf("  rerolls: %d\n", g.rerolls[i]))
		b.WriteString(fmt.Sprintf("  newCellsForBites: %d\n", g.newCellsForBites[i]))
	}

	b.WriteString("newCellsForBitesThreshold: ")
	b.WriteString(fmt.Sprintf("%d\n", g.newCellsForBitesThreshold))

	b.WriteString("startBites: ")
	b.WriteString(fmt.Sprintf("%d\n", g.startBites))

	b.WriteString("startRerolls: ")
	b.WriteString(fmt.Sprintf("%d\n", g.startRerolls))

	b.WriteString("bonusBiteCells: ")
	b.WriteString(fmt.Sprintf("%t\n", g.bonusBiteCells))

	b.WriteString("bonusRerollCells: ")
	b.WriteString(fmt.Sprintf("%d\n", g.bonusRerollCells))

	b.WriteString("pieces: ")
	b.WriteString(fmt.Sprintf("%v\n", g.pieces))

	b.WriteString("nextPiece: ")
	b.WriteString(fmt.Sprintf("%v\n", g.nextPiece))

	b.WriteString("captureMode: ")
	switch g.captureMode {
	case gameModeCaptureFromPiece:
		b.WriteString("gameModeCaptureFromPiece\n")
	case gameModeCaptureAnywhereCurrentPlayer:
		b.WriteString("gameModeCaptureAnywhereCurrentPlayer\n")
	case gameModeCaptureAnywhereAllPlayers:
		b.WriteString("gameModeCaptureAnywhereAllPlayers\n")
	default:
		b.WriteString(fmt.Sprintf("Unknown (%d)\n", g.captureMode))
	}

	b.WriteString("randomizeStartPos: ")
	b.WriteString(fmt.Sprintf("%t\n", g.randomizeStartPos))

	b.WriteString("lastBoardUpdate: ")
	b.WriteString(fmt.Sprintf("%v\n", g.lastBoardUpdate))

	b.WriteString("Board:\n")
	b.WriteString(g.board.String2D())

	return b.String()
}

// GameBoard is an collection of rows that make up the game board.
// GameBoard[0] is the top row.
type GameBoard [][]Cell

func (board GameBoard) String2D() string {
	var sb strings.Builder
	var columnWidth = 3

	for r := 0; r < len(board); r++ {
		for c := 0; c < len(board[0]); c++ {
			bytesWritten := 0
			cell := board[r][c]

			player := cell & CellMaskPlayer
			if player == 0 {
				sb.WriteByte('.')
				bytesWritten++
			} else {
				sb.WriteByte(byte('0' + player))
				bytesWritten++
			}

			if cell&CellFlagHome != 0 {
				sb.WriteByte('H')
				bytesWritten++
			}

			if cell&CellFlagBonusBite != 0 {
				sb.WriteByte('B')
				bytesWritten++
			}

			if cell&CellFlagBonusReroll != 0 {
				sb.WriteByte('R')
				bytesWritten++
			}

			sb.WriteByte(' ')
			bytesWritten++
			for bytesWritten < columnWidth {
				sb.WriteByte(' ')
				bytesWritten++
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// convert the row, column offset to an 1D index used by the client.
func (board GameBoard) getIndex1D(r, c int) int {
	return r*len(board[0]) + c
}

// convert the 1D offset to row and column.
func (board GameBoard) getIndex2D(index int) (int, int) {
	return index / len(board[0]), index % len(board[0])
}

// getPieceIndices return 1D indices for PieceMask at index
func (board *GameBoard) getPieceIndices(index int, pmask PieceMask) []int {
	var indices []int = make([]int, 0, 4)

	bRow, bCol := board.getIndex2D(index)
	for pRow := 0; pRow < pieceMaskMaxLength; pRow++ {
		for pCol := 0; pCol < pieceMaskMaxLength; pCol++ {
			if pmask.has(pRow, pCol) {
				indices = append(indices, board.getIndex1D(bRow+pRow, bCol+pCol))
			}
		}
	}

	return indices
}

// Cell is a space on the game board
// lower byte is the owner:
// - 0 means the square is unowned
// - 1 means it's owned by Player 1
// - 2 means it's owned by Player 2, etc
// upper byte is for special attributes of the square
// defined by CellFlag... constants
type Cell uint16

const (
	CellFlagHome Cell = 0x100 << iota
	CellFlagBonusBite
	CellFlagBonusReroll

	CellMaskPlayer = 0x00ff
	CellMaskFlags  = 0xff00
)

// PieceMask is bitmask that represents a set of one or more squares
// which are placed during a turn
type PieceMask uint32

const maxPieceRotations = 4

// a piece must fit in a pieceMaskMaxLength by pieceMaskMaxLength square
const (
	pieceMaskMaxLength                 = 5
	pieceMaskSectionMask     PieceMask = (1 << pieceMaskMaxLength) - 1
	pieceMaskFirstRowMask    PieceMask = pieceMaskSectionMask << (pieceMaskMaxLength * (pieceMaskMaxLength - 1))
	pieceMaskFirstColumnMask PieceMask = (((1 << (pieceMaskMaxLength * pieceMaskMaxLength)) - 1) / ((1 << pieceMaskMaxLength) - 1)) << (pieceMaskMaxLength - 1)
	pieceMaskFullMask        PieceMask = (1 << (pieceMaskMaxLength * pieceMaskMaxLength)) - 1
)

const (
	biteSmall PieceMask = 0b10000 << (pieceMaskMaxLength * (pieceMaskMaxLength - 1))
	biteLarge PieceMask = 0b11000_11000 << (pieceMaskMaxLength * (pieceMaskMaxLength - 2))
	biteNone  PieceMask = 0
)

var biteCosts = map[PieceMask]int{
	biteSmall: biteSmall.CalcBiteCost(),
	biteLarge: biteLarge.CalcBiteCost(),
}

// String returns a JavaScript (or Go) binary literal with bits grouped by row
func (p PieceMask) String() string {
	var sb strings.Builder

	sb.WriteString("0b")
	for r := 0; r < pieceMaskMaxLength; r++ {
		if r > 0 {
			sb.WriteByte('_')
		}
		for c := 0; c < pieceMaskMaxLength; c++ {
			if p.has(r, c) {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('0')
			}
		}
	}

	return sb.String()
}

func (p PieceMask) String2D() string {
	var sb strings.Builder

	for r := 0; r < pieceMaskMaxLength; r++ {
		if r > 0 {
			sb.WriteByte('\n')
		}
		for c := 0; c < pieceMaskMaxLength; c++ {
			if c > 0 {
				sb.WriteByte(' ')
			}
			if p.has(r, c) {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('0')
			}
		}
	}

	return sb.String()
}

// given a bit mask, determine the cost. Large bites have a slight discount
func (p PieceMask) CalcBiteCost() int {
	numBits := 0
	for i := 0; i < pieceMaskMaxLength*pieceMaskMaxLength; i++ {
		if p&(1<<i) != 0 {
			numBits++
		}
	}
	return numBits - (numBits / 4)
}

func maskAt(r, c int) PieceMask {
	// top-left is (0,0)
	return 1 << ((pieceMaskMaxLength-1-r)*pieceMaskMaxLength + (pieceMaskMaxLength - 1 - c))
}

// getSize returns the number of rows and columns in PieceMask p
func (p PieceMask) getSize() (int, int) {
	r := 0
	for i := 0; i < pieceMaskMaxLength; i++ {
		if p&(pieceMaskFirstRowMask>>(pieceMaskMaxLength*i)) != 0 {
			r = i + 1
		}
	}

	c := 0
	for i := 0; i < pieceMaskMaxLength; i++ {
		if p&(pieceMaskFirstColumnMask>>i) != 0 {
			c = i + 1
		}
	}

	return r, c
}

func (p PieceMask) has(r, c int) bool {
	if r >= pieceMaskMaxLength || c >= pieceMaskMaxLength {
		return false
	}
	return (p & maskAt(r, c)) != 0
}

// return new piece rotated clockwise 90 degrees and shifted to the top left
func (p PieceMask) rotate90() PieceMask {
	var rotated PieceMask

	// rotate
	for r := 0; r < pieceMaskMaxLength; r++ {
		for c := 0; c < pieceMaskMaxLength; c++ {
			if p.has(r, c) {
				newR, newC := pieceMaskMaxLength-1-c, r
				rotated |= 1 << (newR*pieceMaskMaxLength + newC)
			}
		}
	}

	return rotated.shiftUp()
}

// shiftUp moves a PieceMask to the top left as much as possible
func (p PieceMask) shiftUp() PieceMask {
	shifted := p

	// shift up
	for i := 0; i < pieceMaskMaxLength-1; i++ {
		if shifted&pieceMaskFirstRowMask == 0 {
			shifted <<= pieceMaskMaxLength
		} else {
			break
		}
	}

	// shift left
	for i := 0; i < pieceMaskMaxLength-1; i++ {
		if shifted&pieceMaskFirstColumnMask == 0 {
			var temp PieceMask
			for row := 0; row < pieceMaskMaxLength; row++ {
				rowMask := pieceMaskSectionMask << (row * pieceMaskMaxLength)
				shifted := (shifted & rowMask) << 1
				temp |= shifted & rowMask
			}
			shifted = temp
		} else {
			break
		}
	}

	return shifted
}

// generateRotations returns all 4 90 degree rotations of a PieceMask normalized
// to the top right.
func (p PieceMask) generateRotations() [maxPieceRotations]PieceMask {
	var rotations [maxPieceRotations]PieceMask
	rotations[0] = p.shiftUp()
	for i := 1; i < len(rotations); i++ {
		rotations[i] = rotations[i-1].rotate90()
	}
	return rotations
}

// Piece represents a game piece played during a turn
type Piece struct {
	// Mask contains all 90 degree rotations of a PieceMask
	Masks  [maxPieceRotations]PieceMask `json:"masks"`
	Weight float64
}

func (p Piece) has(mask PieceMask) bool {
	for _, m := range p.Masks {
		if m == mask {
			return true
		}
	}
	return false
}

var gbDefaultPieces = []Piece{
	// single square
	{PieceMask(0b10000).generateRotations(), 100},
	// two squares in a row
	{PieceMask(0b10000_10000).generateRotations(), 40},
	// three squares in a row
	{PieceMask(0b10000_10000_10000).generateRotations(), 100},
	// four squares in a row
	{PieceMask(0b10000_10000_10000_10000).generateRotations(), 12},
	// 2x2 square
	{PieceMask(0b11000_11000).generateRotations(), 100},
	// corner
	{PieceMask(0b10000_11000).generateRotations(), 100},
	// t block
	{PieceMask(0b11100_01000_00000).generateRotations(), 20},
	// z block
	{PieceMask(0b11000_01100_00000).generateRotations(), 12},
	// reverse z block
	{PieceMask(0b01100_11000_00000).generateRotations(), 12},
	// l block
	{PieceMask(0b10000_10000_11000).generateRotations(), 8},
	// reverse l block
	{PieceMask(0b11100_00100_00000).generateRotations(), 8},
	// b
	{PieceMask(0b11100_11000_00000).generateRotations(), 4},
	// c
	{PieceMask(0b11100_10100_00000).generateRotations(), 4},
	// d
	{PieceMask(0b01100_11100_00000).generateRotations(), 4},
	// skip line
	{PieceMask(0b10000_00000_10000).generateRotations(), 4},
	// skip pyramid
	{PieceMask(0b10000_01000_10000).generateRotations(), 1},
	// skip 5
	{PieceMask(0b10100_01000_10100).generateRotations(), 1},
	// 2 diag
	{PieceMask(0b10000_01000_00000).generateRotations(), 5},
	// 3 diag
	{PieceMask(0b10000_01000_00100).generateRotations(), 3},
	// 4 diag
	{PieceMask(0b10000_01000_00100_00010).generateRotations(), 2},
}

type WinLossDraw struct {
	W, L, D int
}

// addDraw adds a draw to the record for every player if all players have at
// least minScore (prevents adding a draw when resetting a new game)
func (game *Game) addDraw(minScore int) {
	for i := 0; i < game.playerCount; i++ {
		if game.scores[i] < minScore {
			return
		}
	}
	for i := 0; i < game.playerCount; i++ {
		game.winLossDrawRecord[i].D++
	}
}

// addWin adds a win to the record of player at index playerIndex and
// adds a loss to the other players' records
func (game *Game) addWin(playerIndex int) {
	for i := 0; i < game.playerCount; i++ {
		if i == playerIndex {
			game.winLossDrawRecord[i].W++
		} else {
			game.winLossDrawRecord[i].L++
		}
	}
}

// forfeits a game
func (game *Game) forfeitGame(whoami Player) error {
	playerIndex := -1
	for i := 0; i < game.playerCount; i++ {
		if game.players[i].id == whoami.id {
			playerIndex = i
			break
		}
	}
	if playerIndex < 0 {
		return errors.New("Count not find player in forfeitGame: " + whoami.id.String())
	}
	playerCell := Cell(playerIndex + 1)
	isPlayersTurn := playerIndex == game.turn

	game.lastBoardUpdate = game.lastBoardUpdate[:0]
	for r := 0; r < len(game.board); r++ {
		for c := 0; c < len(game.board[0]); c++ {
			cell := game.board[r][c]
			if cell&CellMaskPlayer == playerCell && cell&CellFlagHome != 0 {
				game.addPieceToBoard(0, game.board.getIndex1D(r, c), biteSmall)
				game.lastBoardUpdate = append(game.lastBoardUpdate, game.handleOrphanedCells()...)
			}
		}
	}
	game.updateScores()

	if isPlayersTurn || game.isOver {
		game.advanceTurn()
		game.setNextPiece()
	}
	return nil
}

// setStartingPositions places home cells on the board for each player.
func setStartingPositions(board GameBoard, playerCount int, randomize bool) {
	maxR := len(board) - 1
	maxC := len(board[0]) - 1

	// Holds how far from the edges to place home cells
	offsetR := len(board) / gbStartOffsetDivisor
	offsetC := len(board[0]) / gbStartOffsetDivisor

	startIndices := []func() (int, int){
		func() (int, int) { return offsetR, offsetC },               // top left
		func() (int, int) { return maxR - offsetR, maxC - offsetC }, // bottom right
		func() (int, int) { return offsetR, maxC - offsetC },        // top right
		func() (int, int) { return maxR - offsetR, offsetC },        // bottom left
		func() (int, int) { return offsetR, maxC / 2 },              // top middle
		func() (int, int) { return maxR / 2, maxC - offsetC },       // right middle
		func() (int, int) { return maxR - offsetR, maxC / 2 },       // bottom middle
		func() (int, int) { return maxR / 2, offsetC },              // left middle
	}

	if randomize {
		rand.Shuffle(len(startIndices), func(i, j int) {
			startIndices[i], startIndices[j] = startIndices[j], startIndices[i]
		})
	}

	for i := 0; i < playerCount; i++ {
		r, c := startIndices[i]()
		player := Cell(i + 1)
		board[r][c] = player | CellFlagHome
	}
}

// setBiteFlagPositions places a bonus bite cell in each corner of the board
func setBiteFlagPositions(board GameBoard) {
	maxR := len(board) - 1
	maxC := len(board[0]) - 1

	board[0][0] |= CellFlagBonusBite
	board[0][maxC] |= CellFlagBonusBite
	board[maxR][0] |= CellFlagBonusBite
	board[maxR][maxC] |= CellFlagBonusBite
}

// setRerollFlagPositions places bonus reroll cells randomly on the board
func setRerollFlagPositions(board GameBoard, count int) {
	for i := 0; i < count; i++ {
		var r, c int
		for attempt := 0; attempt < 5; attempt++ {
			r = rand.Intn(len(board))
			c = rand.Intn(len(board[0]))
			if board[r][c]&CellMaskFlags == 0 {
				break
			}
		}
		board[r][c] |= CellFlagBonusReroll
	}
}

// createGame creates a new game and returns the uuid for it.
// It also updates the activeGames global map to add the gameId
// optional arguments can be passed in opts
// fixme, move activeLobbies
func createGame(fromLobby *Lobby, opts map[string]any) (*Game, error) {
	var size int = gbDefaultSize
	var randomizeStartPos bool = gbDefaultRandomizeStartPos
	var startBites int = gbDefaultStartBites
	var startRerolls int = gbDefaultStartRerolls
	var bonusBiteCells bool = gbDefaultHasBonusBiteCells
	var bonusRerollCells int = gbDefaultBonusRerollCells
	var newBitesFreqFactor float64 = 1.0
	var captureMode int
	var pieces []Piece = gbDefaultPieces

	// parse options
	if val, ok := opts["size"].(int); ok {
		size = val
	}
	if val, ok := opts["randomize_start_positions"].(bool); ok {
		randomizeStartPos = val
	}
	if val, ok := opts["starting_bites"].(int); ok {
		startBites = val
	}
	if val, ok := opts["starting_rerolls"].(int); ok {
		startRerolls = val
	}
	if val, ok := opts["has_bonus_bite_cells"].(bool); ok {
		bonusBiteCells = val
	}
	if val, ok := opts["bonus_reroll_cells"].(int); ok {
		bonusRerollCells = val
	}
	if val, ok := opts["new_bites_freq_factor"].(float64); ok {
		newBitesFreqFactor = val
	}
	if val, ok := opts["capture_mode"].(int); ok {
		if val < 0 || val >= gameModeCaptureMax {
			return nil, errors.New("Invalid capture_mode parameter")
		}
		captureMode = val
	}
	if val, ok := opts["pieces"].([]Piece); ok {
		if len(pieces) > 0 {
			pieces = val
		}
	}

	// validate args
	if fromLobby == nil {
		return nil, errors.New("createGame lobby cannot be nil")
	}
	if size <= gbMinSize || size > gbMaxSize {
		return nil, errors.New("createGame size parameter out of bounds")
	}

	// get the players from the lobby
	players := [maxPlayers]Player{}
	playerCount := 0
	for i := 0; i < len(fromLobby.player); i++ {
		if fromLobby.player[i].lastSeen.IsZero() {
			continue
		}
		players[playerCount] = fromLobby.player[i]
		playerCount++
	}

	if playerCount < 2 {
		return nil, errors.New("A game requires at least two players")
	}

	// Build the board
	board := make(GameBoard, size)
	for i := range board {
		board[i] = make([]Cell, size)
	}
	setStartingPositions(board, playerCount, randomizeStartPos)
	if bonusBiteCells {
		setBiteFlagPositions(board)
	}
	if bonusRerollCells > 0 {
		setRerollFlagPositions(board, bonusRerollCells)
	}

	// adjust game options
	var cellsForBitesThreshold int
	if newBitesFreqFactor <= 0 {
		cellsForBitesThreshold = -1
	} else {
		cellsForBitesThreshold = int(float64(len(board)*2) / newBitesFreqFactor)
	}

	// create the game
	activeGameMutex.Lock()
	defer activeGameMutex.Unlock()
	gameId := uuid.New()
	game := &Game{
		playerCount:               playerCount,
		players:                   players,
		newCellsForBitesThreshold: cellsForBitesThreshold,
		startBites:                startBites,
		startRerolls:              startRerolls,
		bonusBiteCells:            bonusBiteCells,
		bonusRerollCells:          bonusRerollCells,
		board:                     board,
		rowCount:                  len(board),
		colCount:                  len(board[0]),
		pieces:                    pieces,
		captureMode:               captureMode,
		randomizeStartPos:         randomizeStartPos,
		created:                   time.Now(),
		fromLobby:                 fromLobby.name,
		uuid:                      gameId,
	}
	game.resetNewCellsForBites()
	game.resetBites()
	game.resetRerolls()
	game.updateScores()
	game.setNextPiece()
	activeGames[gameId] = game

	return game, nil
}

func (game *Game) resetGame() {
	game.mu.Lock()
	defer game.mu.Unlock()

	// Build the board
	size := len(game.board)
	board := make(GameBoard, size)
	for i := range board {
		board[i] = make([]Cell, size)
	}
	setStartingPositions(board, game.playerCount, game.randomizeStartPos)
	if game.bonusBiteCells {
		setBiteFlagPositions(board)
	}
	if game.bonusRerollCells > 0 {
		setRerollFlagPositions(board, game.bonusRerollCells)
	}

	// reset the game
	game.board = board
	game.lastBoardUpdate = nil
	game.turn = 0
	if !game.isOver {
		game.addDraw(2)
	}
	game.isOver = false
	game.created = time.Now()

	game.resetNewCellsForBites()
	game.resetBites()
	game.resetRerolls()
	game.updateScores()
	game.setNextPiece()
}

// getTurnInfo returns 2 values
// first return value: true if it is the player's turn
// second return value: The ownership Cell associated with this turn
func (game *Game) getTurnInfo(whoami Player) (bool, Cell) {
	requestorTurn := -1
	for i := 0; i < game.playerCount; i++ {
		if game.players[i].id == whoami.id {
			requestorTurn = i
			break
		}
	}
	if requestorTurn == game.turn {
		return true, Cell(requestorTurn + 1)
	}
	return false, 0
}

// isPieceAdjacentToPlayer returns true if any part of PieceMask mask at 1D index
// is adject to a cell owned by owner.
func (game *Game) isPieceAdjacentToPlayer(owner Cell, index int, mask PieceMask) bool {
	iRow, iCol := game.board.getIndex2D(index)
	for pRow := 0; pRow < pieceMaskMaxLength; pRow++ {
		for pCol := 0; pCol < pieceMaskMaxLength; pCol++ {
			if mask.has(pRow, pCol) {
				r, c := iRow+pRow, iCol+pCol
				// check above
				if r > 0 && game.board[r-1][c]&CellMaskPlayer == owner {
					return true
				}
				// check right
				if c < game.colCount-1 && game.board[r][c+1]&CellMaskPlayer == owner {
					return true
				}
				// check below
				if r < game.rowCount-1 && game.board[r+1][c]&CellMaskPlayer == owner {
					return true
				}
				// check left
				if c > 0 && game.board[r][c-1]&CellMaskPlayer == owner {
					return true
				}
			}
		}
	}
	return false
}

// isBiteAdjacentToPlayer returns true if any part of PieceMask mask at 1D index
// that is owned by an another player is adject to a cell owned by biteOwner.
// Difference from isPieceAdjacentToPlayer is factoring in ownership.
func (game *Game) isBiteAdjacentToPlayer(biteOwner Cell, index int, mask PieceMask) bool {
	iRow, iCol := game.board.getIndex2D(index)
	for pRow := 0; pRow < pieceMaskMaxLength; pRow++ {
		for pCol := 0; pCol < pieceMaskMaxLength; pCol++ {
			if mask.has(pRow, pCol) {
				r, c := iRow+pRow, iCol+pCol
				cellOwner := game.board[r][c] & CellMaskPlayer

				if cellOwner == 0 || cellOwner == biteOwner {
					continue
				}

				// check above
				if r > 0 && game.board[r-1][c]&CellMaskPlayer == biteOwner {
					return true
				}
				// check right
				if c < game.colCount-1 && game.board[r][c+1]&CellMaskPlayer == biteOwner {
					return true
				}
				// check below
				if r < game.rowCount-1 && game.board[r+1][c]&CellMaskPlayer == biteOwner {
					return true
				}
				// check left
				if c > 0 && game.board[r][c-1]&CellMaskPlayer == biteOwner {
					return true
				}
			}
		}
	}
	return false
}

// addPieceToBoard adds piece to the board at index. Owner can be 0 to make cells unowned
// Capturable flags such as CellFlagBonusBite and CellFlagBonusReroll are processed.
// Updates: game.board, game.bites, and game.rerolls
func (game *Game) addPieceToBoard(owner Cell, index int, mask PieceMask) {
	iRow, iCol := game.board.getIndex2D(index)
	for pRow := 0; pRow < pieceMaskMaxLength; pRow++ {
		for pCol := 0; pCol < pieceMaskMaxLength; pCol++ {
			if mask.has(pRow, pCol) {
				cell := game.board[iRow+pRow][iCol+pCol]
				cell = (cell & CellMaskFlags) | owner

				if owner != 0 {
					if cell&CellFlagBonusBite != 0 {
						cell &= ^CellFlagBonusBite
						game.bites[game.turn] += gbBonusBiteAward
					}
					if cell&CellFlagBonusReroll != 0 {
						cell &= ^CellFlagBonusReroll
						game.rerolls[game.turn] += gbBonusRerollAward
					}
				}

				game.board[iRow+pRow][iCol+pCol] = cell
			}
		}
	}
}

func (game *Game) isPieceInBounds(index int, pmask PieceMask) bool {
	iRow, iCol := game.board.getIndex2D(index)
	pRows, pCols := pmask.getSize()

	return iRow+pRows-1 < game.rowCount && iCol+pCols-1 < game.colCount
}

// true if all cells of mask are on free space
func (game *Game) isPieceOnFreeSpace(index int, mask PieceMask) bool {
	iRow, iCol := game.board.getIndex2D(index)
	for pRow := 0; pRow < pieceMaskMaxLength; pRow++ {
		for pCol := 0; pCol < pieceMaskMaxLength; pCol++ {
			if mask.has(pRow, pCol) {
				if (game.board[iRow+pRow][iCol+pCol] & CellMaskPlayer) != 0 {
					return false
				}
			}
		}
	}
	return true
}

// true if any cells of mask are owned by another player
func (game *Game) isPieceOnOpponentsSpace(player Cell, index int, mask PieceMask) bool {
	iRow, iCol := game.board.getIndex2D(index)
	for pRow := 0; pRow < pieceMaskMaxLength; pRow++ {
		for pCol := 0; pCol < pieceMaskMaxLength; pCol++ {
			if mask.has(pRow, pCol) {
				cellOwner := game.board[iRow+pRow][iCol+pCol] & CellMaskPlayer
				if cellOwner != 0 && cellOwner != player {
					return true
				}
			}
		}
	}
	return false
}

// Direction represents a direction on the game board.
// Positive row and col values mean to scan down and to the right, respectively.
type Direction struct {
	row int
	col int
}

func (d Direction) String() string {
	return fmt.Sprintf("(%+d, %+d)", d.row, d.col)
}

var (
	directionRight     = Direction{+0, +1}
	directionDownRight = Direction{+1, +1}
	directionDown      = Direction{+1, +0}
	directionDownLeft  = Direction{+1, -1}
	directionLeft      = Direction{+0, -1}
	directionUpLeft    = Direction{-1, -1}
	directionUp        = Direction{-1, +0}
	directionUpRight   = Direction{-1, +1}
)

// scanForCapture looks for possible captures starting at 1D index index.
func (game *Game) scanForCapture(player Cell, index int, direction Direction) (bool, []int) {
	var capture []int
	var found bool

	if direction.row == 0 && direction.col == 0 {
		panic("scanForCapture() direction row and col are 0. Breaking to avoid infinite loop.")
	}

	rowStart, colStart := game.board.getIndex2D(index)

	// return unless the starting cell is owned by the player
	if game.board[rowStart][colStart]&CellMaskPlayer != player {
		return found, capture
	}

	r, c := rowStart+direction.row, colStart+direction.col
	for r >= 0 && c >= 0 && r < game.rowCount && c < game.colCount {
		owner := game.board[r][c] & CellMaskPlayer
		if owner == 0 {
			break
		} else if owner == player {
			found = true
			break
		}

		capture = append(capture, r*game.colCount+c)
		r, c = r+direction.row, c+direction.col
	}

	if !found {
		capture = capture[:0]
	}
	return found, capture
}

// captureCells scans the game board for any pieces that can be captured by player
// pieces are captured if they is an continuous line of an opponent's pieces between
// the player's pieces.
// Updates: game.board
// Returns: list of cells that were updated (1D indexes)
func (game *Game) captureCells(player Cell) []int {
	var updates []int
	var tmp []int
	var hadCapture bool

	// keep finding the longest capture until there are no captures left
	longest := make([]int, 8)
	for len(longest) > 0 {
		longest = longest[:0]

		for r := 0; r < game.rowCount; r++ {
			for c := 0; c < game.colCount; c++ {
				for _, d := range []Direction{
					directionRight,
					directionDown,
					directionDownRight,
					directionDownLeft,
				} {
					hadCapture, tmp = game.scanForCapture(player, game.board.getIndex1D(r, c), d)
					if hadCapture && len(tmp) > len(longest) {
						if len(tmp) > cap(longest) {
							longest = make([]int, len(tmp))
						} else {
							longest = longest[:len(tmp)]
						}
						copy(longest, tmp)
					}
				}
			}
		}

		if len(longest) > 0 {
			for _, c := range longest {
				tmpR, tmpC := c/game.rowCount, c%game.rowCount
				game.board[tmpR][tmpC] &= CellMaskFlags
				game.board[tmpR][tmpC] |= player
			}
			updates = append(updates, longest...)
		}
	}

	return updates
}

// captureCellsFromPiece looks for captures starting at piece mask at 1D index.
// Cells that are captured may in turn capture additional cells.
func (game *Game) captureCellsFromPiece(owner Cell, index int, mask PieceMask) []int {
	var cellsToCheck []int = game.board.getPieceIndices(index, mask)
	var updates []int
	var tmp []int
	var hadCapture bool

	for len(cellsToCheck) > 0 {
		cell := cellsToCheck[0]
		cellsToCheck = cellsToCheck[1:]

		for _, d := range []Direction{
			directionDownLeft,
			directionLeft,
			directionUpLeft,
			directionUp,
			directionUpRight,
			directionRight,
			directionDownRight,
			directionDown,
		} {
			hadCapture, tmp = game.scanForCapture(owner, cell, d)
			if hadCapture {
				for _, c := range tmp {
					tmpR, tmpC := c/game.rowCount, c%game.rowCount
					game.board[tmpR][tmpC] &= CellMaskFlags
					game.board[tmpR][tmpC] |= owner
				}
				cellsToCheck = append(cellsToCheck, tmp...)
				updates = append(updates, tmp...)
			}
		}
	}

	return updates
}

// handleOrphanedCells removes any player cells with no path back to the home cell
// Updates: game.board
// Returns: list of cells that were updated (1D indexes)
func (game *Game) handleOrphanedCells() []int {
	var updates []int

	// flaggedCells holds the owners of Cells that have a path back to a home cell
	flaggedCells := make(GameBoard, game.rowCount)
	for i := 0; i < len(flaggedCells); i++ {
		flaggedCells[i] = make([]Cell, game.colCount)
	}

	var flagConnectedNeighbors func(Cell, int, int)
	flagConnectedNeighbors = func(whoami Cell, r, c int) {
		// above
		if c > 0 && game.board[r][c-1]&CellMaskPlayer == whoami {
			if flaggedCells[r][c-1] == 0 {
				flaggedCells[r][c-1] = whoami
				flagConnectedNeighbors(whoami, r, c-1)
			}
		}
		// below
		if c < game.colCount-1 && game.board[r][c+1]&CellMaskPlayer == whoami {
			if flaggedCells[r][c+1] == 0 {
				flaggedCells[r][c+1] = whoami
				flagConnectedNeighbors(whoami, r, c+1)
			}
		}
		// right
		if r < game.rowCount-1 && game.board[r+1][c]&CellMaskPlayer == whoami {
			if flaggedCells[r+1][c] == 0 {
				flaggedCells[r+1][c] = whoami
				flagConnectedNeighbors(whoami, r+1, c)
			}
		}
		// left
		if r > 0 && game.board[r-1][c]&CellMaskPlayer == whoami {
			if flaggedCells[r-1][c] == 0 {
				flaggedCells[r-1][c] = whoami
				flagConnectedNeighbors(whoami, r-1, c)
			}
		}
	}

	// scan the board for home cells and then mark neighbors as connected
	for r := 0; r < game.rowCount; r++ {
		for c := 0; c < game.colCount; c++ {
			// skip unowned squares
			whoami := game.board[r][c] & CellMaskPlayer
			if whoami == 0 {
				continue
			}

			// check for home cell
			if game.board[r][c]&CellFlagHome != 0 {
				flaggedCells[r][c] = whoami
				flagConnectedNeighbors(whoami, r, c)
			}
		}
	}

	// scan board for player cells not in flaggedCells
	// update game.board and build updates
	for r := 0; r < game.rowCount; r++ {
		for c := 0; c < game.colCount; c++ {
			if flaggedCells[r][c] == 0 && game.board[r][c]&CellMaskPlayer != 0 {
				game.board[r][c] &= CellMaskFlags
				updates = append(updates, game.board.getIndex1D(r, c))
			}
		}
	}

	return updates
}

// updateScores() sets game.scores and game.isOver
func (game *Game) updateScores() {
	var scores [maxPlayers]int
	for r := 0; r < len(game.board); r++ {
		for c := 0; c < len(game.board[0]); c++ {
			player := int(game.board[r][c] & CellMaskPlayer)
			if player > 0 && player <= maxPlayers {
				scores[player-1]++
			}
		}
	}
	game.scores = scores

	var activePlayerCount int
	var winnerIndex int
	for i := 0; i < maxPlayers; i++ {
		if scores[i] != 0 {
			activePlayerCount++
			winnerIndex = i
		}
	}

	game.isOver = activePlayerCount <= 1
	if activePlayerCount == 1 {
		game.addWin(winnerIndex)
	}
}

// updateNewCellsForBites() updates the current player's resetNewCellsForBites progress
func (game *Game) updateNewCellsForBites(addedCells int) {
	if addedCells <= 0 || game.newCellsForBitesThreshold <= 0 {
		return
	}
	game.newCellsForBites[game.turn] += addedCells
	newBites := game.newCellsForBites[game.turn] / game.newCellsForBitesThreshold
	remainder := game.newCellsForBites[game.turn] % game.newCellsForBitesThreshold
	if newBites > 0 {
		game.bites[game.turn] += newBites
		game.newCellsForBites[game.turn] = remainder
	}
}

// resetNewCellsForBites() resets each player's resetNewCellsForBites progress
func (game *Game) resetNewCellsForBites() {
	for i := 0; i < game.playerCount; i++ {
		game.newCellsForBites[i] = 0
	}
}

// resetBites() sets each player's available bite count to the game's starting value
func (game *Game) resetBites() {
	for i := 0; i < game.playerCount; i++ {
		game.bites[i] = game.startBites
	}
}

// resetRerolls() sets each player's available reroll count to the game's starting value
func (game *Game) resetRerolls() {
	for i := 0; i < game.playerCount; i++ {
		game.rerolls[i] = game.startRerolls
	}
}

// advanceTurn updates game.turn to the next player, skipping over players that have already lost.
// Sets turn to -1 if the game is over.
func (game *Game) advanceTurn() {
	if game.isOver {
		game.turn = -1
		return
	}
	for i := 0; i < game.playerCount; i++ {
		game.turn = (game.turn + 1) % game.playerCount
		if game.scores[game.turn] != 0 {
			break
		}
	}
}

func getWeightedRandomPiece(pieces []Piece) Piece {
	var totalWeight float64
	for _, p := range pieces {
		totalWeight += p.Weight
	}

	r := rand.Float64() * totalWeight
	for _, p := range pieces {
		r -= p.Weight
		if r <= 0 {
			return p
		}
	}

	// Fallback (should not happen if weights > 0)
	return pieces[0]
}

func (game *Game) setNextPiece() {
	if game.isOver {
		game.nextPiece = Piece{PieceMask(0).generateRotations(), 0}
	} else {
		game.nextPiece = getWeightedRandomPiece(game.pieces)
	}
}

func (game *Game) reroll(whoami Player) error {
	isPlayersTurn, _ := game.getTurnInfo(whoami)
	if !isPlayersTurn {
		return errors.New("Invalid update: not player's turn")
	}
	if game.rerolls[game.turn] <= 0 {
		return errors.New("Invalid update: no rerolls remaining")
	}

	// get the list of pieces, excluding the current piece
	rerollPieces := make([]Piece, 0, len(game.pieces)-1)
	for _, p := range game.pieces {
		if p != game.nextPiece {
			rerollPieces = append(rerollPieces, p)
		}
	}

	if len(rerollPieces) > 0 {
		game.nextPiece = getWeightedRandomPiece(rerollPieces)
	}

	game.rerolls[game.turn]--
	return nil
}

// placePiece adds the proposed update to the board, if allowed
func (game *Game) placePiece(whoami Player, index int, mask PieceMask) error {
	game.mu.Lock()
	defer game.mu.Unlock()

	if game.isOver {
		return errors.New("Invalid update: game over")
	}

	isPlayersTurn, pieceOwner := game.getTurnInfo(whoami)
	if !isPlayersTurn {
		return errors.New("Invalid update: not player's turn")
	}

	// validate legal move
	if index < 0 || index >= game.rowCount*game.colCount {
		return errors.New("Invalid update: index out of bounds")
	}
	if !game.isPieceInBounds(index, mask) {
		return errors.New("Invalid update: bite out of bounds")
	}
	if !game.isPieceOnFreeSpace(index, mask) {
		return errors.New("Invalid update: piece overlaps occupied space")
	}
	if !game.isPieceAdjacentToPlayer(pieceOwner, index, mask) {
		return errors.New("Invalid update: piece not adjactent")
	}
	if !game.nextPiece.has(mask) {
		return errors.New("Invalid update: unexpected game piece")
	}

	scoreBefore := game.scores[game.turn]

	game.addPieceToBoard(pieceOwner, index, mask)
	game.lastBoardUpdate = game.lastBoardUpdate[:0]

	// necessary for game pieces with gaps
	game.lastBoardUpdate = append(game.lastBoardUpdate, game.handleOrphanedCells()...)

	if game.captureMode == gameModeCaptureFromPiece {
		// Handle captures caused by newly placed piece.
		game.lastBoardUpdate = append(
			game.lastBoardUpdate,
			game.captureCellsFromPiece(pieceOwner, index, mask)...,
		)
	} else if game.captureMode == gameModeCaptureAnywhereCurrentPlayer {
		// Handle captures for the current player only.
		game.lastBoardUpdate = append(
			game.lastBoardUpdate,
			game.captureCells(pieceOwner)...,
		)
	} else if game.captureMode == gameModeCaptureAnywhereAllPlayers {
		// Handle captures for all players, anywhere on the board.
		// Current player goes last so that established pieces win.
		for i := 1; i <= game.playerCount; i++ {
			capturer := Cell(((game.turn + i) % game.playerCount) + 1)
			game.lastBoardUpdate = append(game.lastBoardUpdate,
				game.captureCells(capturer)...)
		}
	} else {
		panic(fmt.Sprintf("game.captureMode unimplemented. Got %d", game.captureMode))
	}

	game.lastBoardUpdate = append(game.lastBoardUpdate, game.handleOrphanedCells()...)
	game.updateScores()
	game.updateNewCellsForBites(game.scores[game.turn] - scoreBefore)
	game.advanceTurn()
	game.setNextPiece()
	return nil
}

// placeBite applies the proposed update to the board, if allowed
func (game *Game) placeBite(whoami Player, index int, bite PieceMask) error {
	game.mu.Lock()
	defer game.mu.Unlock()

	if game.isOver {
		return errors.New("Invalid update: game over")
	}

	isPlayersTurn, pieceOwner := game.getTurnInfo(whoami)
	if !isPlayersTurn {
		return errors.New("Invalid update: not player's turn")
	}

	// validate legal move
	if index < 0 || index >= game.rowCount*game.colCount {
		return errors.New("Invalid update: index out of bounds")
	}
	if !game.isPieceInBounds(index, bite) {
		return errors.New("Invalid update: bite out of bounds")
	}
	if !game.isPieceOnOpponentsSpace(pieceOwner, index, bite) {
		return errors.New("Invalid update: bite does not overlap an opponent's space")
	}
	if !game.isBiteAdjacentToPlayer(pieceOwner, index, bite) {
		return errors.New("Invalid update: bite not adjacent")
	}
	cost, ok := biteCosts[bite]
	if !ok {
		return errors.New("Invalid update: invalid bite mask")
	}
	if game.bites[game.turn] < cost {
		return errors.New("Invalid update: not enough bites remaining")
	}

	game.addPieceToBoard(0, index, bite)
	game.bites[game.turn] -= cost
	game.lastBoardUpdate = game.handleOrphanedCells()
	game.updateScores()
	game.advanceTurn()
	game.setNextPiece()
	return nil
}

func (game *Game) skipTurn(whoami Player) error {
	isPlayersTurn, _ := game.getTurnInfo(whoami)
	if !isPlayersTurn {
		return errors.New("Invalid update: not player's turn")
	}
	game.lastBoardUpdate = nil
	game.advanceTurn()
	game.setNextPiece()
	return nil
}

func (game *Game) clearLastBoardUpdate() {
	game.lastBoardUpdate = nil
}

// cleanUpGames removes inactive players from all lobbies and removes empty lobbies
func cleanUpGames(serverlog *log.Logger, debug bool) {
	activeGameMutex.Lock()
	defer activeGameMutex.Unlock()

	for k := range activeGames {
		if time.Since(activeGames[k].created) > activeGameMaxAge {
			serverlog.Println("cleanUpGames(): Purging old game " + activeGames[k].shortDesc())
			delete(activeGames, k)
		}
	}
}

func cleanUpGamesBackgroundTask(serverlog *log.Logger, debug bool) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		<-ticker.C
		cleanUpGames(serverlog, debug)
	}
}

// vim:nowrap
