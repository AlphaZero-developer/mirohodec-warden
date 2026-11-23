package web

import (
	"net/http"
)

func (s *server) mafiaMiniAppJoinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		RoomID string `json:"roomId"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	state, err := s.cfg.Service.MafiaMiniAppJoinRoom(r.Context(), s.mafiaMiniAppAuthFromRequest(r), input.RoomID)
	if err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	s.mafiaHub.broadcastRoom(state.Room.ID)
	writeMafiaAPI(w, state)
}

func (s *server) mafiaMiniAppRoom(w http.ResponseWriter, r *http.Request) {
	roomID, action := mafiaMiniAppRoomPath(r.URL.Path)
	if roomID == "" || action == "" {
		http.NotFound(w, r)
		return
	}
	auth := s.mafiaMiniAppAuthFromRequest(r)
	var (
		data        interface{}
		state       MafiaMiniAppState
		err         error
		broadcastID = roomID
	)
	switch action {
	case "state":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err = s.cfg.Service.MafiaMiniAppRoomState(r.Context(), auth, roomID)
		data = state
	case "message":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppMessageInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		input.RoomID = roomID
		state, err = s.cfg.Service.MafiaMiniAppSendMessage(r.Context(), auth, input)
		data = state
	case "will":
		if r.Method != http.MethodPost && r.Method != http.MethodPatch {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppWillInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		input.RoomID = roomID
		state, err = s.cfg.Service.MafiaMiniAppSaveWill(r.Context(), auth, input)
		data = state
	case "skip-phase":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err = s.cfg.Service.MafiaMiniAppSkipPhase(r.Context(), auth, roomID)
		data = state
	case "vote":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppVoteInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		input.RoomID = roomID
		state, err = s.cfg.Service.MafiaMiniAppSubmitVote(r.Context(), auth, input)
		data = state
	case "night-action":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppNightActionInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		input.RoomID = roomID
		state, err = s.cfg.Service.MafiaMiniAppSubmitNightAction(r.Context(), auth, input)
		data = state
	case "start":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err = s.cfg.Service.MafiaMiniAppStartGame(r.Context(), auth, roomID)
		data = state
	case "next-phase":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err = s.cfg.Service.MafiaMiniAppNextPhase(r.Context(), auth, roomID)
		data = state
	case "settings":
		if r.Method != http.MethodPost && r.Method != http.MethodPatch {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppSettingsInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		state, err = s.cfg.Service.MafiaMiniAppUpdateSettings(r.Context(), auth, roomID, input)
		data = state
	case "kick":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppKickInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		input.RoomID = roomID
		state, err = s.cfg.Service.MafiaMiniAppKickPlayer(r.Context(), auth, input)
		data = state
	case "test-bots":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input MafiaMiniAppTestBotsInput
		if err := decodeJSON(r, &input); err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		input.RoomID = roomID
		state, err = s.cfg.Service.MafiaMiniAppUpdateTestBots(r.Context(), auth, input)
		data = state
	case "leave":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err = s.cfg.Service.MafiaMiniAppLeaveRoom(r.Context(), auth, roomID)
	case "close":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err = s.cfg.Service.MafiaMiniAppCloseRoom(r.Context(), auth, roomID)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	if action != "state" {
		if state.Room.ID != "" {
			broadcastID = state.Room.ID
		}
		s.mafiaHub.broadcastRoom(broadcastID)
	}
	writeMafiaAPI(w, data)
}
