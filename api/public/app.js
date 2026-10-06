// ---------- bar visualizer ----------

const barsEl = document.getElementById("bars");
const BAR_COUNT = 28;

for (let i = 0; i < BAR_COUNT; i++) {
  const bar = document.createElement("span");
  const h = 20 + Math.random() * 70;
  bar.style.setProperty("--h", h + "%");
  bar.style.animationDelay = (Math.random() * 2).toFixed(2) + "s";
  barsEl.appendChild(bar);
}

function setAnalyzing(on) {
  barsEl.classList.toggle("analyzing", on);
}

// ---------- state ----------

let selectedFile = null; // File or Blob

const dropzone = document.getElementById("dropzone");
const dropzoneLabel = document.getElementById("dropzoneLabel");
const fileInput = document.getElementById("fileInput");
const filenameEl = document.getElementById("filename");
const identifyBtn = document.getElementById("identifyBtn");
const recordBtn = document.getElementById("recordBtn");
const recordDot = document.getElementById("recordDot");
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
const RECORD_SECONDS = 8;

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
      recordBtn.lastChild.textContent = "Record 8s from mic";

      const blob = new Blob(recordedChunks, {
        type: mediaRecorder.mimeType || "audio/webm",
      });
      setFile(blob, "mic-recording.webm");
    };

    mediaRecorder.start();
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
  const name = selectedFile.name || "recording.webm";
  form.append("recording", selectedFile, name);

  try {
    const res = await fetch("/identify", { method: "POST", body: form });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || "Request failed with status " + res.status);
    }

    const data = await res.json();
    hide(statusEl);
    renderResult(data);
  } catch (err) {
    showStatus(
      err.message || "Something went wrong talking to the server.",
      true,
    );
  } finally {
    setAnalyzing(false);
    identifyBtn.disabled = false;
  }
});

// ---------- rendering ----------

function showStatus(message, isError) {
  statusEl.hidden = false;
  statusEl.classList.toggle("error", !!isError);
  statusEl.textContent = message;
}

function renderResult(data) {
  const confidence = data.confidence || "no_match";

  if (confidence === "no_match") {
    resultEl.innerHTML = `
      <div class="result-empty">
        <span class="badge badge-no_match"><span class="dot"></span>No match</span>
        <p>Nothing in the library matched closely enough. Try a longer or clearer clip.</p>
      </div>
    `;
    resultEl.hidden = false;
    return;
  }

  const label =
    confidence === "confident" ? "Confident match" : "Possible match";
  const art = data.artwork_url
    ? `<img class="result-art" src="${escapeAttr(data.artwork_url)}" alt="">`
    : `<div class="result-art"></div>`;

  resultEl.innerHTML = `
    ${art}
    <div class="result-body">
      <span class="badge badge-${escapeAttr(confidence)}"><span class="dot"></span>${label}</span>
      <p class="result-title">${escapeHtml(data.title || "Unknown title")}</p>
      <p class="result-artist">${escapeHtml(data.artist || "Unknown artist")}</p>
      ${typeof data.votes === "number" ? `<p class="result-votes">${data.votes} hashes agreed</p>` : ""}
    </div>
  `;
  resultEl.hidden = false;
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

function escapeAttr(str) {
  return String(str).replace(/"/g, "&quot;");
}
