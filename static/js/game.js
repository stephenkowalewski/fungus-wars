// requires: vars.js
// requires: game_piece.js


/*
 * Global variables
 */

// WebSocket related
var socket = null;
var lastMessage = 0; // timestamp
var intervalId = null;

// game board related
var boardCols = -1;
var boardRows = -1;
var board = null;
var boardIsAnimating = false;
var boardAnimationRate = 350; // ms between each update

// game piece related
var nextPiece = null;
var currentRotation = -1;
var lastPreviewIndex = -1;
var currentPreviewPieceMask = -1;
const previewPiece = 0;
const previewBite = 1;
var lastPreviewType = previewPiece;
var bite = 0; // bit mask, 0 = no bite selected

// game piece preview related
const previewMinGridSize = 4;

// vars for this player
var playerNumber = null;
var playerIndex = -1;
var playerClass = null;
var playerBites = null;
var playerRerolls = null;

// player info
var playerInfo = [];
var currentTurn = -1; // index into playerInfo

// elements set in initVars()
var gbElem = null;
var buttonPanelElem = null;
var nextTurnPreviewElem = null;
var rotatePieceBtn = null;
var skipTurnBtn = null;
var smallBiteBtn = null;
var largeBiteBtn = null;
var rerollBtn = null;

// get 1D offset mask for a piece, taking into account the board size
function pieceGetGameBoardOffsets(pmask) {
	const result = [];
	let lastOffset = 0;
	for (let r = 0; r < pieceMaskMaxLength; r++) {
		for (let c = 0; c < pieceMaskMaxLength; c++) {
			if ( pieceHas(pmask, r, c) ) {
				result.push(1);
				lastOffset = result.length;
			} else {
				result.push(0);
			}
		}
		// fill the rest of the board's row
		if ( r < pieceMaskMaxLength - 1 ) {
			for (let n = pieceMaskMaxLength; n < boardCols; n++) {
				result.push(0);
			}
		}
	}
	result.length = lastOffset;
	return result;
}

function pieceSelectNextRotation() {
	if ( currentTurn !== playerIndex ) {
		console.log("Skipped pieceSelectNextRotation(), not player's turn");
		return;
	}
	currentRotation = ( currentRotation + 1 ) % maxPieceRotations;

	// update local up next preview
	updateNextTurnPreview(
		nextTurnPreviewElem,
		currentTurn,
		nextPiece.masks[currentRotation],
		playerInfo
	);

	// update local game board
	gbPreviewUpdateBoard(
		gbElem,
		lastPreviewIndex,
		nextPiece.masks[currentRotation]
	);

	// update remote game boards
	gbPreviewSendPiece(lastPreviewIndex, nextPiece.masks[currentRotation]);

	// send button notification
	buttonDisplayNotification(rotatePieceBtn);
	sendButtonNotificationUpdate(rotatePieceBtn);

}

function initVars() {
	gbElem = document.getElementById('game-board');
	buttonPanelElem = document.getElementById("button-panel");
	nextTurnPreviewElem = document.getElementById('next-turn-preview');

	rotatePieceBtn = document.getElementById("rotatePiece");
	skipTurnBtn = document.getElementById("skipTurn");
	smallBiteBtn = document.getElementById("smallBite");
	largeBiteBtn = document.getElementById("largeBite");
	rerollBtn = document.getElementById("reroll");
}

function setHandlers() {

	gbElem.removeEventListener('click', gbClickHandler);
	gbElem.addEventListener('click', gbClickHandler);
	gbElem.removeEventListener('mouseover', gbHoverHandler);
	gbElem.addEventListener('mouseover', gbHoverHandler);
	gbElem.removeEventListener('mouseleave', gbCancelPreviewHandler);
	gbElem.addEventListener('mouseleave', gbCancelPreviewHandler);

	gbElem.removeEventListener("touchstart", gbTouchStartHandler);
	gbElem.addEventListener("touchstart", gbTouchStartHandler);
	gbElem.removeEventListener("touchmove", gbTouchMoveHandler);
	gbElem.addEventListener("touchmove", gbTouchMoveHandler);
	gbElem.removeEventListener("touchend", gbTouchEndHandler);
	gbElem.addEventListener("touchend", gbTouchEndHandler);
	gbElem.removeEventListener("touchcancel", gbCancelPreviewHandler);
	gbElem.addEventListener("touchcancel", gbCancelPreviewHandler);

	buttonPanelElem.removeEventListener('click', buttonPanelClickHandler);
	buttonPanelElem.addEventListener('click', buttonPanelClickHandler);

	document.removeEventListener('keydown', gameKeydownHandler);
	document.addEventListener('keydown', gameKeydownHandler);
}


