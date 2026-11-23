package main

import (
	"math/rand"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func startMafiaTestRoom(t *testing.T, store *Store, players int) (*MafiaService, MafiaRoom) {
	t.Helper()
	service := NewMafiaService(store)
	service.rng = rand.New(rand.NewSource(7))
	now := time.Now().UTC()
	users := make([]*tele.User, 0, players)
	for i := 0; i < players; i++ {
		user := &tele.User{ID: int64(5000 + i), FirstName: "Player"}
		users = append(users, user)
		if _, err := service.RegisterUser(user, now); err != nil {
			t.Fatalf("RegisterUser(%d): %v", user.ID, err)
		}
	}
	room, err := service.CreateRoom(users[0], now)
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	for _, user := range users[1:] {
		if _, err := service.JoinRoom(room.ID, user, now); err != nil {
			t.Fatalf("JoinRoom(%d): %v", user.ID, err)
		}
	}
	if _, err := service.BindGroup(room.ID, users[0].ID, -1005000, now); err != nil {
		t.Fatalf("BindGroup: %v", err)
	}
	room, err = service.StartGame(room.ID, users[0].ID, now)
	if err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	return service, room
}
