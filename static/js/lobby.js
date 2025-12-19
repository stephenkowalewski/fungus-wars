// requires: vars.js
// requires: common.js
// requires: game_piece.js

const piecePreviewColor = "green";

const idToDefaultValue = {
	"board-size-slider":             gbDefaultSize,
	"rand-start-pos-checkbox":       gbDefaultRandomizeStartPos,
	"starting-bites-slider":         gbDefaultStartBites,
	"bonus-bite-cells-checkbox":     gbDefaultHasBonusBiteCells,
	"starting-rerolls-slider":       gbDefaultStartRerolls,
	"bonus-reroll-cells-slider":     gbDefaultBonusRerollCells,
	"new-bite-freq-factor-slider":   gbDefaultNewBiteFreqFactor,
	"capture-mode-choice":           "",
	"use-custom-piece-set-checkbox": false,
};

const idToJoinGameArg = {
	"board-size-slider":           "size",
	"rand-start-pos-checkbox":     "randomize_start_positions",
	"starting-bites-slider":       "starting_bites",
	"bonus-bite-cells-checkbox":   "has_bonus_bite_cells",
	"starting-rerolls-slider":     "starting_rerolls",
	"bonus-reroll-cells-slider":   "bonus_reroll_cells",
	"new-bite-freq-factor-slider": "new_bites_freq_factor",
	"capture-mode-choice":         "capture_mode"
};

const idToSelectOptionList = {
	"capture-mode-choice": gameCaptureModes
}

let idToSavedValue = {};
function loadSavedValuesFromLocalStorage() {
	const jsonString = localStorage.getItem("lastGameArgs");
	if (!jsonString) {
		return;
	}

	console.log(`lastGameArgs (from localStorage): ${jsonString}`);
	const lsObj = JSON.parse(jsonString);
	for (const [id, arg] of Object.entries(lsObj)) {
		idToSavedValue[id] = arg;
	}
}
loadSavedValuesFromLocalStorage();

// Set the value of a slider based on localStorage and defaults.
// Add a listener that displays the value.
function setupSlider(outputId, sliderId) {
	const output = document.getElementById(outputId);
	const slider = document.getElementById(sliderId);
	const val = idToSavedValue[sliderId] ?? idToDefaultValue[sliderId];

	output.textContent = val;
	slider.value = val;

	slider.addEventListener("input", () => {
		output.textContent = slider.value;
	});
}

// Set the value of a checkbox based on localStorage and defaults.
// Add a listener that displays the value.
function setupCheckbox(checkboxId) {
	const checkbox = document.getElementById(checkboxId);
	const val = idToSavedValue[checkboxId] ?? idToDefaultValue[checkboxId];

	checkbox.checked = val;
}

// Set the value of a select element based on localStorage and defaults.
// Add a listener that displays the value.
function setupSelect(selectId) {
	const select = document.getElementById(selectId);
	const options = idToSelectOptionList[selectId];

	for (option in options) {
		const newOption = document.createElement("option");
		newOption.text = option;
		newOption.value = options[option];
		select.appendChild(newOption);
	}

	const savedValue = idToSavedValue[selectId];
	if ( savedValue !== undefined ) {
		select.value = savedValue;
	}
}


function setupGameOptionInputs() {
	// game board size
	setupSlider("board-size-rows", "board-size-slider");
	setupSlider("board-size-cols", "board-size-slider");

	const boardSizeSlider = document.getElementById("board-size-slider");
	boardSizeSlider.min = gbMinSize;
	boardSizeSlider.max = gbMaxSize;

	// randomize start positions
	setupCheckbox("rand-start-pos-checkbox");

	// starting bites
	setupSlider("starting-bites", "starting-bites-slider");

	// bonus bite cells
	setupCheckbox("bonus-bite-cells-checkbox");

	// starting rerolls
	setupSlider("starting-rerolls", "starting-rerolls-slider");

	// bonus reroll cells
	setupSlider("bonus-reroll-cells", "bonus-reroll-cells-slider");

	// new bite frequency adjustment
	setupSlider("new-bite-freq-factor", "new-bite-freq-factor-slider");

	// game capture mode
	setupSelect("capture-mode-choice");

	// use custom piece set
	setupCheckbox("use-custom-piece-set-checkbox");
	const customPieceCheckbox = document.getElementById("use-custom-piece-set-checkbox");
	customPieceCheckbox.addEventListener("input", () => {
		toggleShowHideCustomPieceOptions(event.target);
	});

	// generate table for custom pieces and maybe display it
	if ( idToSavedValue.pieces?.data?.length ) {
		generateCustomPieceTable("custom-pieces-table", idToSavedValue.pieces.data);
	} else {
		generateCustomPieceTable("custom-pieces-table", gbDefaultPieces);
	}
	toggleShowHideCustomPieceOptions(customPieceCheckbox);
}