/*
 * Helper functions for dealing with the game board
 */


// draw an empty game board
function initializeGameBoard(gbElem, cols, rows = -1) {
	if ( rows < 0 ) rows = cols; // make a square board if rows is unset
	const cellCount = cols * rows;

	// set CSS column rules
	gbElem.style.gridTemplateColumns = `repeat(${String(cols)}, 1fr)`;
	gbElem.style.gridTemplateRows = `repeat(${String(rows)}, 1fr)`;

	// draw board
	gbElem.innerHTML = "";
	for (let i = 0; i < cellCount; i++) {
		const cell = document.createElement('div');
		cell.classList.add('cell');
		gbElem.appendChild(cell);
	}
}

// update game board to apply player moves
function updateGameBoard(gbElem) {
	const cells = gbElem.getElementsByTagName('div');
	for (let i = 0; i < cells.length; i++) {
		const cell = board[ Math.floor(i/boardCols) ][ i%boardCols ]
		updateGameBoardCell(cells[i], cell);
	}
}

function updateGameBoardCell(elem, cell) {
	const owner = cell & cellMaskPlayer;
	switch (owner) {
		case 1: case 2: case 3: case 4:
			elem.className = `cell player${owner}`;
			break;
		default:
			elem.className = 'cell';
			break;
	}

	const textElem = document.createElement("span");
	textElem.classList.add("cell-content");

	if ( cell & cellFlagHome ) {
		textElem.innerText = "🏠";
	} else if ( cell & cellFlagBonusBite ) {
		textElem.innerText = "▴";
	} else if ( cell & cellFlagBonusReroll ) {
		textElem.innerText = "🎲";
	} else {
		textElem.innerText = "";
	}
	elem.replaceChildren(textElem);
}

// Update the indicator showing the turn and next piece to place.
// Updates global currentPreviewPieceMask.
function updateNextTurnPreview(previewContainer, turn, nextPieceMask, playerList) {
	let player = playerList[turn] ?? { name: "", color: "#000" };

	// Update player name
	const playerNameLabel = previewContainer.querySelector(".player-name");
	if (!playerNameLabel) {
		displayError("Something went wrong in updateNextTurnPreview()");
		return;
	}
	playerNameLabel.innerText = player.name;

	// Show piece preview
	const previewGrid = previewContainer.querySelector(".next-turn");
	if (!previewGrid) {
		displayError("Something went wrong in updateNextTurnPreview()");
		return;
	}
	drawPiecePreview(previewGrid, nextPieceMask, player.color, previewMinGridSize);

	// Update globals
	currentPreviewPieceMask = nextPieceMask;
}


// Show a preview of a piece on the game board with top left of the piece at index.
// Unsets className on all other cells.
function gbPreviewUpdateBoard(gbElem, index, nextPieceMask, className="hover-piece-local") {
	if ( index < 0 ) {
		return;
	}
	const cells = Array.from(gbElem.getElementsByTagName('div'));

	let i = 0;

	// before piece preview
	for (; i < index; i++) {
		cells[i].classList.remove(className);
	}

	// piece preview
	let pieceOffsets = pieceGetGameBoardOffsets(nextPieceMask);
	for (let stop = Math.min(i+pieceOffsets.length,cells.length); i < stop; i++) {
		if ( pieceOffsets[i-index] !== 0 ) {
			if ( i % boardCols >= (i-index) % boardCols ) { // prevent line wrap
				cells[i].classList.add(className);
			}
		} else {
			cells[i].classList.remove(className);
		}
	}

	// after piece preview
	for (let stop = cells.length; i < stop; i++) {
		cells[i].classList.remove(className);
	}
}

// Show preview of piece, replacing other classes. Only touches cells of nextPieceMask at index.
function gbPreviewPlacementUpdateBoard(gbElem, index, nextPieceMask, className="cell") {
	if ( index < 0 ) {
		return;
	}
	const cells = Array.from(gbElem.getElementsByTagName('div'));

	let i = index;

	// piece preview
	let pieceOffsets = pieceGetGameBoardOffsets(nextPieceMask);
	for (let stop = Math.min(i+pieceOffsets.length,cells.length); i < stop; i++) {
		if ( pieceOffsets[i-index] !== 0 ) {
			if ( i % boardCols >= (i-index) % boardCols ) { // prevent line wrap
				cells[i].classList = "cell";
				cells[i].classList.add(className);
			}
		}
	}

}

