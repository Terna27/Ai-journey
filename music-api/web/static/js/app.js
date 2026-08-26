const musicList = document.getElementById("music-list");

const audio = document.getElementById("audio");

const playerCover = document.getElementById("player-cover");
const playerTitle = document.getElementById("player-title");
const playerArtist = document.getElementById("player-artist");

const playBtn = document.getElementById("play-btn");
const previousBtn = document.getElementById("previous-btn");
const nextBtn = document.getElementById("next-btn");

const progressBar = document.getElementById("progress-bar");

const currentTimeElement = document.getElementById("current-time");
const durationElement = document.getElementById("duration");

const volumeBar = document.getElementById("volume-bar");

let music = [];
let currentIndex = -1;

// =========================
// LOAD MUSIC
// =========================

async function loadMusic(search = "", genre = "", sort = "newest") {

    try {

        const params = new URLSearchParams();

        if (search) {
            params.set("search", search);
        }

        if (genre) {
            params.set("genre", genre);
        }

        if (sort) {
            params.set("sort", sort);
        }

        const query = params.toString();

        const response = await fetch(
            `/music${query ? `?${query}` : ""}`
        );

        if (!response.ok) {
            throw new Error("Failed to fetch music");
        }

        music = await response.json();

        displayMusic();

    } catch (error) {

        console.error(error);

        musicList.innerHTML = `
            <p class="error">
                Failed to load music.
            </p>
        `;
    }
}

// =========================
// DISPLAY MUSIC
// =========================

function displayMusic() {
  if (!music || music.length === 0) {
    musicList.innerHTML = `
            <p class="loading">
                No music available.
            </p>
        `;

    return;
  }

  musicList.innerHTML = "";

  music.forEach((song, index) => {
    const card = document.createElement("article");

    card.className = "music-card";

    card.innerHTML = `
            <img
                src="${song.image_url || ""}"
                alt="${song.song_title}"
                class="album-cover"
            >

            <div class="music-info">

                <h3>${song.song_title}</h3>

                <p class="artist">
                    ${song.artist_name}
                </p>

                <p class="genre">
                    ${song.genre}
                </p>

              <div class="song-meta">
                <span class="likes">👍 ${song.likes}</span>
                <span class="rating">⭐ ${song.rating}</span>
              </div>

                <button
                    class="song-play-btn"
                    data-index="${index}"
                >
                    ▶ Play
                </button>

            </div>
        `;

    musicList.appendChild(card);
  });

  document.querySelectorAll(".song-play-btn").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number(button.dataset.index);

      playSong(index);
    });
  });
}

// =========================
// PLAY SONG
// =========================

function playSong(index) {
  if (!music[index]) {
    return;
  }

  currentIndex = index;

  const song = music[index];

  audio.src = song.audio_url;

  playerTitle.textContent = song.song_title;
  playerArtist.textContent = song.artist_name;

  playerCover.src = song.image_url || "";

  audio
    .play()
    .then(() => {
      playBtn.textContent = "⏸";
    })
    .catch((error) => {
      console.error("Playback failed:", error);
    });
}

// =========================
// PLAY / PAUSE
// =========================

playBtn.addEventListener("click", () => {
  if (!audio.src) {
    return;
  }

  if (audio.paused) {
    audio.play();

    playBtn.textContent = "⏸";
  } else {
    audio.pause();

    playBtn.textContent = "▶";
  }
});

// =========================
// PREVIOUS
// =========================

previousBtn.addEventListener("click", () => {
  if (music.length === 0) {
    return;
  }

  let index = currentIndex - 1;

  if (index < 0) {
    index = music.length - 1;
  }

  playSong(index);
});

// =========================
// NEXT
// =========================

nextBtn.addEventListener("click", () => {
  if (music.length === 0) {
    return;
  }

  let index = currentIndex + 1;

  if (index >= music.length) {
    index = 0;
  }

  playSong(index);
});

// =========================
// AUTO NEXT SONG
// =========================

audio.addEventListener("ended", () => {
  if (music.length === 0) {
    return;
  }

  let nextIndex = currentIndex + 1;

  if (nextIndex >= music.length) {
    nextIndex = 0;
  }

  playSong(nextIndex);
});

// =========================
// PROGRESS
// =========================

audio.addEventListener("timeupdate", () => {
  if (!audio.duration) {
    return;
  }

  const progress = (audio.currentTime / audio.duration) * 100;

  progressBar.value = progress;

  currentTimeElement.textContent = formatTime(audio.currentTime);
});

audio.addEventListener("loadedmetadata", () => {
  durationElement.textContent = formatTime(audio.duration);
});

progressBar.addEventListener("input", () => {
  if (!audio.duration) {
    return;
  }

  const time = (progressBar.value / 100) * audio.duration;

  audio.currentTime = time;
});

// =========================
// VOLUME
// =========================

volumeBar.addEventListener("input", () => {
  audio.volume = volumeBar.value;
});

// =========================
// TIME FORMAT
// =========================

function formatTime(seconds) {
  if (!seconds || Number.isNaN(seconds)) {
    return "0:00";
  }

  const minutes = Math.floor(seconds / 60);

  const remainingSeconds = Math.floor(seconds % 60);

  return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
}

const searchInput = document.getElementById("search-input");
const genreFilter = document.getElementById("genre-filter");
const sortFilter = document.getElementById("sort-filter");
const searchBtn = document.getElementById("search-btn");


searchBtn.addEventListener("click", () => {

    const search = searchInput.value.trim();
    const genre = genreFilter.value;
    const sort = sortFilter.value;

    loadMusic(search, genre, sort);
});


searchInput.addEventListener("keydown", (event) => {

    if (event.key === "Enter") {

        const search = searchInput.value.trim();
        const genre = genreFilter.value;
        const sort = sortFilter.value;

        loadMusic(search, genre, sort);
    }
});

// =========================
// START
// =========================

loadMusic();
