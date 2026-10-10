// ---------- bar visualizer ----------

const barsEl = document.getElementById("bars");
const BAR_COUNT = 28;

for (let i = 0; i < BAR_COUNT; i++) {
  const bar = document.createElement("span");
  bar.style.setProperty("--h", 20 + Math.random() * 70 + "%");
  bar.style.animationDelay = (Math.random() * 2).toFixed(2) + "s";
  barsEl.appendChild(bar);
}

function setAnalyzing(on) {
  barsEl.classList.toggle("analyzing", on);
}

// ---------- state ----------

let selectedFile = null; // File or Blob

const dropzone = document.getElementById("dropzone");
const fileInput = document.getElementById("fileInput");
const filenameEl = document.getElementById("filename");
const identifyBtn = document.getElementById("identifyBtn");
const recordBtn = document.getElementById("recordBtn");
const statusEl = document.getElementById("status");
const resultEl = document.getElementById("result");

function setFile(file, label) {
  selectedFile = file;
  identifyBtn.disabled = !file;
  filenameEl.hidden = !file;
  filenameEl.textContent = label || (file ? file.name : "");
  hide(resultEl);
  hide(statusEl);
}

function hide(el) {
  el.hidden = true;
  el.innerHTML = "";
}

// ---------- file picking ----------

dropzone.addEventListener("click", () => fileInput.click());
dropzone.addEventListener("keydown", (e) => {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    fileInput.click();
  }
});

fileInput.addEventListener("change", () => {
  const f = fileInput.files[0];
  if (f) setFile(f);
});

["dragover", "dragenter"].forEach((evt) =>
  dropzone.addEventListener(evt, (e) => {
    e.preventDefault();
    dropzone.classList.add("drag");
  }),
);
["dragleave", "drop"].forEach((evt) =>
  dropzone.addEventListener(evt, (e) => {
    e.preventDefault();
    dropzone.classList.remove("drag");
  }),
);
dropzone.addEventListener("drop", (e) => {
  const f = e.dataTransfer.files[0];
  if (f) setFile(f);
});

// ---------- mic recording ----------

let mediaRecorder = null;
let recordedChunks = [];
let recordTimer = null;
const RECORD_SECONDS = 15;

recordBtn.addEventListener("click", async () => {
  if (mediaRecorder && mediaRecorder.state === "recording") {
    mediaRecorder.stop();
    return;
  }

  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    recordedChunks = [];
    mediaRecorder = new MediaRecorder(stream);

    mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) recordedChunks.push(e.data);
    };

    mediaRecorder.onstop = () => {
      clearTimeout(recordTimer);
      stream.getTracks().forEach((t) => t.stop());
      recordBtn.classList.remove("recording");
      recordBtn.lastChild.textContent = "Record 15s from mic";

      const blob = new Blob(recordedChunks, {
        type: mediaRecorder.mimeType || "audio/webm",
      });
      setFile(blob, "mic-recording.webm");
    };

    mediaRecorder.start();
    identifyBtn.disabled = true;
    hide(resultEl);
    hide(statusEl);
    recordBtn.classList.add("recording");
    recordBtn.lastChild.textContent = "Recording\u2026 tap to stop";

    recordTimer = setTimeout(() => {
      if (mediaRecorder.state === "recording") mediaRecorder.stop();
    }, RECORD_SECONDS * 1000);
  } catch (err) {
    showStatus(
      "Couldn't access the microphone. Check your browser permissions.",
      true,
    );
  }
});

// ---------- identify ----------

identifyBtn.addEventListener("click", async () => {
  if (!selectedFile) return;

  hide(resultEl);
  showStatus("Uploading and analyzing\u2026", false);
  setAnalyzing(true);
  identifyBtn.disabled = true;

  const form = new FormData();
  form.append("recording", selectedFile, selectedFile.name || "recording.webm");

  try {
    const res = await fetch("/identify", { method: "POST", body: form });
    if (!res.ok) throw new Error(await readError(res));

    const data = await res.json();
    hide(statusEl);
    renderResult(data);
  } catch (err) {
    let msg = err.message;
    if (err instanceof TypeError) {
      msg = "Couldn't reach the server. Check your connection and try again.";
    } else if (err instanceof SyntaxError) {
      msg = "The server sent a response this page couldn't read.";
    }
    showStatus(msg, true);
  } finally {
    setAnalyzing(false);
    identifyBtn.disabled = false;
  }
});

