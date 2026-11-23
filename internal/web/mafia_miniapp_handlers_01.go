package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const mafiaWSWriteWait = 5 * time.Second

var mafiaWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type mafiaAPIResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type mafiaWSEvent struct {
	Type      string `json:"type"`
	RoomID    string `json:"roomId,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Text      string `json:"text,omitempty"`
	ReplyToID string `json:"replyToId,omitempty"`
	Action    string `json:"action,omitempty"`
	Target    string `json:"target,omitempty"`
	TargetID  int64  `json:"targetId,omitempty"`
}

type mafiaWSOutgoing struct {
	Type    string             `json:"type"`
	State   *MafiaMiniAppState `json:"state,omitempty"`
	Message string             `json:"message,omitempty"`
}

type mafiaWSHub struct {
	mu      sync.Mutex
	clients map[*mafiaWSClient]struct{}
}

type mafiaWSClient struct {
	server *server
	conn   *websocket.Conn
	auth   MafiaMiniAppAuthInput
	roomID string
	write  sync.Mutex
}

func newMafiaWSHub() *mafiaWSHub {
	return &mafiaWSHub{clients: make(map[*mafiaWSClient]struct{})}
}

func (h *mafiaWSHub) add(client *mafiaWSClient) {
	if h == nil || client == nil {
		return
	}
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *mafiaWSHub) remove(client *mafiaWSClient) {
	if h == nil || client == nil {
		return
	}
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
}

func (h *mafiaWSHub) broadcastRoom(roomID string) {
	if h == nil {
		return
	}
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	if roomID == "" {
		return
	}
	h.mu.Lock()
	clients := make([]*mafiaWSClient, 0, len(h.clients))
	for client := range h.clients {
		if strings.EqualFold(client.roomID, roomID) {
			clients = append(clients, client)
		}
	}
	h.mu.Unlock()
	for _, client := range clients {
		go client.sendState("room_state")
	}
}

func (s *server) mafiaMiniAppAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input MafiaMiniAppAuthInput
	if err := decodeJSON(r, &input); err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	payload, err := s.cfg.Service.MafiaMiniAppAuth(r.Context(), input)
	if err != nil {
		writeMafiaAPIError(w, http.StatusUnauthorized, err)
		return
	}
	writeMafiaAPI(w, payload)
}

func (s *server) mafiaMiniAppProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input MafiaMiniAppProfileInput
	if err := decodeJSON(r, &input); err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	payload, err := s.cfg.Service.MafiaMiniAppUpdateProfile(r.Context(), s.mafiaMiniAppAuthFromRequest(r), input)
	if err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	for _, room := range payload.Rooms {
		s.mafiaHub.broadcastRoom(room.ID)
	}
	writeMafiaAPI(w, payload)
}

func (s *server) mafiaMiniAppProfileView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rawID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mafia/profiles/"), "/")
	userID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || userID == 0 {
		writeMafiaAPIError(w, http.StatusBadRequest, errors.New("user id is required"))
		return
	}
	payload, err := s.cfg.Service.MafiaMiniAppProfile(r.Context(), s.mafiaMiniAppAuthFromRequest(r), userID)
	if err != nil {
		writeMafiaAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeMafiaAPI(w, payload)
}

func (s *server) mafiaMiniAppRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		payload, err := s.cfg.Service.MafiaMiniAppLobby(r.Context(), s.mafiaMiniAppAuthFromRequest(r))
		if err != nil {
			writeMafiaAPIError(w, http.StatusUnauthorized, err)
			return
		}
		writeMafiaAPI(w, payload)
	case http.MethodPost:
		state, err := s.cfg.Service.MafiaMiniAppCreateRoom(r.Context(), s.mafiaMiniAppAuthFromRequest(r))
		if err != nil {
			writeMafiaAPIError(w, http.StatusBadRequest, err)
			return
		}
		s.mafiaHub.broadcastRoom(state.Room.ID)
		writeMafiaAPI(w, state)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
