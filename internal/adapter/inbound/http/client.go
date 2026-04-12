package http

import (
	"net/http"

	"github.com/google/uuid"

	"bakery/internal/app"
)

func (s *Server) listClients(w http.ResponseWriter, r *http.Request) {
	cs, err := s.client.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clientDTOs(cs))
}

func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string     `json:"name"`
		UserID         *uuid.UUID `json:"user_id"`
		TelegramChatID *int64     `json:"telegram_chat_id"`
		Note           string     `json:"note"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	c, err := s.client.Create(r.Context(), app.CreateClientInput{
		Name:           body.Name,
		UserID:         body.UserID,
		TelegramChatID: body.TelegramChatID,
		Note:           body.Note,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, clientDTO(c))
}

func (s *Server) getClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := s.client.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clientDTO(c))
}

func (s *Server) updateClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name           string `json:"name"`
		TelegramChatID *int64 `json:"telegram_chat_id"`
		Note           string `json:"note"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	c, err := s.client.Update(r.Context(), id, app.UpdateClientInput{
		Name:           body.Name,
		TelegramChatID: body.TelegramChatID,
		Note:           body.Note,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clientDTO(c))
}

func (s *Server) deleteClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.client.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