// show a game board_update preview_piece locally
function gbPreviewUpdateBoardPlacedPiece(gbElem, index, nextPieceMask, className) {
	const cells = Array.from(gbElem.getElementsByTagName('div'));

	let pieceOffsets = pieceGetGameBoardOffsets(nextPieceMask);
	let i = index;
	for (let stop = Math.min(i+pieceOffsets.length,cells.length); i < stop; i++) {
		if ( pieceOffsets[i-index] !== 0 ) {
			if ( i % boardCols > (i-index) % boardCols ) { // prevent line wrap
				cells[i].classList.add(className);
			}
		}
	}
}

// show a game board_update preview_bite locally
function gbPreviewUpdateBoardPlacedBite(gbElem, index, biteMask) {
	const cells = Array.from(gbElem.getElementsByTagName('div'));

	let pieceOffsets = pieceGetGameBoardOffsets(biteMask);
	let i = index;
	for (let stop = Math.min(i+pieceOffsets.length,cells.length); i < stop; i++) {
		if ( pieceOffsets[i-index] !== 0 ) {
			if ( i % boardCols > (i-index) % boardCols ) { // prevent line wrap
				cells[i].classList.remove("player1", "player2", "player3", "player4");
			}
		}
	}

}

function updateGameScores(scores) {
	for ( let i=0; i<scores.length; i++ ) {
		const playerScoreElem = document.getElementById(`player${i+1}-score`);
		if ( playerScoreElem === null ) {
			console.log(`updateGameScore(${i+1}): elem not found. Invalid player number?`);
			return;
		}
		playerScoreElem.innerText = scores[i];
	}
}

function updateBites(bites) {
	for ( let i=0; i<bites.length; i++ ) {
		const playerBitesElem = document.getElementById(`player${i+1}-bites`);
		if ( playerBitesElem === null ) {
			console.log(`updateBites(${i+1}): elem not found. Invalid player number?`);
			return;
		}
		playerBitesElem.innerText = bites[i];
	}
}

function updateRerolls(rerolls) {
	for ( let i=0; i<rerolls.length; i++ ) {
		const playerRerollsElem = document.getElementById(`player${i+1}-rerolls`);
		if ( playerRerollsElem === null ) {
			console.log(`updateRerolls(${i+1}): elem not found. Invalid player number?`);
			return;
		}
		playerRerollsElem.innerText = rerolls[i];
	}
}

// returns the player number of the owner or null for a cell element
function gbGetCellOwner(elem) {
	for (const className of elem.classList) {
		const match = className.match(/player(\d)/);
		if ( match ) {
			return match[1];
		}
	}
	return null;
}

async function sendSkipTurn() {
	// allow the button notification to send first
	await new Promise(r => setTimeout(r, 0));

	const gameUpdate = {
		type: "game_update",
		payload: {
			action: "skip_turn",
		}
	};
	console.log(gameUpdate);
	socket.send(JSON.stringify(gameUpdate));
}

function sendReroll() {
	const gameUpdate = {
		type: "game_update",
		payload: {
			action: "reroll",
		}
	};
	console.log(gameUpdate);
	socket.send(JSON.stringify(gameUpdate));
}

// Show an indicator on the info of the player whose turn it is.
// Remove the indicator from all other players.
// Passing an invalid turn like -1 unselects all.
function updatePlayerTurnIndicator(turn) {
	for (let i=0; i<playerInfo.length; i++) {
		const pInfoElem = document.getElementById(`player${i+1}-info`);
		if (i === turn) {
			pInfoElem.classList.add("player-turn");
		} else {
			pInfoElem.classList.remove("player-turn");
		}
	}
}

function advanceBite() {
	const biteNames = Object.keys(biteNameToMask);
	for (let i = 0; i < biteNames.length; i++) {
		if (biteNameToMask[biteNames[i]] === bite) {
			let nextKey = biteNames[(i + 1) % biteNames.length];
			bite = biteNameToMask[nextKey];
			activateBiteButton(nextKey);
			updateBiteCostPreview(nextKey);
			return;
		}
	}
}

// toggles between no bite and the bite defined by elem
function toggleBite(elem) {
	const bmask = biteNameToMask[elem.id];
	if ( bite === bmask ) {
		// clicking the already selected bite toggles it off
		bite = 0;
		activateBiteButton("noBite");
		updateBiteCostPreview("noBite");
	} else {
		bite = bmask;
		activateBiteButton(elem.id);
		updateBiteCostPreview(elem.id);
	}
}