function generateCustomPieceTable(id, pieceList) {
	const table = document.getElementById(id);
	table.replaceChildren();

	const thead = document.createElement("thead");
	thead.innerHTML = `
		<th></th>
		<th>Piece</th>
		<th></th>
		<th title="Hint: Set weight to zero to disable a piece">Weight</th>
`;
	table.appendChild(thead);

	const tbody = document.createElement("tbody");
	table.appendChild(tbody);

	pieceList.forEach(piece => {
		addCustomPieceRowToTable(table, piece);
	});
}

// add a piece customization row to the table containing el
function addCustomPieceRowToTable(el, piece={"mask":0,"weight":0} ) {
	const table = el.closest("table");
	const tbody = table.tBodies[0];
	const tr = document.createElement("tr");
	let td;

	// gap
	td = document.createElement("td");
	td.classList.add("column_gap_large");
	tr.appendChild(td);

	// hidden mask input and preview
	td = document.createElement("td");

	const hiddenInput = document.createElement("input");
	hiddenInput.type = "hidden";
	hiddenInput.value = piece.mask;
	hiddenInput.classList.add("piece-mask-input");
	td.appendChild(hiddenInput);

	const maskPreview = document.createElement("div");
	maskPreview.classList.add("custom-piece-preview");
	drawPiecePreview(maskPreview, piece.mask, piecePreviewColor, pieceMaskMaxLength);
	maskPreview.addEventListener('click', modifyPieceHandler);
	td.appendChild(maskPreview);

	tr.appendChild(td);

	// gap
	td = document.createElement("td");
	td.classList.add("column_gap_large");
	tr.appendChild(td);

	// weight input
	td = document.createElement("td");
	const weightInput = document.createElement("input");
	weightInput.type = "number";
	weightInput.value = piece.weight;
	weightInput.classList.add("piece-weight-input");
	td.appendChild(weightInput);
	tr.appendChild(td);

	// gap
	td = document.createElement("td");
	td.classList.add("column_gap_large");
	tr.appendChild(td);

	// delete
	td = document.createElement("td");
	const deleteLink = document.createElement("a");
	deleteLink.textContent = "delete";
	deleteLink.href = "#";
	deleteLink.onclick = "removeTableRow()";
	deleteLink.addEventListener("click", () => {
		event.preventDefault();
		removeTableRow(event.target);
	});
	td.appendChild(deleteLink);
	tr.appendChild(td);

	tbody.appendChild(tr);
}

// remove table row containing this element
function removeTableRow(elem) {
	const tr = elem.closest("tr");
	if (tr) {
		tr.remove();
	}
}

// toggles a bit of a piece on or off in the custom pieces table
function modifyPieceHandler(event) {
	const cells = Array.from(this.getElementsByTagName('div'));
	const index = cells.indexOf(event.target);
	if ( index < 0 ) {
		return;
	}

	const r = Math.floor(index / pieceMaskMaxLength);
	const c = index % pieceMaskMaxLength;
	const isSet = cells[index].classList.contains("occupied");
	const hiddenInput = this.parentElement.querySelector('input');
	let pmask = hiddenInput.value;

	if ( isSet ) {
		pmask = pmask & ~pieceMaskAt(r,c);
	} else {
		pmask = pmask | pieceMaskAt(r,c);
	}

	hiddenInput.value = pmask;
	drawPiecePreview(this, pmask, piecePreviewColor, pieceMaskMaxLength);
}


function setGameOptionsToDefaults() {
	for (const [id, val] of Object.entries(idToDefaultValue)) {
		const el = document.getElementById(id);
		if (!el) {
			console.log(`element with id ${id} does not exist`);
			continue;
		}

		switch (el.type) {
			case "range":
				el.value = val;
				el.dispatchEvent(new Event("input", {}));
				break;
			case "checkbox":
				el.checked = val;
				el.dispatchEvent(new Event("input", {}));
				break;
			case "select-one":
				el.value = val;
				break;
			default:
				console.log(`startGame(): skipping elem id=${id}, type=${el.type}`);
		}
	}

	generateCustomPieceTable("custom-pieces-table", gbDefaultPieces);
}

function toggleShowHideCustomPieceOptions(elem) {
	const customPiecesTableElem = document.getElementById("custom-pieces-container");
	if ( elem.checked ) {
		customPiecesTableElem.style.display = "block";
	} else {
		customPiecesTableElem.style.display = "none";
	}
}

function getLobbyName() {
	try {
		return getCookie("lobby-name")
	} catch (error) {
		console.error(error.message);
	}
	return undefined;
}

async function getLobbyMembers(lobby) {
	const url = "/lobby/get";

	try {
		const response = await fetch(url, { headers: { Accept: "application/json" }});
		if (!response.ok) {
			throw new Error(`Response status: ${response.status}`);
		}

		return await response.json();
	} catch (error) {
		console.error(error.message);
		return {"error": true, "message": error.message};
	}
}

