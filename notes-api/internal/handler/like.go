package handler


import (
"encoding/json"
"errors"
"net/http"
"strconv"


"github.com/jackc/pgx/v5"


)

func (h *NoteHandler) LikeNote(w http.ResponseWriter, r *http.Request) {
idStr := r.PathValue("id")


id, err := strconv.Atoi(idStr)
if err != nil || id < 1 {
	http.Error(w, "invalid music post ID", http.StatusBadRequest)
	return
}

updatedNote, err := h.Repo.Like(r.Context(), id)
if err != nil {
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "music post not found", http.StatusNotFound)
		return
	}

	http.Error(
		w,
		"failed to like music post",
		http.StatusInternalServerError,
	)
	return
}

w.Header().Set("Content-Type", "application/json")

if err := json.NewEncoder(w).Encode(updatedNote); err != nil {
	http.Error(
		w,
		"failed to encode response",
		http.StatusInternalServerError,
	)
	return
}


}