async function readError(res) {
  const text = (await res.text()).trim();
  if (res.status >= 500) {
    return "The server hit an error while analyzing that clip. Try again in a moment.";
  }
  return text || "Request failed (" + res.status + ")";
}

// ---------- title cleanup (display only) ----------

// Stored titles come from YouTube filenames, so strip the usual noise.
function cleanTitle(raw) {
  return (raw || "")
    .replace(
      /\s*[\(\[][^\)\]]*(official|lyric|audio|video|visualizer|skiza|hd|4k)[^\)\]]*[\)\]]/gi,
      "",
    )
    .replace(/\s*\bSMS\b.*$/i, "")
    .replace(/\s{2,}/g, " ")
    .trim();
}

function splitArtistTitle(raw) {
  const cleaned = cleanTitle(raw);
  const i = cleaned.indexOf(" - ");
  if (i === -1) return { artist: "", title: cleaned };
  return {
    artist: cleaned.slice(0, i).trim(),
    title: cleaned.slice(i + 3).trim(),
  };
}

// ---------- rendering ----------

function showStatus(message, isError) {
  statusEl.hidden = false;
  statusEl.classList.toggle("error", !!isError);
  statusEl.textContent = message;
}

function placeholderArt(title) {
  const initial = (title || "?").trim().charAt(0).toUpperCase();
  return `<div class="result-art placeholder" aria-hidden="true">${escapeHtml(initial)}</div>`;
}

function renderResult(data) {
  const confidence = data && data.confidence;

  // Don't guess. A missing field means the API and this page disagree.
  if (!confidence) {
    console.warn("Unexpected /identify response:", data);
    showStatus(
      "The server replied in a format this page didn't expect. Details are in the browser console.",
      true,
    );
    return;
  }

  if (confidence === "no_match") {
    resultEl.innerHTML = `
      <div class="result-empty">
        <span class="badge badge-no_match"><span class="dot"></span>No match</span>
        <p>Nothing in the library matched closely enough.</p>
        <p class="result-note">Clips of 8 to 15 seconds with little background noise work best. This server can only recognise tracks that were fingerprinted into its library.</p>
      </div>
    `;
    resultEl.hidden = false;
    return;
  }

  // Use iTunes' clean title/artist when we have them, otherwise parse the stored one.
  const parsed = splitArtistTitle(data.title);
  const artist = data.artist || parsed.artist;
  const title = data.artist
    ? data.title || "Unknown title"
    : parsed.title || data.title || "Unknown title";

  const label =
    confidence === "confident" ? "Confident match" : "Possible match";

  const art = data.artwork_url
    ? `<img class="result-art" src="${escapeAttr(data.artwork_url)}" alt="">`
    : placeholderArt(title);

  resultEl.innerHTML = `
    ${art}
    <div class="result-body">
      <span class="badge badge-${escapeAttr(confidence)}"><span class="dot"></span>${label}</span>
      <p class="result-title">${escapeHtml(title)}</p>
      ${artist ? `<p class="result-artist">${escapeHtml(artist)}</p>` : ""}
      ${typeof data.votes === "number" ? `<p class="result-votes">${data.votes} hashes agreed</p>` : ""}
      ${
        confidence === "possible"
          ? `<p class="result-note">Weaker signal than usual. It may be this track. A longer or cleaner clip would settle it.</p>`
          : ""
      }
      ${data.artwork_url ? "" : `<p class="result-note subtle">No artwork found for this track.</p>`}
    </div>
  `;
  resultEl.hidden = false;

  // If the image URL is dead, fall back to the placeholder instead of a broken icon.
  const img = resultEl.querySelector("img.result-art");
  if (img) {
    img.addEventListener("error", () => {
      img.outerHTML = placeholderArt(title);
    });
  }
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

function escapeAttr(str) {
  return String(str).replace(/"/g, "&quot;");
}