async function updateLobbyMembers() {

	var lobby = getLobbyName();

	var json = await getLobbyMembers(lobby);
	var lobby_div = document.getElementById("member-list");

	if ( "error" in json && json.error === true ) {
		let page = "/lobby/leave"
		if ( "error_page" in json && typeof json.error_page === "string" ) {
			window.location.replace(json.error_page);
		}
	}

	if ( "game" in json && typeof json.game === "string" ) {
		return joinGame();
	}
	if ( !("members" in json) ) {
		lobby_div.innerHTML='<p class="error">Error getting the list of members.</p>';
		return;
	}
	if ( json.members.length === 0 ) {
		lobby_div.innerHTML='<p>Lobby is empty.</p>';
		return;
	}

	document.getElementById("start_game").disabled = json.members.length < 2;

	var tbl = document.createElement("table");
	for (var i = 0; i < json.members.length; i++) {
		var tr = tbl.insertRow();
		var td = tr.insertCell();
		var span = document.createElement("span");
		span.classList.add("colorbox");
		span.style.backgroundColor = json.members[i].color;
		span.style.display = "flex";
		td.appendChild(span);

		td = tr.insertCell();
		td.classList.add("column_gap")

		td = tr.insertCell();
		var p = document.createElement("p");
		p.innerText = json.members[i].name;
		td.appendChild(p);
	}
	lobby_div.replaceChildren(tbl);
}

function getCustomPieces() {
	let pieces = {"data":[]};
	const trs = document.getElementById("custom-pieces-table").querySelectorAll("tbody > tr");

	trs.forEach((tr) => {
		try {
			const pmask = tr.querySelector('.piece-mask-input').value;
			const pweight = tr.querySelector('.piece-weight-input').value;
			pieces.data.push({"mask": parseInt(pmask), "weight": parseFloat(pweight)});
		} catch (error) {
			console.error("Error processing", tr, error.message);
		}
	});
	return pieces;
}

function startGame() {

	// build args
	let lsObj = {}; // localStorage object
	let args = {}; // create game args
	for (const [id, arg] of Object.entries(idToJoinGameArg)) {
		const el = document.getElementById(id);
		if (el === null) {
			console.log(`startGame(): skipping elem id=${id}`);
			continue;
		}

		switch (el.type) {
			case "range":
				lsObj[id] = el.value;
				args[arg] = encodeURIComponent(lsObj[id]);
				break;
			case "checkbox":
				lsObj[id] = el.checked;
				args[arg] = encodeURIComponent(lsObj[id]);
				break;
			case "select-one":
				if (el.value) {
					lsObj[id] = el.value;
					args[arg] = encodeURIComponent(lsObj[id]);
				}
				break;
			default:
				console.log(`startGame(): skipping elem id=${id}, type=${el.type}`);
		}
	}

	const customArgsCheckbox = document.getElementById("use-custom-piece-set-checkbox");
	lsObj["use-custom-piece-set-checkbox"] = customArgsCheckbox.checked;
	if (customArgsCheckbox.checked) {
		const customPieces = getCustomPieces();
		lsObj["pieces"] = customPieces;
		args["pieces"] = encodeURIComponent(JSON.stringify(lsObj["pieces"]));
	}

	// save to localStorage
	try {
		localStorage.setItem("lastGameArgs", JSON.stringify(lsObj));
	} catch (el) {
		console.warn('Failed to store lastGameArgs in localStorage', el);
	}

	// start game
	//console.log(`/game/create?${new URLSearchParams(args).toString()}`);
	window.location.replace(`/game/create?${new URLSearchParams(args).toString()}`);
}

function joinGame() {
	window.location.replace(`/game/join`);
}

function showInviteLink() {
	const lname = getCookie("lobby-name");
	const shareLink = `${window.location.origin}/?lobby=${lname}`;
	displayModalFromURL("/static/modal/invite_link.html", (modal) => {
		// Update the invite link in the modal
		for (el of modal.getElementsByClassName("invite-link")) {
			el.textContent = shareLink;
			if (typeof el.href === "string") {
				el.href = shareLink;
			}
		}
		// Prevent closing the modal when clicking on the link
		for (el of modal.getElementsByClassName("invite-container")) {
			el.addEventListener('click', (event) => {
				event.preventDefault();
				event.stopPropagation();
			});
		}
		// Only expose the copy to clipboard section when it is likely to work
		if (navigator.clipboard && window.isSecureContext) {
			modal.querySelector("#copy-to-clipboard-container").hidden = false;
			modal.querySelector("#copy-to-clipboard-button")
				.addEventListener("click", async () => {
					buttonDisplayNotification(event.target);
					const clipboardMsgElem = modal.querySelector("#clipboard-message");
					try {
						await navigator.clipboard.writeText(shareLink);
						clipboardMsgElem.innerText = "Copied!";
					} catch (err) {
						clipboardMsgElem.innerText = "Copy to clipboard failed.";
						clipboardMsgElem.classList.add("error")
						console.error("Clipboard write failed:", err);
					}
				});
		}
	});

}

// vim: ts=2
