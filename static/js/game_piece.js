// requires: vars.js

/*
 * Helper functions for dealing with game pieces
 */

// Returns the bitmask at position (row, column).
// Top left is (0,0).
function pieceMaskAt(r, c) {
	return 1 << ((pieceMaskMaxLength-1-r)*pieceMaskMaxLength + (pieceMaskMaxLength - 1 - c))
}

// Check if a given PieceMask has a bit set at (row, column).
// Top left is (0,0).
function pieceHas(p, r, c) {
	if (r >= pieceMaskMaxLength || c >= pieceMaskMaxLength) {
		return false;
	}
	return (p & pieceMaskAt(r, c)) !== 0;
}

function pieceString2D(pmask) {
	let s = "";

	for (let r = 0; r < pieceMaskMaxLength; r++) {
		if (r > 0) s += "\n";

		for (let c = 0; c < pieceMaskMaxLength; c++) {
			if (c > 0) s += " ";

			if (pieceHas(pmask, r, c)) {
				s += "1";
			} else {
				s += "0";
			}
		}
	}

	return s;
}


// get the size of a piece mask
function pieceGetSize(pmask) {
	var r = 0;
	for (let i = 0; i < pieceMaskMaxLength; i++) {
		if ( (pmask & (pieceMaskFirstRowMask >>> (pieceMaskMaxLength*i))) !== 0 ) {
			r = i+1;
		}
	}

	var c = 0;
	for (let i = 0; i < pieceMaskMaxLength; i++) {
		if ( (pmask & (pieceMaskFirstColumnMask >>> i)) !== 0 ) {
			c = i+1;
		}
	}

	return [r, c];
}

// draws pieceMask in div container
function drawPiecePreview(container, pieceMask, color, minGridSize) {
	// Remove any existing preview
	container.innerHTML = "";

	// Get piece size
	const [pieceRows, pieceCols] = pieceGetSize(pieceMask);
	const gridSize = Math.max(pieceRows, pieceCols, minGridSize);

	// Create preview grid
	container.style.gridTemplateColumns = `repeat(${gridSize}, 1fr)`;
	container.style.gridTemplateRows = `repeat(${gridSize}, 1fr)`;

	for (let r = 0; r < gridSize; r++) {
		for (let c = 0; c < gridSize; c++) {
			const gridCell = document.createElement("div");

			if (pieceHas(pieceMask, r, c)) {
				gridCell.style.backgroundColor = color;
				gridCell.classList.add("occupied");
			} else {
				gridCell.classList.add("spacer");
			}

			if (r >= pieceRows || c >= pieceCols) {
				gridCell.classList.add("spacer");
			}

			container.appendChild(gridCell);
		}
	}

}

// vim: ts=2
