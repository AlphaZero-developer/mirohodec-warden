package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *server) mafiaMiniAppWS(w http.ResponseWriter, r *http.Request) {
	auth := s.mafiaMiniAppAuthFromRequest(r)
	roomID := strings.TrimSpace(r.URL.Query().Get("room_id"))
	if roomID == "" {
		roomID = strings.TrimSpace(r.URL.Query().Get("roomId"))
	}
	if roomID == "" {
		writeMafiaAPIError(w, http.StatusBadRequest, errors.New("room_id is required"))
		return
	}
	if _, err := s.cfg.Service.MafiaMiniAppRoomState(r.Context(), auth, roomID); err != nil {
		writeMafiaAPIError(w, http.StatusUnauthorized, err)
		return
	}
	conn, err := mafiaWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		if s.cfg.Logger != nil {
			s.cfg.Logger.Warn("mafia mini app websocket upgrade failed", zap.Error(err))
		}
		return
	}
	client := &mafiaWSClient{server: s, conn: conn, auth: auth, roomID: roomID}
	s.mafiaHub.add(client)
	defer func() {
		s.mafiaHub.remove(client)
		_ = conn.Close()
	}()
	client.sendState("room_state")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var event mafiaWSEvent
			if err := conn.ReadJSON(&event); err != nil {
				return
			}
			client.handleEvent(event)
		}
	}()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case <-ticker.C:
			client.sendState("room_state")
		}
	}
}

func (c *mafiaWSClient) handleEvent(event mafiaWSEvent) {
	if c == nil || c.server == nil {
		return
	}
	roomID := strings.TrimSpace(event.RoomID)
	if roomID == "" {
		roomID = c.roomID
	}
	var err error
	switch strings.TrimSpace(event.Type) {
	case "request_room_state":
		c.sendState("room_state")
		return
	case "send_message":
		_, err = c.server.cfg.Service.MafiaMiniAppSendMessage(context.Background(), c.auth, MafiaMiniAppMessageInput{RoomID: roomID, Channel: event.Channel, Text: event.Text, ReplyToID: event.ReplyToID})
	case "save_will":
		_, err = c.server.cfg.Service.MafiaMiniAppSaveWill(context.Background(), c.auth, MafiaMiniAppWillInput{RoomID: roomID, Text: event.Text})
	case "skip_phase":
		_, err = c.server.cfg.Service.MafiaMiniAppSkipPhase(context.Background(), c.auth, roomID)
	case "submit_vote":
		_, err = c.server.cfg.Service.MafiaMiniAppSubmitVote(context.Background(), c.auth, MafiaMiniAppVoteInput{RoomID: roomID, Target: event.Target})
	case "submit_night_action":
		_, err = c.server.cfg.Service.MafiaMiniAppSubmitNightAction(context.Background(), c.auth, MafiaMiniAppNightActionInput{RoomID: roomID, Action: event.Action, TargetID: event.TargetID})
	case "start_game":
		_, err = c.server.cfg.Service.MafiaMiniAppStartGame(context.Background(), c.auth, roomID)
	case "next_phase":
		_, err = c.server.cfg.Service.MafiaMiniAppNextPhase(context.Background(), c.auth, roomID)
	case "join_room":
		_, err = c.server.cfg.Service.MafiaMiniAppJoinRoom(context.Background(), c.auth, roomID)
	case "kick_player":
		_, err = c.server.cfg.Service.MafiaMiniAppKickPlayer(context.Background(), c.auth, MafiaMiniAppKickInput{RoomID: roomID, TargetID: event.TargetID})
	case "update_test_bots":
		_, err = c.server.cfg.Service.MafiaMiniAppUpdateTestBots(context.Background(), c.auth, MafiaMiniAppTestBotsInput{RoomID: roomID, Action: event.Action, Count: int(event.TargetID)})
	case "update_settings":
		_, err = c.server.cfg.Service.MafiaMiniAppUpdateSettings(context.Background(), c.auth, roomID, MafiaMiniAppSettingsInput{})
	default:
		c.writeJSON(mafiaWSOutgoing{Type: "error_message", Message: "unknown event"})
		return
	}
	if err != nil {
		c.writeJSON(mafiaWSOutgoing{Type: "error_message", Message: err.Error()})
		return
	}
	c.writeJSON(mafiaWSOutgoing{Type: "action_submitted", Message: "Выбор принят Лайей."})
	c.server.mafiaHub.broadcastRoom(roomID)
}

func (c *mafiaWSClient) sendState(eventType string) {
	if c == nil || c.server == nil {
		return
	}
	state, err := c.server.cfg.Service.MafiaMiniAppRoomState(context.Background(), c.auth, c.roomID)
	if err != nil {
		c.writeJSON(mafiaWSOutgoing{Type: "error_message", Message: err.Error()})
		return
	}
	c.writeJSON(mafiaWSOutgoing{Type: eventType, State: &state})
}

func (c *mafiaWSClient) writeJSON(value interface{}) {
	if c == nil || c.conn == nil {
		return
	}
	c.write.Lock()
	defer c.write.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(mafiaWSWriteWait))
	_ = c.conn.WriteJSON(value)
}

func (s *server) mafiaMiniAppAuthFromRequest(r *http.Request) MafiaMiniAppAuthInput {
	query := r.URL.Query()
	devID, _ := strconv.ParseInt(firstNonEmpty(r.Header.Get("X-Mafia-Dev-User-Id"), query.Get("dev_user_id"), query.Get("devUserId")), 10, 64)
	return MafiaMiniAppAuthInput{
		InitData:    firstNonEmpty(r.Header.Get("X-Telegram-Init-Data"), query.Get("init_data"), query.Get("initData")),
		DevUserID:   devID,
		DevUsername: firstNonEmpty(r.Header.Get("X-Mafia-Dev-Username"), query.Get("dev_username"), query.Get("devUsername")),
		DevName:     firstNonEmpty(r.Header.Get("X-Mafia-Dev-Name"), query.Get("dev_name"), query.Get("devName")),
	}
}

func mafiaMiniAppRoomPath(path string) (string, string) {
	rest := strings.TrimPrefix(path, "/api/mafia/rooms/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func decodeJSON(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeMafiaAPI(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(mafiaAPIResponse{OK: true, Data: data})
}

func writeMafiaAPIError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	message := "unknown error"
	if err != nil {
		message = err.Error()
	}
	_ = json.NewEncoder(w).Encode(mafiaAPIResponse{OK: false, Error: message})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