// marks a bite button as being selected
function activateBiteButton(buttonId) {
	const inactive = ["smallBite","largeBite"].filter(v => v !== buttonId);
	for (let id of inactive) {
		document.getElementById(id).classList.remove("active");
	}
	const active = ["smallBite","largeBite"].find(v => v === buttonId);
	if ( active !== undefined ) {
		document.getElementById(buttonId).classList.add("active");
	}
}

function updateBiteCostPreview(biteName) {
	const biteChangeElem = document.getElementById(`player${playerNumber}-bite-change-indicator`);
	const cost = biteNameToCost[biteName];
	if ( biteChangeElem === null || cost === undefined ) {
		console.log("updateBiteCostPreview() is missing expected data");
		return;
	}

	if ( cost === 0 ) {
		biteChangeElem.innerText = "";
	} else {
		biteChangeElem.innerText = `(-${cost})`;
	}

	if ( playerBites >= cost ) {
		biteChangeElem.style.color = "green";
	} else {
		biteChangeElem.style.color = "red";
	}
}

function buttonPanelClickHandler(event) {
	if (event.target.classList.contains('sendNotification')) {
		buttonDisplayNotification(event.target);
		sendButtonNotificationUpdate(this);
	} else if (event.target.classList.contains('sendState')) {
		sendButtonStateUpdate(this);
	}
}

function sendButtonStateUpdate(elem) {
		const sendStateButtons = Array.from(elem.querySelectorAll("button.sendState"));
		const buttonUpdate = {
			type: "button_update",
			payload: {
				active: sendStateButtons.filter(e => e.classList.contains("active")).map(e => e.id),
				inactive: sendStateButtons.filter(e => !e.classList.contains("active")).map(e => e.id),
			}
		};
		console.log(buttonUpdate);
		socket.send(JSON.stringify(buttonUpdate));
}

function sendButtonNotificationUpdate(elem) {
		const sendNotificationButtons = elem.tagName === "BUTTON" ?
			[ elem ] : Array.from(elem.querySelectorAll("button.sendNotification"));
		const buttonUpdate = {
			type: "button_update",
			payload: {
				notify: sendNotificationButtons.filter(e => e.classList.contains("notify")).map(e => e.id)
			}
		};
		console.log(buttonUpdate);
		socket.send(JSON.stringify(buttonUpdate));
}

function restartGame() {
	clearMessages();
	bite = 0;
	const update = {
		type: "game_update",
		payload: {
			action: "reset_game",
		}
	};
	//console.log(update);
	socket.send(JSON.stringify(update));
}

function forfeitGame() {
	clearMessages();
	const update = {
		type: "game_update",
		payload: {
			action: "forfeit_game",
		}
	};
	//console.log(update);
	socket.send(JSON.stringify(update));
}

// Add piece or bite to the local game board and then notify the server
function gbPlacePiece(gbElem, index) {
	if ( bite === 0 ) {
		// update local board view
		gbPreviewUpdateBoardPlacedPiece(gbElem, index, nextPiece.masks[currentRotation], playerClass);

		// notify the server
		const boardUpdate = {
			type: "board_update",
			payload: {
				action: "place_piece",
				index: index,
				mask: nextPiece.masks[currentRotation]
			}
		};
		console.log(boardUpdate);
		socket.send(JSON.stringify(boardUpdate));
	} else {
		// update local board view
		gbPreviewUpdateBoardPlacedBite(gbElem, index, bite);

		// notify the server
		const boardUpdate = {
			type: "board_update",
			payload: {
				action: "place_bite",
				index: index,
				mask: bite
			}
		};
		console.log(boardUpdate);
		socket.send(JSON.stringify(boardUpdate));
		updateBiteCostPreview("noBite");
	}
}

function gbClickHandler(event) {
	if (!event.target.classList.contains('cell')) {
		return;
	}

	clearMessages();

	const cells = Array.from(this.getElementsByTagName('div'));
	const index = cells.indexOf(event.target);
	console.log(`Clicked div at index: ${index}`);

	gbPlacePiece(this, index);
}

// When a player moves their pointer over the game board, show a preview of the move
function gbHoverHandler(event) {
	if (!event.target.classList.contains('cell')) {
		return;
	}
	if ( currentTurn !== playerIndex ) {
		return;
	}

	const cells = Array.from(this.getElementsByTagName('div'));
	lastPreviewIndex = cells.indexOf(event.target);

	gbPreviewShow(this);
}

