//go:build ignore
// +build ignore

// This program generates static/js/vars.js.
// Run it via `go generate`.

package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

var jsVarsPath string = "static/js/vars.js"

func main() {
	f, err := os.Create(jsVarsPath)
	if err != nil {
		panic(err)
	}

	// Write out JS variable assignments
	fmt.Fprintln(f, "// Auto-generated", time.Now().UTC().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintln(f)
	fmt.Fprintf(f, "const maxPlayers = %d;\n", maxPlayers)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "const cellFlagHome = 0x%04x;\n", CellFlagHome)
	fmt.Fprintf(f, "const cellFlagBonusBite = 0x%04x;\n", CellFlagBonusBite)
	fmt.Fprintf(f, "const cellFlagBonusReroll = 0x%04x;\n", CellFlagBonusReroll)
	fmt.Fprintf(f, "const cellMaskPlayer = 0x%04x;\n", CellMaskPlayer)
	fmt.Fprintf(f, "const cellMaskFlags = 0x%04x;\n", CellMaskFlags)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "const pieceMaskMaxLength = %d;\n", pieceMaskMaxLength)
	fmt.Fprintf(f, "const pieceMaskSectionMask = %s;\n", pieceMaskSectionMask)
	fmt.Fprintf(f, "const pieceMaskFirstRowMask = %s;\n", pieceMaskFirstRowMask)
	fmt.Fprintf(f, "const pieceMaskFirstColumnMask = %s;\n", pieceMaskFirstColumnMask)
	fmt.Fprintf(f, "const maxPieceRotations = %d;\n", maxPieceRotations)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "const biteNameToMask = {\n")
	fmt.Fprintf(f, "  \"noBite\": %d,\n", biteNone)
	fmt.Fprintf(f, "  \"smallBite\": %s,\n", biteSmall)
	fmt.Fprintf(f, "  \"largeBite\": %s\n", biteLarge)
	fmt.Fprintln(f, "};")
	fmt.Fprintf(f, "const biteMaskToName = {\n")
	fmt.Fprintf(f, "  %d: \"noBite\",\n", biteNone)
	fmt.Fprintf(f, "  %s: \"smallBite\",\n", biteSmall)
	fmt.Fprintf(f, "  %s: \"largeBite\"\n", biteLarge)
	fmt.Fprintln(f, "};")
	fmt.Fprintf(f, "const biteNameToCost = {\n")
	fmt.Fprintf(f, "  \"noBite\": %d,\n", biteNone.CalcBiteCost())
	fmt.Fprintf(f, "  \"smallBite\": %d,\n", biteSmall.CalcBiteCost())
	fmt.Fprintf(f, "  \"largeBite\": %d\n", biteLarge.CalcBiteCost())
	fmt.Fprintln(f, "};")
	fmt.Fprintln(f)
	fmt.Fprintf(f, "const gbDefaultSize = %d;\n", gbDefaultSize)
	fmt.Fprintf(f, "const gbMinSize = %d;\n", max(gbMinSize, 10))
	fmt.Fprintf(f, "const gbMaxSize = %d;\n", min(gbMaxSize, 50))
	fmt.Fprintf(f, "const gbDefaultRandomizeStartPos = %t;\n", gbDefaultRandomizeStartPos)
	fmt.Fprintf(f, "const gbDefaultStartBites = %d;\n", gbDefaultStartBites)
	fmt.Fprintf(f, "const gbDefaultStartRerolls = %d;\n", gbDefaultStartRerolls)
	fmt.Fprintf(f, "const gbDefaultHasBonusBiteCells = %t;\n", gbDefaultHasBonusBiteCells)
	fmt.Fprintf(f, "const gbDefaultBonusRerollCells = %d;\n", gbDefaultBonusRerollCells)
	fmt.Fprintf(f, "const gbDefaultNewBiteFreqFactor = %f;\n", gbDefaultNewBiteFreqFactor)

	fmt.Fprintln(f, "const gbDefaultPieces = [")
	for i, piece := range gbDefaultPieces {
		if i > 0 {
			fmt.Fprint(f, ",\n")
		}
		fmt.Fprintf(f, `  {"mask": %v, "weight": %f}`, piece.Masks[0], piece.Weight)
	}
	fmt.Fprintln(f, "\n];")

	fmt.Fprintln(f)
	fmt.Fprintf(f, "const gameCaptureModes = {\n")
	fmt.Fprintf(f, "  \"From piece\": %d,\n", gameModeCaptureFromPiece)
	fmt.Fprintf(f, "  \"Whole board current player only\": %d,\n", gameModeCaptureAnywhereCurrentPlayer)
	fmt.Fprintf(f, "  \"Whole board all players\": %d\n", gameModeCaptureAnywhereAllPlayers)
	fmt.Fprintln(f, "};")
	fmt.Fprintln(f)

	f.Close()
	log.Println("Wrote:", jsVarsPath)
}
