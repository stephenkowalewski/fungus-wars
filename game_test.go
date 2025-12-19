package main

import (
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetIndex1D(t *testing.T) {
	var board GameBoard

	board = GameBoard{
		{1, 2, 3, 0, 0, 0, 0, 0},
		{4, 5, 0, 0, 0, 0, 0, 0},
		{6, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}

	tests := []struct {
		row            int
		col            int
		expectedOffset int
		valueAtOffset  Cell
	}{
		{0, 0, 0, 1},
		{0, 1, 1, 2},
		{0, 2, 2, 3},
		{1, 0, 8, 4},
		{1, 1, 9, 5},
		{2, 0, 16, 6},
	}
	for _, test := range tests {
		res := board.getIndex1D(test.row, test.col)
		if res != test.expectedOffset {
			t.Errorf("getIndex1D(%d, %d) got %d. ExpectedOffset %d.\n%s",
				test.row,
				test.col,
				res,
				test.expectedOffset,
				board.String2D(),
			)
		}
		if board[res/len(board[0])][res%len(board[0])] != test.valueAtOffset {
			t.Errorf("Value at 1D offset %d expected to be %d, but board[%d][%d] is %d.\n%s",
				test.expectedOffset,
				test.valueAtOffset,
				test.row,
				test.col,
				board[res/len(board[0])][res%len(board[0])],
				board.String2D(),
			)
		}
		row, col := board.getIndex2D(res)
		if row != test.row || col != test.col {
			t.Errorf("getIndex2D() was not the inverse of getIndex1D(). Expected (%d, %d). Got (%d, %d).",
				test.row, test.col,
				row, col,
			)
		}
	}
}

func TestPieceMaskHas(t *testing.T) {
	type test struct {
		row      int
		col      int
		expected bool
	}
	var p PieceMask
	var tests []test

	p = 0b11100_00000_00000_00000_00000 // horizontal line, length 3
	tests = []test{
		{0, 0, true},
		{0, 1, true},
		{0, 2, true},
		{0, 3, false},
		{1, 0, false},
		{2, 0, false},
		{3, 0, false},
		{1, 1, false},
	}
	for _, testCase := range tests {
		res := p.has(testCase.row, testCase.col)
		if res != testCase.expected {
			t.Errorf("%v.Has(%d, %d) got %t. Expected %t.",
				p,
				testCase.row, testCase.col,
				res,
				testCase.expected,
			)
		}
	}

	p = 0b10000_10000_10000_00000_00000 // vertical line, length 3
	tests = []test{
		{0, 0, true},
		{0, 1, false},
		{0, 2, false},
		{0, 3, false},
		{1, 0, true},
		{2, 0, true},
		{3, 0, false},
		{1, 1, false},
	}
	for _, testCase := range tests {
		res := p.has(testCase.row, testCase.col)
		if res != testCase.expected {
			t.Errorf("%v.Has(%d, %d) got %t. Expected %t.",
				p,
				testCase.row, testCase.col,
				res,
				testCase.expected,
			)
		}
	}

}

func TestGamePieceRotations(t *testing.T) {
	// sanity check Piece.has()
	var p PieceMask
	var r, c int
	p = PieceMask(0b10000_00100_00000_00000_00000)
	r = 0
	c = 0
	if !p.has(r, c) {
		t.Errorf("Piece.has(%d,%d) failed for %v", r, c, p)
	}
	r = 1
	c = 2
	if !p.has(r, c) {
		t.Errorf("Piece.has(%d,%d) failed for %v", r, c, p)
	}
	r = 3
	c = 4
	if p.has(r, c) {
		t.Errorf("Piece.has(%d,%d) erroneously succeeded for %v", r, c, p)
	}

	// test rotations of a single square
	// all rotations should equal the original
	p = PieceMask(0b10000_00000_00000_00000_00000)
	piece := p.generateRotations()
	for i := 0; i < 4; i++ {
		if piece[i] != 0b10000_00000_00000_00000_00000 {
			t.Errorf("%v rotated %d times gave %v. Expected %v", p, i, piece[i], p)
		}
	}

	// test rotations of a corner with a long side
	p = PieceMask(0b11100_10000_00000_00000_00000)
	expected := [4]PieceMask{
		PieceMask(0b11100_10000_00000_00000_00000),
		PieceMask(0b11000_01000_01000_00000_00000),
		PieceMask(0b00100_11100_00000_00000_00000),
		PieceMask(0b10000_10000_11000_00000_00000),
	}
	piece = p.generateRotations()
	for i := 0; i < 4; i++ {
		if piece[i] != expected[i] {
			t.Errorf("%v rotated %d times gave %v. Expected %v", p, i, piece[i], expected[i])
		}
	}
}

func TestPieceMaskGetSize(t *testing.T) {
	type test struct {
		p             PieceMask
		expected_rows int
		expected_cols int
	}
	var tests []test

	tests = []test{
		{0b10000_00000_00000_00000_00000, 1, 1}, // 1x1 square
		{0b11000_11000_00000_00000_00000, 2, 2}, // 2x2 square
		{0b11100_00000_00000_00000_00000, 1, 3}, // horizontal line, length 3
		{0b10000_10000_10000_00000_00000, 3, 1}, // vertical line, length 3
	}
	for _, testCase := range tests {
		r, c := testCase.p.getSize()
		if r != testCase.expected_rows || c != testCase.expected_cols {
			t.Errorf("%v.getSize() got %d, %d. Expected %d, %d.",
				testCase.p,
				r, c,
				testCase.expected_rows, testCase.expected_cols,
			)
		}
	}
}

func TestPieceString2D(t *testing.T) {
	type test struct {
		p        PieceMask
		expected string
	}
	var tests []test

	tests = []test{
		{ // 1x1 square
			0b10000_00000_00000_00000_00000,
			"1 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0",
		},
		{ // 2x2 square
			0b11000_11000_00000_00000_00000,
			"1 1 0 0 0\n1 1 0 0 0\n0 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0",
		},
		{ // horizontal line, length 3
			0b11100_00000_00000_00000_00000,
			"1 1 1 0 0\n0 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0",
		},
		{ // vertical line, length 3
			0b10000_10000_10000_00000_00000,
			"1 0 0 0 0\n1 0 0 0 0\n1 0 0 0 0\n0 0 0 0 0\n0 0 0 0 0",
		},
	}
	for _, testCase := range tests {
		res := testCase.p.String2D()
		if res != testCase.expected {
			t.Errorf("%v.string2D() got:\n%s\nExpected:\n%s",
				testCase.p,
				res,
				testCase.expected,
			)
		}
	}
}

func TestGetPieceIndices(t *testing.T) {
	type test struct {
		i        int
		p        PieceMask
		expected []int
	}
	var board GameBoard
	var tests []test

	board = make(GameBoard, 12)
	for i := range board {
		board[i] = make([]Cell, 12)
	}

	tests = []test{
		{ // 1x1 square
			0,
			0b10000_00000_00000_00000_00000,
			[]int{0},
		},
		{ // 2x2 square
			1,
			0b11000_11000_00000_00000_00000,
			[]int{1, 2, 13, 14},
		},
		{ // horizontal line, length 3
			14,
			0b11100_00000_00000_00000_00000,
			[]int{14, 15, 16},
		},
		{ // vertical line, length 3
			30,
			0b10000_10000_10000_00000_00000,
			[]int{30, 42, 54},
		},
	}

	for _, testCase := range tests {
		res := board.getPieceIndices(testCase.i, testCase.p)
		if !slices.Equal(res, testCase.expected) {
			t.Errorf("getPieceIndices(%d, %v) got: %v\nExpected: %v",
				testCase.i, testCase.p,
				res,
				testCase.expected,
			)
		}
	}
}

func TestIsPieceInBounds(t *testing.T) {
	var game *Game
	var err error

	// create game
	lobbyName := "TestIsPieceInBounds"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err = createGame(activeLobbies[lobbyName], map[string]any{"size": 10})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	tests := []struct {
		index    int
		mask     PieceMask
		expected bool
	}{
		// single cell
		{0, PieceMask(0b00001).shiftUp(), true},
		{9, PieceMask(0b00001).shiftUp(), true},
		{42, PieceMask(0b00001).shiftUp(), true},
		{90, PieceMask(0b00001).shiftUp(), true},
		{99, PieceMask(0b00001).shiftUp(), true},

		// 2x2
		{0, PieceMask(0b00011_00011).shiftUp(), true},
		{8, PieceMask(0b00011_00011).shiftUp(), true},
		{9, PieceMask(0b00011_00011).shiftUp(), false},
		{42, PieceMask(0b00011_00011).shiftUp(), true},
		{80, PieceMask(0b00011_00011).shiftUp(), true},
		{90, PieceMask(0b00011_00011).shiftUp(), false},
		{99, PieceMask(0b00011_00011).shiftUp(), false},

		// tall column
		{0, PieceMask(0b10000_10000_10000_10000_10000), true},
		{9, PieceMask(0b10000_10000_00000_00000_00000), true},
		{42, PieceMask(0b10000_10000_10000_10000_10000), true},
		{67, PieceMask(0b10000_10000_10000_10000_10000), false},
		{99, PieceMask(0b10000_10000_10000_10000_10000), false},

		// long row
		{0, PieceMask(0b11111).shiftUp(), true},
		{9, PieceMask(0b11111).shiftUp(), false},
		{42, PieceMask(0b11111).shiftUp(), true},
		{90, PieceMask(0b11111).shiftUp(), true},
		{99, PieceMask(0b11111).shiftUp(), false},
	}
	for _, test := range tests {
		res := game.isPieceInBounds(test.index, test.mask)
		if res != test.expected {
			t.Errorf("isPieceInBounds(%d, %v) got %t. Expected %t.",
				test.index,
				test.mask,
				res,
				test.expected,
			)
		}
	}
}

func TestIsPieceOnFreeSpace(t *testing.T) {
	var game *Game
	var err error

	// create game
	lobbyName := "TestIsPieceOnFreeSpace"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err = createGame(activeLobbies[lobbyName], map[string]any{"size": 8})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	game.board = GameBoard{
		{0, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}

	tests := []struct {
		index    int
		mask     PieceMask
		expected bool
	}{
		// single cell
		{0, PieceMask(0b00001).shiftUp(), true},
		{1, PieceMask(0b00001).shiftUp(), false},
		{2, PieceMask(0b00001).shiftUp(), false},
		{3, PieceMask(0b00001).shiftUp(), true},

		// 2x2
		{0, PieceMask(0b00011_00011).shiftUp(), false},
		{1, PieceMask(0b00011_00011).shiftUp(), false},
		{2, PieceMask(0b00011_00011).shiftUp(), false},
		{3, PieceMask(0b00011_00011).shiftUp(), true},
		{35, PieceMask(0b00011_00011).shiftUp(), true},
		{36, PieceMask(0b00011_00011).shiftUp(), false},
		{37, PieceMask(0b00011_00011).shiftUp(), false},
		{38, PieceMask(0b00011_00011).shiftUp(), true},

		// tall column
		{0, PieceMask(0b10000_10000_10000_10000_10000), true},
		{1, PieceMask(0b10000_10000_10000_10000_10000), false},
		{2, PieceMask(0b10000_10000_10000_10000_10000), false},
		{3, PieceMask(0b10000_10000_10000_10000_10000), true},
		{12, PieceMask(0b10000_10000_10000_10000_10000), true},
		{13, PieceMask(0b10000_10000_10000_10000_10000), false},
		{14, PieceMask(0b10000_10000_10000_10000_10000), true},
		{15, PieceMask(0b10000_10000_10000_10000_10000), true},
	}
	for _, test := range tests {
		res := game.isPieceOnFreeSpace(test.index, test.mask)
		if res != test.expected {
			t.Errorf("isPieceOnFreeSpace(%d, %v) got %t. Expected %t.\n%s",
				test.index,
				test.mask,
				res,
				test.expected,
				game.board.String2D(),
			)
		}
	}
}

func TestIsPieceAdjacentToPlayer(t *testing.T) {
	var boardSize int = 10
	var lobbyName string
	var game *Game
	var board GameBoard
	var err error
	var result bool

	// create 2 player game
	lobbyName = "TestIsPieceAdjacentToPlayer2P"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err = createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// setup board
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board

	tests := []struct {
		owner    Cell
		index    int
		mask     PieceMask
		expected bool
	}{
		// test Player 1 piece bordering Player 1 cells
		{1, 31, PieceMask(0b10000_11000).shiftUp(), true},
		// test Player 2 piece bordering Player 2 cells
		{2, 16, PieceMask(0b10000_11000).shiftUp(), true},
		// test Player 1 piece bordering empty cells
		{1, 81, PieceMask(0b11000_11000).shiftUp(), false},
		// test Player 2 piece bordering empty cells
		{2, 74, PieceMask(0b11000_11000).shiftUp(), false},
		// test Player 1 piece bordering cell with flags (Player 2 home)
		{1, 69, PieceMask(0b10000_10000_10000).shiftUp(), false},
		// test Player 2 piece bordering cell with flags (Player 1 home)
		{2, 30, PieceMask(0b11110).shiftUp(), false},
	}

	for _, test := range tests {
		result = game.isPieceAdjacentToPlayer(test.owner, test.index, test.mask)
		if result != test.expected {
			t.Errorf("isPieceAdjacentToPlayer(%v, %d, %v) returned error %t instead of %t. Board:\n%s",
				test.owner, test.index, test.mask,
				result, test.expected,
				board.String2D(),
			)
		}
	}

	// create 4 player game
	lobbyName = "TestIsPieceAdjacentToPlayer4P"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	_ = joinLobbyWrapper(t, lobbyName, "p3", "")
	_ = joinLobbyWrapper(t, lobbyName, "p4", "")
	game, err = createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// setup board
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 3, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 4, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][2] |= CellFlagHome
	board[2][boardSize-3] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board

	tests = []struct {
		owner    Cell
		index    int
		mask     PieceMask
		expected bool
	}{
		// test Player 1 piece bordering Player 1 cells
		{1, 23, PieceMask(0b10000_11000).shiftUp(), true},
		// test Player 1 piece bordering Player 2 cells
		{1, 28, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 1 piece bordering Player 3 cells
		{1, 78, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 1 piece bordering Player 4 cells
		{1, 82, PieceMask(0b10000_11000).shiftUp(), false},

		// test Player 2 piece bordering Player 1 cells
		{2, 23, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 2 piece bordering Player 2 cells
		{2, 28, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 2 piece bordering Player 3 cells
		{2, 78, PieceMask(0b10000_11000).shiftUp(), true},
		// test Player 2 piece bordering Player 4 cells
		{2, 82, PieceMask(0b10000_11000).shiftUp(), false},

		// test Player 3 piece bordering Player 1 cells
		{3, 23, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 3 piece bordering Player 2 cells
		{3, 28, PieceMask(0b10000_11000).shiftUp(), true},
		// test Player 3 piece bordering Player 3 cells
		{3, 78, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 3 piece bordering Player 4 cells
		{3, 82, PieceMask(0b10000_11000).shiftUp(), false},

		// test Player 4 piece bordering Player 1 cells
		{4, 23, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 4 piece bordering Player 2 cells
		{4, 28, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 4 piece bordering Player 3 cells
		{4, 78, PieceMask(0b10000_11000).shiftUp(), false},
		// test Player 4 piece bordering Player 4 cells
		{4, 82, PieceMask(0b10000_11000).shiftUp(), true},
	}

	for _, test := range tests {
		result = game.isPieceAdjacentToPlayer(test.owner, test.index, test.mask)
		if result != test.expected {
			t.Errorf("isPieceAdjacentToPlayer(%v, %d, %v) returned error %t instead of %t. Board:\n%s",
				test.owner, test.index, test.mask,
				result, test.expected,
				board.String2D(),
			)
		}
	}

}

func TestIsBiteAdjacentToPlayer(t *testing.T) {
	var boardSize int = 10
	var board GameBoard
	var err error
	var result bool

	// create 2 player game
	lobbyName := "TestIsBiteAdjacentToPlayer2P"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// setup board
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 2, 2, 2, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 2, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 1, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 1, 1, 1, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board

	tests := []struct {
		owner    Cell
		index    int
		mask     PieceMask
		expected bool
	}{
		// test for Player 1 successful bite of Player 2
		{1, 36, biteSmall, true},
		// test for Player 2 successful bite of Player 1
		{2, 65, biteLarge, true},
		// test for Player 1 bite not adjacent to Player 1 cells
		{1, 27, biteLarge, false},
		// test for Player 2 bite not adjacent to Player 2 cells
		{2, 11, biteSmall, false},
		// test for Player 1 bite that is adjactent, but does not conatin any opponents' cells
		{1, 31, biteLarge, false},
		// test for Player 2 bite that is adjactent, but does not conatin any opponents' cells
		{2, 5, biteLarge, false},
	}

	for _, test := range tests {
		result = game.isBiteAdjacentToPlayer(test.owner, test.index, test.mask)
		if result != test.expected {
			t.Errorf("isBiteAdjacentToPlayer(%v, %d, %v) returned error %t instead of %t. Board:\n%s",
				test.owner, test.index, test.mask,
				result, test.expected,
				board.String2D(),
			)
		}
	}
}

func TestCreateGame(t *testing.T) {
	var err error

	// test createGame with an empty or nil lobby
	_, err = createGame(&Lobby{}, nil)
	if err == nil {
		t.Fatal("createGame with an empty lobby should have failed")
	}
	_, err = createGame(nil, nil)
	if err == nil {
		t.Fatal("createGame with an nil lobby should have failed")
	}

	// create a lobby to test with
	lobbyName := "test_game_lobby"
	p1 := joinLobbyWrapper(t, lobbyName, "p1", "")
	p2 := joinLobbyWrapper(t, lobbyName, "p2", "")

	// test createGame with our test lobby
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"has_bonus_bite_cells": false, "bonus_reroll_cells": 0})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// check that the game exists in activeGames
	_, ok := activeGames[game.uuid]
	if !ok {
		t.Fatal("game not found in activeGames")
	}

	// check player count
	if game.playerCount != 2 {
		t.Fatalf("expected 2 players, got %d", game.playerCount)
	}

	// check player details
	foundP1 := false
	foundP2 := false
	for i := 0; i < game.playerCount; i++ {
		if game.players[i].id == p1.id && game.players[i].Name == p1.Name {
			foundP1 = true
		}
		if game.players[i].id == p2.id && game.players[i].Name == p2.Name {
			foundP2 = true
		}
	}

	if !foundP1 {
		t.Error("Player p1 not found in the game")
	}
	if !foundP2 {
		t.Error("Player p2 not found in the game")
	}

	// check initial game board
	if len(game.board) != gbDefaultSize || len(game.board[0]) != gbDefaultSize {
		t.Errorf("Game board is the wrong size. Expected %d by %d. Got %d by %d",
			gbDefaultSize,
			gbDefaultSize,
			len(game.board),
			len(game.board[0]),
		)
	}
	for r := range game.board {
		for c := range game.board[r] {
			if r == len(game.board)/gbStartOffsetDivisor && c == len(game.board[r])/gbStartOffsetDivisor {
				if game.board[r][c]&CellMaskPlayer != 1 {
					t.Errorf("Player 1 starting position not found at game board[%d][%d]. Board is:\n%s", r, c, game.board.String2D())
				}
				continue
			}
			if r == len(game.board)-1-(len(game.board)/gbStartOffsetDivisor) && c == len(game.board[r])-1-(len(game.board[r])/gbStartOffsetDivisor) {
				if game.board[r][c]&CellMaskPlayer != 2 {
					t.Errorf("Player 2 starting position not found at game board[%d][%d]. Board is:\n%s", r, c, game.board.String2D())
				}
				continue
			}
			if game.board[r][c] != 0 {
				t.Errorf("Expected empty cell at board[%d][%d]. Board is:\n%s", r, c, game.board.String2D())
			}
		}
	}

	// check created time
	if time.Since(game.created) > 5*time.Second {
		t.Errorf("game.created is too old: %v", game.created)
	}

	// test creating a game from a lobby with an inactive player
	lobbyName2 := "test_game_lobby_2"
	_ = joinLobbyWrapper(t, lobbyName2, "p3", "")
	_ = joinLobbyWrapper(t, lobbyName2, "p4", "")
	activeLobbies[lobbyName2].player[1].lastSeen = time.Time{} // p4 is inactive

	_, err = createGame(activeLobbies[lobbyName2], map[string]any{})
	if err == nil {
		t.Fatal("createGame with one active player should have failed")
	}
	if err.Error() != "A game requires at least two players" {
		t.Fatalf("createGame with one active player gave wrong error: %v", err)
	}
	lobby2, ok := activeLobbies[lobbyName2]
	if !ok {
		t.Fatalf("lobby %s not found in activeLobbies", lobbyName2)
	}
	if lobby2.gameId != uuid.Nil {
		t.Fatal("gameId should not be set in the lobby for a failed game creation")
	}

	// test creating a game with one player
	lobbyName3 := "one_player_lobby"
	_ = joinLobbyWrapper(t, lobbyName3, "p5", "")
	_, err = createGame(activeLobbies[lobbyName3], map[string]any{})
	if err == nil {
		t.Fatal("createGame with one player should have failed")
	}
	if err.Error() != "A game requires at least two players" {
		t.Fatalf("createGame with one player gave wrong error: %v", err)
	}
	lobby3, ok := activeLobbies[lobbyName3]
	if !ok {
		t.Fatalf("lobby %s not found in activeLobbies", lobbyName3)
	}
	if lobby3.gameId != uuid.Nil {
		t.Fatal("gameId should not be set in the lobby for a failed game creation")
	}

	// test creating a game with an invalid board size
	lobbyName4 := "game_board_too_small"
	_ = joinLobbyWrapper(t, lobbyName4, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName4, "p2", "")
	_, err = createGame(activeLobbies[lobbyName4], map[string]any{"size": 1})
	if err == nil {
		t.Fatal("createGame with a 1x1 board should have failed")
	}
	if err.Error() != "createGame size parameter out of bounds" {
		t.Fatalf("createGame with a 1x1 board gave wrong error: %v", err)
	}
	lobbyName5 := "game_board_too_large"
	_ = joinLobbyWrapper(t, lobbyName5, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName5, "p2", "")
	_, err = createGame(activeLobbies[lobbyName5], map[string]any{"size": 1000})
	if err == nil {
		t.Fatal("createGame with a 1000x1000 board should have failed")
	}
	if err.Error() != "createGame size parameter out of bounds" {
		t.Fatalf("createGame with a 1000x1000 board gave wrong error: %v", err)
	}
}

func TestGameAdvanceTurn(t *testing.T) {
	var err error
	var players []string = []string{"p1", "p2", "p3", "p4"}
	var expected []int

	// create game
	lobbyName := "TestGameAdvanceTurn"
	for _, p := range players {
		_ = joinLobbyWrapper(t, lobbyName, p, "")
	}
	game, err := createGame(activeLobbies[lobbyName], map[string]any{})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// advance turn until wrap-around
	game.turn = 0
	expected = []int{0, 1, 2, 3, 0}
	for i := 0; i < len(expected); i++ {
		if game.turn != expected[i] {
			t.Errorf("Expected game.turn to be %d. Got %d.", expected[i], game.turn)
		}
		game.advanceTurn()
	}

	// remove players 2 and 4
	for r := 0; r < len(game.board); r++ {
		for c := 0; c < len(game.board[0]); c++ {
			cellOwner := int(game.board[r][c] & CellMaskPlayer)
			if cellOwner == 2 || cellOwner == 4 {
				game.board[r][c] &= CellMaskFlags
			}
		}
	}
	game.updateScores()

	// Now that p2 and p4 are removed, advance turn until we get back to player 1
	game.turn = 0
	expected = []int{0, 2, 0}
	for i := 0; i < len(expected); i++ {
		if game.turn != expected[i] {
			t.Errorf("Expected game.turn to be %d. Got %d.", expected[i], game.turn)
		}
		game.advanceTurn()
	}

	// remove player 1, so that player 3 wins
	for r := 0; r < len(game.board); r++ {
		for c := 0; c < len(game.board[0]); c++ {
			cellOwner := int(game.board[r][c] & CellMaskPlayer)
			if cellOwner == 1 {
				game.board[r][c] &= CellMaskFlags
			}
		}
	}
	game.updateScores()
	game.advanceTurn()

	// the current turn should be -1 and advancing the turn should also give -1
	expected = []int{-1, -1}
	for i := 0; i < len(expected); i++ {
		if game.turn != expected[i] {
			t.Errorf("Expected game.turn to be %d. Got %d.", expected[i], game.turn)
			t.Error(game.String())
		}
		game.advanceTurn()
	}
}

func TestScanForCapture(t *testing.T) {
	var boardSize int = 8
	var board GameBoard
	var found bool
	var expectedFound bool
	var capture []int
	var expectedCapture []int
	var startIndex int
	var direction Direction

	// create game
	lobbyName := "TestScanForCapture"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// test horizontal capture
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 2, 2, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	// left to right
	expectedFound = true
	expectedCapture = []int{11, 12}
	startIndex = 10
	direction = directionRight
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// right to left
	expectedFound = true
	expectedCapture = []int{12, 11}
	startIndex = 13
	direction = directionLeft
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}

	// test vertical capture
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 2, 2, 1, 0, 0},
		{0, 0, 2, 0, 0, 0, 0, 0},
		{0, 0, 2, 0, 0, 0, 0, 0},
		{0, 0, 2, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	// top to bottom
	expectedFound = true
	expectedCapture = []int{18, 26, 34}
	startIndex = 10
	direction = directionDown
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// bottom to top
	expectedFound = true
	expectedCapture = []int{34, 26, 18}
	startIndex = 42
	direction = directionUp
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}

	// test forward diagonal capture
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	// top to bottom
	expectedFound = true
	expectedCapture = []int{19}
	startIndex = 10
	direction = directionDownRight
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// bottom to top
	expectedFound = true
	expectedCapture = []int{19}
	startIndex = 28
	direction = directionUpLeft
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}

	// test reverse diagonal capture
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 2, 0},
		{0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 2, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	// top to bottom
	expectedFound = true
	expectedCapture = []int{22, 29, 36, 43}
	startIndex = 15
	direction = directionDownLeft
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// bottom to top
	expectedFound = true
	expectedCapture = []int{19}
	expectedCapture = []int{43, 36, 29, 22}
	startIndex = 50
	direction = directionUpRight
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}

	// test failed scans against edges
	board = GameBoard{
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 2, 2},
		{2, 2, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 2, 0, 0, 0},
		{0, 0, 0, 0, 2, 0, 0, 0},
	}
	game.board = board
	// top
	expectedFound = false
	expectedCapture = []int{}
	startIndex = 19
	direction = directionUp
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// bottom
	expectedFound = false
	expectedCapture = []int{}
	startIndex = 44
	direction = directionDown
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// right
	expectedFound = false
	expectedCapture = []int{}
	startIndex = 37
	direction = directionRight
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
	// left
	expectedFound = false
	expectedCapture = []int{}
	startIndex = 42
	direction = directionLeft
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}

	// test failed scan due to gap
	board = GameBoard{
		{0, 1, 2, 2, 0, 2, 1, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	expectedFound = false
	expectedCapture = []int{}
	startIndex = 1
	direction = directionRight
	found, capture = game.scanForCapture(1, startIndex, direction)
	if found != expectedFound || !slices.Equal(capture, expectedCapture) {
		t.Errorf("unexpected result from scanForCapture(1, %d, %v)\nExpected %t, %v.\nGot %t, %v\nboard:\n%s",
			startIndex, direction,
			expectedFound, expectedCapture,
			found, capture,
			game.board.String2D(),
		)
	}
}

func TestGameCaptureCellFromPiece(t *testing.T) {
	var boardSize int = 8
	var board GameBoard
	var expectedBoard GameBoard
	var boardError bool
	var updates []int
	var expectedUpdates []int
	var pieceOwner Cell
	var pieceIndex int
	var piece Piece

	// create game
	lobbyName := "TestGameCaptureCellFromPiece"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// test horizontal capture
	pieceOwner = 1
	pieceIndex = 13
	piece = Piece{PieceMask(0b10000).generateRotations(), 1}
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 2, 2, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	game.nextPiece = piece
	updates = game.captureCellsFromPiece(pieceOwner, pieceIndex, piece.Masks[0])
	expectedUpdates = []int{
		board.getIndex1D(1, 3),
		board.getIndex1D(1, 4),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad return value. Expected %v. Got %v",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedUpdates, updates,
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf(
					"captureCellsFromPiece(%d, %d, %v): Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					pieceOwner, pieceIndex, piece.Masks[0],
					r, c, expectedBoard[r][c], game.board[r][c],
				)
			}
		}
	}
	if boardError {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedBoard.String2D(),
			game.board.String2D(),
		)
	}

	// test double capture, horizontal then diagonal
	pieceOwner = 1
	pieceIndex = 13
	piece = Piece{PieceMask(0b10000).generateRotations(), 1}
	board = GameBoard{
		{0, 0, 0, 2, 2, 2, 2, 0},
		{0, 0, 1, 2, 2, 1, 0, 0},
		{0, 0, 1, 2, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 2, 2, 2, 2, 0},
		{0, 0, 1, 1, 1, 1, 0, 0},
		{0, 0, 1, 1, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	game.nextPiece = piece
	updates = game.captureCellsFromPiece(pieceOwner, pieceIndex, piece.Masks[0])
	expectedUpdates = []int{
		board.getIndex1D(1, 3),
		board.getIndex1D(1, 4),
		board.getIndex1D(2, 3),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad return value. Expected %v. Got %v",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedUpdates, updates,
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf(
					"captureCellsFromPiece(%d, %d, %v): Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					pieceOwner, pieceIndex, piece.Masks[0],
					r, c, expectedBoard[r][c], game.board[r][c],
				)
			}
		}
	}
	if boardError {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedBoard.String2D(),
			game.board.String2D(),
		)
	}

	// test no capture, not adjacent
	pieceOwner = 1
	pieceIndex = 9
	piece = Piece{PieceMask(0b10000).generateRotations(), 1}
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 2, 2, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 2, 2, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	game.nextPiece = piece
	updates = game.captureCellsFromPiece(pieceOwner, pieceIndex, piece.Masks[0])
	expectedUpdates = []int{}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad return value. Expected %v. Got %v",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedUpdates, updates,
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf(
					"captureCellsFromPiece(%d, %d, %v): Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					pieceOwner, pieceIndex, piece.Masks[0],
					r, c, expectedBoard[r][c], game.board[r][c],
				)
			}
		}
	}
	if boardError {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedBoard.String2D(),
			game.board.String2D(),
		)
	}

	// test no capture - blank space in the middle
	pieceOwner = 1
	pieceIndex = 17
	piece = Piece{PieceMask(0b10000).generateRotations(), 1}
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{1, 1, 0, 0, 0, 0, 0, 0},
		{1, 1, 0, 0, 0, 0, 0, 0},
		{1, 2, 2, 2, 0, 0, 0, 0},
		{1, 0, 0, 2, 0, 0, 0, 0},
		{1, 2, 2, 2, 0, 0, 0, 0},
		{1, 1, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{1, 1, 0, 0, 0, 0, 0, 0},
		{1, 1, 0, 0, 0, 0, 0, 0},
		{1, 2, 2, 2, 0, 0, 0, 0},
		{1, 0, 0, 2, 0, 0, 0, 0},
		{1, 2, 2, 2, 0, 0, 0, 0},
		{1, 1, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	game.nextPiece = piece
	updates = game.captureCellsFromPiece(pieceOwner, pieceIndex, piece.Masks[0])
	expectedUpdates = []int{}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad return value. Expected %v. Got %v",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedUpdates, updates,
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf(
					"captureCellsFromPiece(%d, %d, %v): Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					pieceOwner, pieceIndex, piece.Masks[0],
					r, c, expectedBoard[r][c], game.board[r][c],
				)
			}
		}
	}
	if boardError {
		t.Errorf(
			"captureCellsFromPiece(%d, %d, %v): Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			pieceOwner, pieceIndex, piece.Masks[0],
			expectedBoard.String2D(),
			game.board.String2D(),
		)
	}

}