// When a player touches over the game board, show a preview of the move
function gbTouchStartHandler(event) {
	event.preventDefault();

	if (!event.target.classList.contains('cell')) {
		return;
	}
	if ( currentTurn !== playerIndex ) {
		return;
	}

	const cells = Array.from(this.getElementsByTagName('div'));
	lastPreviewIndex = cells.indexOf(event.target);
	gbPreviewShow(this);
}

function gbTouchMoveHandler(event) {
	if ( currentTurn !== playerIndex ) {
		return;
	}

  const t = event.touches[0];
  const target = document.elementFromPoint(t.clientX, t.clientY);

	const cells = Array.from(this.getElementsByTagName('div'));
	const targetIndex = cells.indexOf(target);
	if ( targetIndex === lastPreviewIndex ) {
		return;
	}
	lastPreviewIndex = targetIndex;
	if ( targetIndex < 0 ) {
		gbClearHoverClasses(this);
		gbPreviewSendClear();
	} else {
		gbPreviewShow(this);
	}
}

function gbTouchEndHandler(event) {
	event.preventDefault();
	if ( currentTurn !== playerIndex ) {
		return;
	}

  const t = event.changedTouches[0];
  const target = document.elementFromPoint(t.clientX, t.clientY);

	if (!target.classList.contains('cell')) {
		return;
	}
  if ( !this.contains(target) ) {
		return;
	}
	clearMessages();

	const cells = Array.from(this.getElementsByTagName('div'));
	const index = cells.indexOf(target);
	console.log(`Selected div at index: ${index} via touchend`);

	gbPlacePiece(this, index);
}

// displays a preview locally and notifies the server so that other players see the update
function gbPreviewShow(elem) {
	let thisPreviewType = bite ? previewBite : previewPiece;
	if ( thisPreviewType !== lastPreviewType ) {
		// clear the board
		gbClearHoverClasses(elem);
		// send to server
		gbPreviewSendClear();
	}
	lastPreviewType = thisPreviewType;

	if ( bite !== 0 ) {
		// preview bite piece
		gbPreviewUpdateBoard(elem, lastPreviewIndex, bite, "hover-bite-local");
		// send to server
		gbPreviewSendBite(lastPreviewIndex, bite);
	} else {
		// preview regular game piece
		gbPreviewUpdateBoard(elem, lastPreviewIndex, nextPiece.masks[currentRotation]);
		// send to server
		gbPreviewSendPiece(lastPreviewIndex, nextPiece.masks[currentRotation]);
	}
}

function gbPreviewSendBite(index, mask) {
		const boardUpdatePreview = {
			type: "board_update_preview",
			payload: {
				action: "preview_bite",
				index: index,
				mask: mask
			}
		};
		//console.log(boardUpdatePreview);
		socket.send(JSON.stringify(boardUpdatePreview));
}

function gbPreviewSendPiece(index, mask) {
		const boardUpdatePreview = {
			type: "board_update_preview",
			payload: {
				action: "preview_piece",
				index: index,
				mask: mask
			}
		};
		//console.log(boardUpdatePreview);
		socket.send(JSON.stringify(boardUpdatePreview));
}

function gbPreviewSendClear() {
		const boardUpdatePreview = {
			type: "board_update_preview",
			payload: {
				action: "clear"
			}
		};
		//console.log(boardUpdatePreview);
		socket.send(JSON.stringify(boardUpdatePreview));
}

function gbCancelPreviewHandler(event) {
	if ( currentTurn !== playerIndex ) {
		return;
	}

	lastPreviewIndex = -1;
	gbClearHoverClasses(this);
	gbPreviewSendClear();
}

function gbClearHoverClasses(elem) {
	const cells = Array.from(elem.getElementsByTagName('div'));

	for (let i = 0, stop=cells.length; i < stop; i++) {
		cells[i].classList.remove("hover-piece-local", "hover-bite-local", "hover-piece-remote", "hover-bite-remote");
	}
}

