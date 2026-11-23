package main

import (
	"errors"
	"math/rand"
	"time"

	tele "gopkg.in/telebot.v3"
)

var (
	errMafiaUserNotRegistered = errors.New("mafia user is not registered")
	errMafiaRoomNotFound      = errors.New("mafia room not found")
	errMafiaRoomNotLobby      = errors.New("mafia room is not in lobby")
	errMafiaNotRoomOwner      = errors.New("only room owner can do this")
	errMafiaPlayerNotFound    = errors.New("mafia player not found")
	errMafiaPlayerDead        = errors.New("dead players cannot do this")
	errMafiaWrongRole         = errors.New("wrong mafia role for this action")
	errMafiaWrongPhase        = errors.New("wrong mafia phase for this action")
	errMafiaShadowUnavailable = errors.New("mafia shadow channel unavailable")
	errMafiaVoteExists        = errors.New("mafia vote already exists")
	errMafiaNightActionExists = errors.New("mafia night action already submitted")
	errMafiaWillsDisabled     = errors.New("mafia wills are disabled")
	errMafiaSettingsLocked    = errors.New("mafia settings are locked")
	errMafiaRecruitmentClosed = errors.New("mafia recruitment is closed")
	errMafiaRoomFull          = errors.New("mafia room is full")
	errMafiaRequestExists     = errors.New("mafia join request already exists")
	errMafiaInviteLimited     = errors.New("mafia invite limit exceeded")
	errMafiaInvitesDisabled   = errors.New("mafia invites are disabled")
	errMafiaHostBlocked       = errors.New("mafia host blocked this player")
	errMafiaProfileIncomplete = errors.New("mafia profile is incomplete")
)

type MafiaInviteLimits struct {
	MaxInviteBroadcastPerRoom         int
	MinSecondsBetweenInviteBroadcasts int
	MaxInvitesPerUserPerDay           int
}

var mafiaInviteLimits = MafiaInviteLimits{
	MaxInviteBroadcastPerRoom:         1,
	MinSecondsBetweenInviteBroadcasts: 1800,
	MaxInvitesPerUserPerDay:           3,
}

type MafiaService struct {
	store RuntimeStore
	rng   *rand.Rand
	ai    AIService
	aiCfg AIConfig
}

type MafiaNightResult struct {
	KilledID int64
	Saved    bool
	Quiet    bool
}

type MafiaVoteResult struct {
	ExecutedID int64
	Tie        bool
	Votes      map[string]string
}

const (
	mafiaMinPlayersLimit int   = 4
	mafiaMaxPlayersLimit int   = 12
	mafiaTestBotBaseID   int64 = -880000000000
)

func NewMafiaService(store RuntimeStore) *MafiaService {
	return &MafiaService{store: store, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func NewMafiaServiceWithAI(store RuntimeStore, ai AIService, cfg AIConfig) *MafiaService {
	service := NewMafiaService(store)
	service.ai = ai
	service.aiCfg = cfg
	return service
}

func (s *MafiaService) RegisterUser(user *tele.User, now time.Time) (MafiaRegisteredUser, error) {
	return s.store.RegisterMafiaUser(user, now)
}

func (s *MafiaService) CreateRoom(owner *tele.User, now time.Time) (MafiaRoom, error) {
	if owner == nil || owner.ID == 0 {
		return MafiaRoom{}, errMafiaUserNotRegistered
	}
	registered, ok := s.store.MafiaUser(owner.ID)
	if !ok {
		return MafiaRoom{}, errMafiaUserNotRegistered
	}
	room := MafiaRoom{
		ID:                s.newRoomID(),
		OwnerID:           owner.ID,
		GroupChatID:       s.rememberedGroupChatID(owner.ID),
		Status:            MafiaStatusLobby,
		Players:           []MafiaPlayer{mafiaPlayerFromUser(registered, now)},
		DayNumber:         0,
		Settings:          defaultMafiaRoomSettings(),
		CreatedAt:         now.UTC().Unix(),
		UpdatedAt:         now.UTC().Unix(),
		RecruitmentStatus: MafiaRecruitmentClosed,
		RecruitmentMode:   MafiaRecruitmentRequests,
	}
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) rememberedGroupChatID(ownerID int64) int64 {
	if s == nil || s.store == nil || ownerID == 0 {
		return 0
	}
	if user, ok := s.store.MafiaUser(ownerID); ok {
		settings := normalizeMafiaUserSettings(user.Settings)
		if settings.LastDiscussionGroupChatID != 0 {
			return settings.LastDiscussionGroupChatID
		}
	}
	for _, room := range s.store.MafiaRoomsForUser(ownerID) {
		if room.OwnerID == ownerID && room.GroupChatID != 0 {
			return room.GroupChatID
		}
	}
	return 0
}

func (s *MafiaService) JoinRoom(roomID string, user *tele.User, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusLobby {
		return MafiaRoom{}, errMafiaRoomNotLobby
	}
	if s.store.MafiaHostBlocked(room.OwnerID, user.ID) {
		return MafiaRoom{}, errMafiaHostBlocked
	}
	registered, ok := s.store.MafiaUser(user.ID)
	if !ok {
		return MafiaRoom{}, errMafiaUserNotRegistered
	}
	if mafiaRoomPlayerIndex(room, user.ID) >= 0 {
		return room, nil
	}
	if len(room.Players) >= normalizeMafiaSettings(room.Settings).MaxPlayers {
		return MafiaRoom{}, errMafiaRoomFull
	}
	room.Players = append(room.Players, mafiaPlayerFromUser(registered, now))
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) BindGroup(roomID string, ownerID, groupChatID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusLobby {
		return MafiaRoom{}, errMafiaRoomNotLobby
	}
	room.GroupChatID = groupChatID
	room.UpdatedAt = now.UTC().Unix()
	if user, ok := s.store.MafiaUser(ownerID); ok {
		settings := normalizeMafiaUserSettings(user.Settings)
		settings.LastDiscussionGroupChatID = groupChatID
		_ = s.store.SetMafiaUserSettings(ownerID, settings)
	}
	return room, s.store.SaveMafiaRoom(room)
}