func TestGameCaptureCells(t *testing.T) {
	var boardSize int = 8
	var board GameBoard
	var expectedBoard GameBoard
	var boardError bool
	var updates []int
	var expectedUpdates []int
	var captureType string

	// create game
	lobbyName := "test_capture_cells_lobby"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// test horizontal capture
	captureType = "horizontal"
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 2, 2, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{
		board.getIndex1D(1, 3),
		board.getIndex1D(1, 4),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test vertical capture
	captureType = "vertical"
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 2, 0},
		{0, 0, 0, 0, 0, 0, 2, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{
		board.getIndex1D(3, 6),
		board.getIndex1D(4, 6),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test down+right diagonal capture
	captureType = "down+right diagonal"
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 0, 2, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{
		board.getIndex1D(2, 3),
		board.getIndex1D(3, 4),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test down+left diagonal capture
	captureType = "down+left diagonal"
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 2, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 1, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{
		board.getIndex1D(2, 5),
		board.getIndex1D(3, 4),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test double capture, vertical then horizontal
	captureType = "vertical then horizontal"
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{1, 0, 0, 1, 0, 0, 0, 0},
		{2, 2, 2, 1, 0, 0, 0, 0},
		{1, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{1, 0, 0, 1, 0, 0, 0, 0},
		{1, 1, 1, 1, 0, 0, 0, 0},
		{1, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{
		board.getIndex1D(2, 0),
		board.getIndex1D(2, 1),
		board.getIndex1D(2, 2),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test double capture, horizontal then vertical
	captureType = "horizontal then vertical"
	board = GameBoard{
		{0, 0, 1, 2, 1, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{
		board.getIndex1D(0, 3),
		board.getIndex1D(1, 3),
		board.getIndex1D(2, 3),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test no capture - blank space in the middle
	captureType = "no capture (blank space)"
	board = GameBoard{
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 2, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.captureCells(1)
	expectedUpdates = []int{}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("captureCells %s capture failed. Bad return value. Expected %v. Got %v", captureType, expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("captureCells %s capture: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					captureType, r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("captureCells %s capture: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			captureType,
			expectedBoard.String2D(),
			game.board.String2D())
	}
}

func TestGameHandleOrphanedCells(t *testing.T) {
	var boardSize int = 8
	var board GameBoard
	var expectedBoard GameBoard
	var boardError bool
	var updates []int
	var expectedUpdates []int

	// create game
	lobbyName := "test_orphaned_cells_lobby"
	_ = joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// test removal of board[6][5]
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 1, 1, 0, 0},
		{0, 0, 0, CellFlagHome | 1, 0, 1, 0, 0},
		{0, 0, 1, 0, 0, 1, 0, 0},
		{0, 0, 1, 1, 1, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 1, 1, 0, 0},
		{0, 0, 0, CellFlagHome | 1, 0, 1, 0, 0},
		{0, 0, 1, 0, 0, 1, 0, 0},
		{0, 0, 1, 1, 1, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	game.board = board
	updates = game.handleOrphanedCells()
	expectedUpdates = []int{board.getIndex1D(6, 5)}
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("handleOrphanedCells: Bad return value. Expected %v. Got %v", expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("handleOrphanedCells: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("handleOrphanedCells: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test removal at sides with more than one CellFlagHome
	board = GameBoard{
		{0, 0, 1, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{1, 1, 1, 1, CellFlagHome | 1, 1, 1, 1},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{2, 0, 0, 0, 1, 0, 0, 4},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, CellFlagHome | 2, 2, 0, 1, 0, 3, 0},
	}
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{1, 1, 1, 1, CellFlagHome | 1, 1, 1, 1},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, CellFlagHome | 2, 2, 0, 1, 0, 0, 0},
	}
	game.board = board
	updates = game.handleOrphanedCells()
	expectedUpdates = []int{
		board.getIndex1D(0, 2),
		board.getIndex1D(5, 0),
		board.getIndex1D(5, 7),
		board.getIndex1D(7, 6),
	}
	sort.Ints(updates)
	sort.Ints(expectedUpdates)
	if !slices.Equal(updates, expectedUpdates) {
		t.Errorf("handleOrphanedCells: Bad return value. Expected %v. Got %v", expectedUpdates, updates)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("handleOrphanedCells: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("handleOrphanedCells: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}
}

func TestPlaceBite(t *testing.T) {
	var boardSize int = 10
	var board, expectedBoard GameBoard
	var biteIndex int
	var biteMask PieceMask
	var err error
	var expectedErrMsg string
	var boardError bool

	// create game
	lobbyName := "TestPlaceBite"
	p1 := joinLobbyWrapper(t, lobbyName, "p1", "")
	_ = joinLobbyWrapper(t, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// test small bite, no orphaned cells
	biteIndex = 36
	biteMask = biteSmall
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard[2][2] |= CellFlagHome
	expectedBoard[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err != nil {
		t.Errorf("placeBite(%v, %d, %v) returned error %v. Board:\n%s",
			p1.Name, biteIndex, biteMask,
			err,
			board.String2D(),
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("placeBite: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("placeBite: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test small bite, with orphaned cells
	biteIndex = 36
	biteMask = biteSmall
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 2, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 2, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard[2][2] |= CellFlagHome
	expectedBoard[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err != nil {
		t.Errorf("placeBite(%v, %d, %v) returned error %v. Board:\n%s",
			p1.Name, biteIndex, biteMask,
			err,
			board.String2D(),
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("placeBite: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("placeBite: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// test large bite over mix of opponent and empty cells
	biteIndex = 26
	biteMask = biteLarge
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard[2][2] |= CellFlagHome
	expectedBoard[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err != nil {
		t.Errorf("placeBite(%v, %d, %v) returned error %v. Board:\n%s",
			p1.Name, biteIndex, biteMask,
			err,
			board.String2D(),
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("placeBite: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("placeBite: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}

	// Look for error on index out of bounds
	expectedErrMsg = "Invalid update: index out of bounds"
	biteIndex = 101
	biteMask = biteLarge
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err == nil || err.Error() != expectedErrMsg {
		t.Errorf("placeBite(%v, %d, %v) did not return error when it should have.\nGot: %v, Expected: %s\nBoard:\n%s",
			p1.Name, biteIndex, biteMask,
			err, expectedErrMsg,
			board.String2D(),
		)
	}

	// Look for error for bite on empty space and player's own cells
	expectedErrMsg = "Invalid update: bite does not overlap an opponent's space"
	biteIndex = 14
	biteMask = biteLarge
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err == nil || err.Error() != expectedErrMsg {
		t.Errorf("placeBite(%v, %d, %v) did not return error when it should have.\nGot: %v, Expected: %s\nBoard:\n%s",
			p1.Name, biteIndex, biteMask,
			err, expectedErrMsg,
			board.String2D(),
		)
	}

	// Look for error for bite that does not border player's cells
	expectedErrMsg = "Invalid update: bite not adjacent"
	biteIndex = 27
	biteMask = biteLarge
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err == nil || err.Error() != expectedErrMsg {
		t.Errorf("placeBite(%v, %d, %v) did not return error when it should have.\nGot: %v, Expected: %s\nBoard:\n%s",
			p1.Name, biteIndex, biteMask,
			err, expectedErrMsg,
			board.String2D(),
		)
	}

	// Look for error for bite whose mask does border player's cells,
	// but does not contain a bordering opponent cell.
	// In other words, the bite borders only on empty cells.
	expectedErrMsg = "Invalid update: bite not adjacent"
	biteIndex = 36
	biteMask = biteLarge
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost()
	err = game.placeBite(p1, biteIndex, biteMask)
	if err == nil || err.Error() != expectedErrMsg {
		t.Errorf("placeBite(%v, %d, %v) did not return error when it should have.\nGot: %v, Expected: %s\nBoard:\n%s",
			p1.Name, biteIndex, biteMask,
			err, expectedErrMsg,
			board.String2D(),
		)
	}

	// test that the bite is denied if the player doesn't have enough bites - Small bite
	expectedErrMsg = "Invalid update: not enough bites remaining"
	biteIndex = 46
	biteMask = biteSmall
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost() - 1
	err = game.placeBite(p1, biteIndex, biteMask)
	if err == nil || err.Error() != expectedErrMsg {
		t.Errorf("placeBite(%v, %d, %v) did not return error when it should have.\nGot: %v, Expected: %s\nBoard:\n%s",
			p1.Name, biteIndex, biteMask,
			err, expectedErrMsg,
			board.String2D(),
		)
	}

	// test that the bite is denied if the player doesn't have enough bites - Large bite
	expectedErrMsg = "Invalid update: not enough bites remaining"
	biteIndex = 46
	biteMask = biteLarge
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	game.board = board
	game.turn = 0
	game.bites[0] = biteMask.CalcBiteCost() - 1
	err = game.placeBite(p1, biteIndex, biteMask)
	if err == nil || err.Error() != expectedErrMsg {
		t.Errorf("placeBite(%v, %d, %v) did not return error when it should have.\nGot: %v, Expected: %s\nBoard:\n%s",
			p1.Name, biteIndex, biteMask,
			err, expectedErrMsg,
			board.String2D(),
		)
	}
}

func TestForfeitGame(t *testing.T) {
	var boardSize int = 10
	var board, expectedBoard GameBoard
	var err error
	var boardError bool
	var playerNames []string = []string{"p1", "p2", "p3", "p4"}
	var players []Player = make([]Player, len(playerNames))

	// create game
	lobbyName := "TestForfeitGame"
	for i, p := range playerNames {
		players[i] = joinLobbyWrapper(t, lobbyName, p, "")
	}
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		t.Fatalf("createGame failed: %v", err)
	}

	// player 2 forfeits (during player 1's turn)
	// turn remains unchanged
	game.turn = 0
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 4, 4, 0},
		{0, 1, 1, 1, 1, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 1, 1, 2, 2, 0, 0},
		{0, 0, 0, 0, 0, 0, 2, 2, 0, 0},
		{0, 0, 0, 0, 3, 0, 0, 2, 0, 0},
		{0, 0, 3, 3, 3, 0, 0, 2, 2, 0},
		{0, 0, 3, 0, 3, 3, 0, 2, 2, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	board[boardSize-3][2] |= CellFlagHome
	board[2][boardSize-3] |= CellFlagHome
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 4, 4, 0},
		{0, 1, 1, 1, 1, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 1, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 3, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 0, 3, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard[2][2] |= CellFlagHome
	expectedBoard[boardSize-3][boardSize-3] |= CellFlagHome
	expectedBoard[boardSize-3][2] |= CellFlagHome
	expectedBoard[2][boardSize-3] |= CellFlagHome
	game.board = board
	err = game.forfeitGame(players[1])
	if err != nil {
		t.Errorf("forfeitGame returned error %v. Board:\n%s",
			err,
			board.String2D(),
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("forfeitGame: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("forfeitGame: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}
	if game.turn != 0 {
		t.Errorf("forfeitGame unexpectedly changed game.turn from %d to %d. Game:\n%v\n", 0, game.turn, game.String())
	}

	// player 1 forfeits (during own turn)
	// turn advances
	game.turn = 0
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 0, 0, 0, 0, 4, 4, 0},
		{0, 1, 1, 1, 1, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 1, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 3, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 0, 3, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	board[boardSize-3][2] |= CellFlagHome
	board[2][boardSize-3] |= CellFlagHome
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 0, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 3, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 0, 3, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard[2][2] |= CellFlagHome
	expectedBoard[boardSize-3][boardSize-3] |= CellFlagHome
	expectedBoard[boardSize-3][2] |= CellFlagHome
	expectedBoard[2][boardSize-3] |= CellFlagHome
	game.board = board
	err = game.forfeitGame(players[0])
	if err != nil {
		t.Errorf("forfeitGame returned error %v. Board:\n%s",
			err,
			board.String2D(),
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("forfeitGame: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("forfeitGame: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}
	if game.turn != 2 {
		t.Errorf("forfeitGame did not correctly advance game.turn from %d to %d. Got %d. Game:\n%v\n", 0, 2, game.turn, game.String())
	}

	// player 4 forfeits
	// game is over
	game.turn = 3
	board = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 0, 0, 0, 4, 4, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 3, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 0, 3, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	board[2][2] |= CellFlagHome
	board[boardSize-3][boardSize-3] |= CellFlagHome
	board[boardSize-3][2] |= CellFlagHome
	board[2][boardSize-3] |= CellFlagHome
	expectedBoard = GameBoard{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 3, 3, 0, 0, 0, 0, 0},
		{0, 0, 3, 0, 3, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	expectedBoard[2][2] |= CellFlagHome
	expectedBoard[boardSize-3][boardSize-3] |= CellFlagHome
	expectedBoard[boardSize-3][2] |= CellFlagHome
	expectedBoard[2][boardSize-3] |= CellFlagHome
	game.board = board
	err = game.forfeitGame(players[3])
	if err != nil {
		t.Errorf("forfeitGame returned error %v. Board:\n%s",
			err,
			board.String2D(),
		)
	}
	boardError = false
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if game.board[r][c] != expectedBoard[r][c] {
				boardError = true
				t.Errorf("forfeitGame: Bad update at game.board[%d][%d]. Expected: %v. Got: %v.",
					r, c, expectedBoard[r][c], game.board[r][c])
			}
		}
	}
	if boardError {
		t.Errorf("forfeitGame: Bad update of game.board.\nExpected:\n%s\nGot:\n%s",
			expectedBoard.String2D(),
			game.board.String2D())
	}
	if game.turn != -1 {
		t.Errorf("forfeitGame did not correctly advance game.turn from %d to %d. Got %d. Game:\n%v\n", 0, -1, game.turn, game.String())
	}
	if !game.isOver {
		t.Errorf("Expected game to be over after player's 1 and 2 forfeited. Game:\n%v\n", game.String())
	}
}

func BenchmarkGameHandleOrphanedCells(b *testing.B) {
	var boardSize int = 128
	var updates []int

	// create game
	lobbyName := "test_orphaned_cells_bench"
	_ = joinLobbyWrapper(b, lobbyName, "p1", "")
	_ = joinLobbyWrapper(b, lobbyName, "p2", "")
	game, err := createGame(activeLobbies[lobbyName], map[string]any{"size": boardSize})
	if err != nil {
		b.Fatalf("createGame failed: %v", err)
	}

	for b.Loop() {
		game.board = GameBoard{
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		}
		game.board[5][5] |= CellFlagHome
		game.board[boardSize-6][boardSize-6] |= CellFlagHome
		updates = game.handleOrphanedCells()
		if len(updates) != 12 {
			b.Fatal("Unexpected update count from handleOrphanedCells():", updates)
		}
	}
}