// gets the index to display the next piece preview when an arrow key is pressed
// does not yet account for piece size
function getPreviewIndexForArrowKey(key) {
	if ( lastPreviewIndex < 0 ) {
		return Math.floor(boardCols * boardRows / 2);
	} else if ( key == "ArrowUp" && lastPreviewIndex >= boardCols ) {
		return lastPreviewIndex - boardCols;
	} else if ( key == "ArrowDown" && lastPreviewIndex < boardCols * boardRows - boardCols ) {
		return lastPreviewIndex + boardCols;
	} else if ( key == "ArrowLeft" && lastPreviewIndex % boardCols != 0 ) {
		return lastPreviewIndex - 1;
	} else if ( key == "ArrowRight" && lastPreviewIndex % boardCols != boardCols - 1 ) {
		return lastPreviewIndex + 1;
	} else {
		return lastPreviewIndex;
	}
}

function gameKeydownHandler(event) {
	//console.log(`gameKeydownHandler ${event.code}: "${event.key}"`);

  if ( currentTurn === playerIndex ) {
		switch (event.key) {
			case "ArrowUp": // move piece preview
			case "ArrowDown":
			case "ArrowLeft":
			case "ArrowRight":
				event.preventDefault();
				const newIndex = getPreviewIndexForArrowKey(event.key);
				const cell = gbElem.getElementsByTagName('div').item(newIndex);
				lastPreviewIndex = newIndex;
				gbPreviewShow(gbElem);
				break;
			case "Enter": // place piece
				if ( lastPreviewIndex >= 0 ) {
					event.preventDefault();
					clearMessages();
					gbPlacePiece(gbElem, lastPreviewIndex);
				} else {
					displayWarning("Could not place piece. No piece selected.");
				}
				break;
			case " ": // rotate piece
				event.preventDefault();
				if ( bite === 0 ) {
					pieceSelectNextRotation();
				}
				break;
			case "b": // bite
				advanceBite();
				gbClearHoverClasses(gbElem);
				gbPreviewShow(gbElem);
				sendButtonStateUpdate(buttonPanelElem);
				if ( bite === 0 ) {
					gbPreviewSendPiece(lastPreviewIndex, nextPiece.masks[currentRotation]);
				} else {
					gbPreviewSendBite(lastPreviewIndex, bite);
				}
				break;
			case "r": // reroll
				buttonDisplayNotification(rerollBtn);
				sendButtonNotificationUpdate(rerollBtn);
				sendReroll();
				break;
			case "s": // skip turn
				buttonDisplayNotification(skipTurnBtn);
				sendButtonNotificationUpdate(skipTurnBtn);
				sendSkipTurn();
				break;
		}
	}
}




function displayError(text, elemId = "game_errors") {
	document.getElementById(elemId).innerHTML =
		`<p class="error">${text}</p><br>`;
}

function displayWarning(text, elemId = "game_errors") {
	document.getElementById(elemId).innerHTML =
		`<p class="warning">${text}</p><br>`;
}

function offerReconnect() {
	document.getElementById("reconnect").innerHTML =
		`<button onclick="gameWsConnect()">Re-establish connection?</button>`;
}

function clearMessages() {
	document.getElementById("game_errors").innerHTML = "";
	document.getElementById("idle_warning").innerHTML = "";
	document.getElementById("reconnect").innerHTML = "";
}

function gameWsConnect() {
	if ( socket !== null ) {
		clearMessages();
		try { socket.close(); } catch { console.log("socket.close() failed"); }
	}
	const wsConnectSocket = new WebSocket("/game/ws");
	socket = wsConnectSocket;

	wsConnectSocket.addEventListener('open', () => {
		console.log('WebSocket connection established.');
		// You can send an initial message here if needed
		// socket.send(JSON.stringify({ type: 'hello' }));
		watchForIdleTimeout("start");
	});

	wsConnectSocket.addEventListener('message', (event) => {
		gameWsDispatch(wsConnectSocket, event);
	});

	wsConnectSocket.addEventListener('error', (err) => {
		console.error('WebSocket error:', err);
		displayError("Game connection had unexpected error.")
		offerReconnect();
	});

	wsConnectSocket.addEventListener('close', () => {
		console.log('WebSocket connection closed.');
		// on re-connect wsConnectSocket !== global socket
		if ( wsConnectSocket === socket ) {
			watchForIdleTimeout("stop");
			displayError("Game connection closed.")
			offerReconnect();
		}
	});
}

function checkIdleTimeout(timeoutMs) {
	const delta = Date.now() - lastMessage;
	if ( delta > timeoutMs ) {
		displayWarning(`No game data in ${delta/1000} seconds`, "idle_warning");
		offerReconnect();
	} else {
		document.getElementById("idle_warning").innerHTML = "";
	}
}

