package http

import (
	"net/http"

	"github.com/google/uuid"

	"bakery/internal/app"
	"bakery/internal/domain/user"
)

func (s *Server) listKitchens(w http.ResponseWriter, r *http.Request) {
	ks, err := s.kitchen.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, kitchenDTOs(ks))
}

func (s *Server) createKitchen(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string   `json:"name"`
		UserID *user.ID `json:"user_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	k, err := s.kitchen.Create(r.Context(), app.CreateKitchenInput{Name: body.Name, UserID: body.UserID})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, kitchenDTO(k))
}

func (s *Server) getKitchen(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	k, err := s.kitchen.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, kitchenDTO(k))
}

func (s *Server) updateKitchen(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	k, err := s.kitchen.Update(r.Context(), id, app.UpdateKitchenInput{Name: body.Name})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, kitchenDTO(k))
}

func (s *Server) deleteKitchen(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.kitchen.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
