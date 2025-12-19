let decodedCookies;
function getCookie(name) {
	if ( !decodedCookies ) {
		decodedCookies = Object.fromEntries(document.cookie.split("; ")
			.filter(v => v.includes("="))
			.map(v => {
				const [key, val] = v.split("=");
				return [key, decodeURIComponent(val.replace(/\+/g, " "))];
		}));
	}
	return decodedCookies[name] || "";
}

function setContentByClass(text, className) {
	var elems = document.getElementsByClassName(className);
	for (let i = 0; i < elems.length; i++) {
		try {
			elems.item(i).textContent = text;
		} finally {
		}
	}
}

// Display an underline animation on a button
function buttonDisplayNotification(elem) {
	elem.classList.add("notify");
	setTimeout(() => {
		elem.classList.remove("notify");
	}, 250);
}

// Display a full screen modal. Click or Escape key to dismiss.
// customizeModal is an optional function that recieves the modal div as
// its only argument to allow further customization.
async function displayModalFromURL(url, customizeModal = null) {
	var content;

	try {
		const response = await fetch(url, { headers: { Accept: "text/html" }});
		if (!response.ok) {
			throw new Error(`Response status: ${response.status}`);
		}
		content = await response.text();
	} catch (error) {
		console.error(error.message);
		content = `<p class="error">Error loading ${url}<p>`;
	}

	const modalDiv = document.createElement("div");
	modalDiv.classList.add("full-screen-modal-overlay");
	modalDiv.tabIndex = 0;
	modalDiv.innerHTML = content;

	if (typeof customizeModal === "function") {
		customizeModal(modalDiv);
	}

	modalDiv.addEventListener('click', (event) => {
		event.currentTarget.remove();
		document.body.style.overflow = '';
	});

	modalDiv.addEventListener('keydown', (event) => {
		if (event.key === "Escape" ) {
			event.currentTarget.remove();
			document.body.style.overflow = '';
		}
	});

	document.body.prepend(modalDiv);
	document.body.style.overflow = 'hidden';
	modalDiv.focus();
}

async function displayHowToPlay() {
	displayModalFromURL("/static/modal/how_to_play.html");
}

// vim: ts=2