function watchForIdleTimeout(mode, timeoutMs = 6000) {
	if ( mode === "start" ) {
		intervalId = setInterval(() => { checkIdleTimeout(timeoutMs); }, timeoutMs);
	} else {
		try {
			clearInterval(intervalId);
		} catch (err) {
			console.log(gameWsWatchForIdleTimeout, err);
		}
	}
}

function gameWsDispatch(socket, event) {
	lastMessage = Date.now();
	try {
		const data = JSON.parse(event.data);
		if (data.type === 'ping') {
			socket.send(JSON.stringify({ type: 'pong' }));
		} else if (data.type === 'board_info_preview') {
			gameWsHandleMsgBoardUpdatePreview(socket, data);
		} else if (data.type === 'button_info') {
			gameWsHandleMsgButtonInfo(socket, data);
		} else if (data.type === 'game_info') {
			gameWsHandleMsgGameInfo(socket, data);
		} else if (data.type === 'error') {
			displayError(data.payload.message);
		} else if (data.type === 'player_info') {
			gameWsHandleMsgPlayerInfo(socket, data);
		} else {
			console.warn('Unknown message format:', data);
		}
	} catch (err) {
		console.error('Failed to parse message:', event.data, err);
		displayError('Failed to parse message from server');
		offerReconnect();
	}
}

// updates globals: playerIndex, playerNumber, playerClass, playerInfo
function gameWsHandleMsgPlayerInfo(_socket, data) {
	console.log(data);
	if (
		!data.payload ||
		!data.payload.players?.length ||
		data.payload.identity == null ||
		data.payload.identity < 0 ||
		!data.payload.win_loss_draw_record?.length
	) {
		throw new Error(`Missing expected payload for message type ${data.type}`);
	}


	// handle global vars
	playerIndex = data.payload.identity;
	playerNumber = playerIndex + 1;
	playerClass = `player${playerNumber}`;
	playerInfo = data.payload.players;


	// handle player list
	const styleId = `player_info_style`;
	let style = document.getElementById(styleId);
	if (!style) {
		style = document.createElement('style');
		style.id = styleId;
		document.head.appendChild(style);
	}
	style.textContent = "";

	let i=0;
	for (; i<data.payload.players.length; i++) {
		const name = data.payload.players[i].name;
		const color = data.payload.players[i].color;
		const playerId = `player${i+1}`;
		document.getElementById(`${playerId}-name`).innerText = name;
		document.getElementById(`${playerId}-color`).style.backgroundColor = color;
		style.textContent += `.cell.${playerId} { background-color: ${color}; }\n`;
	}
	for (; i<maxPlayers; i++) {
		document.getElementById(`player${i+1}-info`).style.display = "none";
	}

	// update win/loss/draw records
	for (let i=0; i<data.payload.win_loss_draw_record.length; i++) {
		const record = data.payload.win_loss_draw_record[i];
		document.getElementById(`player${i+1}-wins`).innerText = record.W;
		document.getElementById(`player${i+1}-losses`).innerText = record.L;
		document.getElementById(`player${i+1}-draws`).innerText = record.D;
	}
}

// updates buttons to show an indicator when the active remote player has them selected
function gameWsHandleMsgButtonInfo(_socket, data) {
	console.log(data);
	if ( !data.payload) {
		throw new Error(`Missing expected payload for message type ${data.type}`);
	}

	if ( currentTurn === playerIndex ) {
		return;
	}

	if ( data.payload.inactive?.length !== undefined ) {
		for (let i = 0; i < data.payload.inactive.length; i++) {
			document.getElementById(data.payload.inactive[i]).classList.remove("active");
		}
	}

	if ( data.payload.active?.length !== undefined ) {
		for (let i = 0; i < data.payload.active.length; i++) {
			document.getElementById(data.payload.active[i]).classList.add("active");
		}
	}

	if ( data.payload.notify?.length !== undefined ) {
		for (let i = 0; i < data.payload.notify.length; i++) {
			buttonDisplayNotification(document.getElementById(data.payload.notify[i]));
		}
	}
}

