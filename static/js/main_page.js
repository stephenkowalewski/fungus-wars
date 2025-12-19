// requires: common.js

// save player name and color choice
function storePlayerInfo() {
	const player_name = document.getElementById("pname").value;
	const player_color = document.getElementById("pcolor").value;

	try {
		localStorage.setItem("playerName", player_name);
		localStorage.setItem("playerColor", player_color);
	} catch (el) {
		console.warn('Failed to store player info in localStorage', el);
	}
}

// restore player name and color choice
function restorePlayerInfo() {
	const storedName = localStorage.getItem("playerName");
	const storedColor = localStorage.getItem("playerColor");

	if (storedName) {
		const el = document.getElementById("pname");
		el.value = storedName;
		el.dispatchEvent(new Event("input", {}));
	}
	if (storedColor) {
		const el = document.getElementById("pcolor");
		el.value = storedColor;
		el.dispatchEvent(new Event("input", {}));
	}
}

function processURLArgs() {
	const urlParams = new URLSearchParams(window.location.search);
	const lobby = urlParams.get("lobby");
	if (lobby) {
		document.getElementById("lname").value = lobby;
	}
}

async function getOpenLobbies() {
	const url = "/lobby/list";

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

async function updateOpenLobbyList() {
	var json = await getOpenLobbies();
	var lobby_datalist = document.getElementById("open-lobbies");

	if ( !("lobbies" in json) ) {
		const lobby_notice = document.getElementById("lobby_notice");
		if (lobby_notice.innerHTML.trim() === "") {
			lobby_notice.innerHTML='<p class="error">Error getting the list of lobbies.</p>';
		}
		console.log("Error getting the list of lobbies from", json);
		return;
	}

	// provide a default option when there are no active lobbies
	if ( json.lobbies.length === 0 ) {
		json.lobbies = ["lobby1"];
	}

  lobby_datalist.innerHTML = "";
	json.lobbies.forEach(lobby => {
		const option = document.createElement("option");
		option.value = lobby;
		lobby_datalist.appendChild(option);
	});
}

// If the user is already in a lobby, let them know
// and prevent clicking submit
function updateLobbyNotice() {
	let lobby = getCookie("lobby-name");
	if ( lobby === "" ) { return; }
	document.getElementById("lobby_notice").innerHTML = `
<p class="error">You are already in a lobby (${lobby})</p>
<p>You can <a href="/lobby">go there now</a> or <a href="/lobby/leave">leave the lobby</a>.</p><br>
`;
	let submit = document.getElementById("join_lobby");
	submit.disabled = "disabled";
}

// update the color preview or display an invalid color error
function previewColorInput() {
	let input = document.getElementById("pcolor");
	let preview = document.getElementById("pcolor_preview");
	let messageDiv = document.getElementById("pcolor_message");
	let newMessageDiv = messageDiv.cloneNode();
	if ( input === null || preview === null || messageDiv === null ) {
		console.log("previewColorInput() could not find elements for pcolor or pcolor_preview");
		return;
	}
	let s = input.value;
	if ( s.length < 3 ) {
		preview.style.display = "none";
		preview.style.backgroundColor = "";
		newMessageDiv.innerText = "";
		newMessageDiv.classList = [];
		messageDiv.replaceWith(newMessageDiv);
		return;
	}
	if (s[0] === "#") { s=s.substr(1); }
	if ( s.length === 3 && s.match(/^[0-9a-fA-F]{3}$/)) { s=`${s[0]}${s[0]}${s[1]}${s[1]}${s[2]}${s[2]}`; }
	if ( s.length !== 6 || !s.match(/^[0-9a-fA-F]{6}$/) ) {
		preview.style.display = "none";
		preview.style.backgroundColor = "";
		newMessageDiv.classList.add("warning");
		newMessageDiv.innerText = "Invalid color code";
		messageDiv.replaceWith(newMessageDiv);
		return;
	}

	preview.style.display = "flex";
	preview.style.backgroundColor = "#"+s;
	newMessageDiv.innerText = "";
	newMessageDiv.classList = [];
	messageDiv.replaceWith(newMessageDiv);
}

function randomizeColor() {
	let r = Math.floor(Math.random()*256).toString(16).padStart(2, "0");
	let g = Math.floor(Math.random()*256).toString(16).padStart(2, "0");
	let b = Math.floor(Math.random()*256).toString(16).padStart(2, "0");

	let input = document.getElementById("pcolor");
	input.value = `#${r}${g}${b}`;
	previewColorInput();
}

// vim: ts=2
