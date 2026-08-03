package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func (h *NoteHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid note ID", http.StatusBadRequest)
		return
	}

	err = h.Repo.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