// updates globals: board, boardCols, boardRows, currentTurn, currentRotation, nextPiece, playerBites, playerRerolls
function gameWsHandleMsgGameInfo(_socket, data) {
	console.log(data);
	if (
		!data.payload ||
		!data.payload.board?.length ||
		!data.payload.next_piece?.masks?.length ||
		data.payload.turn === null ||
		!data.payload.scores?.length ||
		!data.payload.bites?.length ||
		!data.payload.rerolls?.length ||
		data.payload.game_over === null
	) {
		throw new Error(`Missing expected payload for message type ${data.type}`);
	}

	updateGameScores(data.payload.scores);
	updateBites(data.payload.bites);
	playerBites = data.payload.bites[playerIndex];
	updateRerolls(data.payload.rerolls);
	playerRerolls = data.payload.rerolls[playerIndex];

	if ( data.payload.game_over ) {
		clearMessages();
		bite = 0;
		let winner = "undefined";
		for ( let i=0; i<data.payload.scores.length; i++ ) {
			if ( data.payload.scores[i] > 0 ) {
				winner = playerInfo[i].name;
				break;
			}
		}
		document.getElementById("game_over").innerText = `${winner} wins!`;
	} else {
		document.getElementById("game_over").innerText = "";
	}

	const cols = data.payload.board.length;
	const rows = data.payload.board[0].length;
	if ( cols !== boardCols || rows !== boardRows ) {
		boardCols = cols;
		boardRows = rows;
		initializeGameBoard(gbElem, boardCols, boardRows);
	}
	board = data.payload.board;

	if ( data.payload.board_updates_to_animate?.length ) {
		(async () => {
			await animateBoardUpdates(data.payload.board_updates_to_animate);
			updateGameBoard(gbElem, board);
		})();
	} else {
		boardIsAnimating = false;
		updateGameBoard(gbElem, board);
	}

	nextPiece = data.payload.next_piece;
	if (currentTurn != data.payload.turn ) {
		// conditional prevents invalid moves from resetting the selected rotation or bite
		currentRotation = 0;
		bite = 0;
		currentTurn = data.payload.turn;
	}

	// enable/disable buttons based on the current turn
	const disabled = currentTurn !== playerIndex;
	rotatePieceBtn.disabled = disabled;
	skipTurnBtn.disabled = disabled;
	smallBiteBtn.disabled = disabled || playerBites < biteNameToCost["smallBite"];
	largeBiteBtn.disabled = disabled || playerBites < biteNameToCost["largeBite"];
	rerollBtn.disabled = disabled || playerRerolls < 1;

	activateBiteButton(biteMaskToName[bite]);
	updateBiteCostPreview(biteMaskToName[bite]);

	updatePlayerTurnIndicator(currentTurn);

	updateNextTurnPreview(
		nextTurnPreviewElem,
		currentTurn,
		nextPiece.masks[currentRotation],
		playerInfo
	);
}

function gameWsHandleMsgBoardUpdatePreview(_socket, data) {
	//console.log(data);
	if (
		!data.payload ||
		data.payload.action === null ||
		data.payload.index === null ||
		data.payload.mask === null
	) {
		throw new Error(`Missing expected payload for message type ${data.type}`);
	}

	switch (data.payload.action) {
		case "preview_piece":
			gbPreviewUpdateBoard(
				gbElem,
				data.payload.index,
				data.payload.mask,
				"hover-piece-remote"
			);
			break;
		case "preview_bite":
			gbPreviewUpdateBoard(
				gbElem,
				data.payload.index,
				data.payload.mask,
				"hover-bite-remote"
			);
			break;
		case "place_piece":
			gbPreviewPlacementUpdateBoard(
				gbElem,
				data.payload.index,
				data.payload.mask,
				`player${data.payload.owner}`,
			);
			break;
		case "place_bite":
			gbPreviewPlacementUpdateBoard(
				gbElem,
				data.payload.index,
				data.payload.mask,
				"hover-bite-remote"
			);
			break;
		case "clear":
			gbClearHoverClasses(gbElem);
			break;
		default:
			console.log(`Unknown action ${data.payload.action} for message type ${data.type}`);
			return;
	}

	if ( data.payload.mask !== currentPreviewPieceMask && data.payload.action.match(/_piece$/) ) {
		updateNextTurnPreview(
			nextTurnPreviewElem,
			currentTurn,
			data.payload.mask,
			playerInfo
		);
	}
}

async function animateBoardUpdates(indices) {
	const cells = gbElem.getElementsByTagName('div');
	boardIsAnimating = true;

	while (indices.length && boardIsAnimating) {
		await new Promise(r => setTimeout(r, boardAnimationRate));

		const index = indices.shift();
		console.log(`animating ${index}`);

		const elem = cells[index];
		const cell = board[ Math.floor(index/boardCols) ][ index%boardCols ]
		updateGameBoardCell(elem, cell);
	}
	boardIsAnimating = false;
}

// vim: ts=2
