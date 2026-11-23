package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) routeMafiaCallback(c tele.Context, data string) error {
	switch {
	case data == "mafia:register":
		return a.callbackMafiaRegister(c)
	case data == "mafia:register_intro_ack":
		return a.callbackMafiaRegistrationIntroAck(c)
	case data == "mafia:register_hint":
		return c.Send("⛔ Регистрация проходит в ЛС с Лайей. Открой бота и отправь /start.")
	case data == "mafia:create":
		return a.callbackMafiaCreate(c)
	case data == "mafia:test":
		return a.callbackMafiaFullTest(c, "")
	case data == "mafia:rules":
		return c.Send(mafiaRulesText())
	case data == "mafia:rooms":
		return a.callbackMafiaRooms(c)
	case data == "mafia:bindhint":
		return c.Send("🔗 Привязка группы делается в нужной группе:\n/mafia_bind MF-4821")
	case data == "mafia:queue_join":
		return a.callbackMafiaReadyQueueJoin(c)
	case data == "mafia:queue_leave":
		return a.callbackMafiaReadyQueueLeave(c)
	case data == "mafia:invite_settings":
		return a.callbackMafiaInviteSettings(c)
	case data == "mafia:invite_toggle":
		return a.callbackMafiaInviteToggle(c)
	case data == "mafia:invite_disable":
		return a.callbackMafiaInviteDisable(c)
	case data == "mafia:profile":
		return a.callbackMafiaProfileOpen(c)
	case data == "mafia:achievements":
		return a.callbackMafiaAchievements(c)
	case data == "mafia:profile_edit_name":
		return a.callbackMafiaProfileEditName(c)
	case data == "mafia:profile_edit_backstory":
		return a.callbackMafiaProfileEditBackstory(c)
	case data == "mafia:profile_generate":
		return a.callbackMafiaProfileGenerate(c)
	case data == "mafia:profile_generate_again":
		return a.callbackMafiaProfileGenerate(c)
	case data == "mafia:profile_use_generated":
		return a.callbackMafiaProfileUseGenerated(c)
	case data == "mafia:profile_manual_input":
		return a.startMafiaProfileRegistration(c)
	case data == "mafia:player_search":
		if sender := c.Sender(); sender != nil {
			if room, ok := a.activeMafiaRoomForUser(sender.ID); ok {
				return a.callbackMafiaPlayerSearch(c, room.ID)
			}
		}
		return c.Send("⛔ Комната не найдена.")
	case data == "mafia:back":
		return a.handleMafia(c)
	case strings.HasPrefix(data, "mafia:start:"):
		return a.callbackMafiaStart(c, strings.TrimPrefix(data, "mafia:start:"))
	case strings.HasPrefix(data, "mafia:start_voting:"):
		return a.callbackMafiaStartVoting(c, strings.TrimPrefix(data, "mafia:start_voting:"))
	case strings.HasPrefix(data, "mafia:resolve_night:"):
		return a.callbackMafiaResolveNight(c, strings.TrimPrefix(data, "mafia:resolve_night:"))
	case strings.HasPrefix(data, "mafia:resolve_voting:"):
		return a.callbackMafiaResolveVoting(c, strings.TrimPrefix(data, "mafia:resolve_voting:"))
	case strings.HasPrefix(data, "mafia:players:"):
		return a.callbackMafiaPlayers(c, strings.TrimPrefix(data, "mafia:players:"))
	case strings.HasPrefix(data, "mafia:profiles:"):
		return a.callbackMafiaPlayerProfiles(c, strings.TrimPrefix(data, "mafia:profiles:"))
	case strings.HasPrefix(data, "mafia:invite:"):
		return a.callbackMafiaInviteMenu(c, strings.TrimPrefix(data, "mafia:invite:"))
	case strings.HasPrefix(data, "mafia:invite_menu:"):
		return a.callbackMafiaInviteMenu(c, strings.TrimPrefix(data, "mafia:invite_menu:"))
	case strings.HasPrefix(data, "mafia:recruit_open:"):
		return a.callbackMafiaRecruitmentOpen(c, strings.TrimPrefix(data, "mafia:recruit_open:"))
	case strings.HasPrefix(data, "mafia:recruit_close:"):
		return a.callbackMafiaRecruitmentClose(c, strings.TrimPrefix(data, "mafia:recruit_close:"))
	case strings.HasPrefix(data, "mafia:recruit_mode:"):
		return a.callbackMafiaRecruitmentMode(c, data)
	case strings.HasPrefix(data, "mafia:join_request:"):
		return a.callbackMafiaJoinRequest(c, strings.TrimPrefix(data, "mafia:join_request:"))
	case strings.HasPrefix(data, "mafia:requests:"):
		return a.callbackMafiaRequests(c, strings.TrimPrefix(data, "mafia:requests:"))
	case strings.HasPrefix(data, "mafia:request_accept_first:"):
		return a.callbackMafiaRequestAcceptFirst(c, strings.TrimPrefix(data, "mafia:request_accept_first:"))
	case strings.HasPrefix(data, "mafia:request_decline_all:"):
		return a.callbackMafiaRequestDeclineAll(c, strings.TrimPrefix(data, "mafia:request_decline_all:"))
	case strings.HasPrefix(data, "mafia:request_accept:"):
		return a.callbackMafiaRequestDecision(c, data, true)
	case strings.HasPrefix(data, "mafia:request_decline:"):
		return a.callbackMafiaRequestDecision(c, data, false)
	case strings.HasPrefix(data, "mafia:queue_autofill:"):
		return a.callbackMafiaQueueAutoFill(c, strings.TrimPrefix(data, "mafia:queue_autofill:"))
	case strings.HasPrefix(data, "mafia:invite_active:"):
		return a.callbackMafiaInviteBroadcast(c, strings.TrimPrefix(data, "mafia:invite_active:"), false)
	case strings.HasPrefix(data, "mafia:invite_favorites:"):
		return a.callbackMafiaInviteBroadcast(c, strings.TrimPrefix(data, "mafia:invite_favorites:"), true)
	case strings.HasPrefix(data, "mafia:invite_user:"):
		return a.callbackMafiaInviteUser(c, data)
	case strings.HasPrefix(data, "mafia:favorite_add:"):
		return a.callbackMafiaFavoriteAdd(c, data)
	case strings.HasPrefix(data, "mafia:player_search:"):
		return a.callbackMafiaPlayerSearch(c, strings.TrimPrefix(data, "mafia:player_search:"))
	case strings.HasPrefix(data, "mafia:invite_decline:"):
		return c.Send("🚫 Хорошо, сейчас не зовём.")
	case strings.HasPrefix(data, "mafia:alive:"):
		return a.callbackMafiaAlivePlayers(c, strings.TrimPrefix(data, "mafia:alive:"))
	case strings.HasPrefix(data, "mafia:shadow:"):
		return a.callbackMafiaShadow(c, strings.TrimPrefix(data, "mafia:shadow:"))
	case strings.HasPrefix(data, "mafia:status:"):
		return a.callbackMafiaStatusRoom(c, strings.TrimPrefix(data, "mafia:status:"))
	case strings.HasPrefix(data, "mafia:role:"):
		return a.callbackMafiaRole(c, strings.TrimPrefix(data, "mafia:role:"))
	case strings.HasPrefix(data, "mafia:will:"):
		return a.callbackMafiaWill(c, strings.TrimPrefix(data, "mafia:will:"))
	case strings.HasPrefix(data, "mafia:kill_menu:"):
		return a.callbackMafiaActionTargetMenu(c, strings.TrimPrefix(data, "mafia:kill_menu:"), "kill")
	case strings.HasPrefix(data, "mafia:heal_menu:"):
		return a.callbackMafiaActionTargetMenu(c, strings.TrimPrefix(data, "mafia:heal_menu:"), "heal")
	case strings.HasPrefix(data, "mafia:check_menu:"):
		return a.callbackMafiaActionTargetMenu(c, strings.TrimPrefix(data, "mafia:check_menu:"), "check")
	case strings.HasPrefix(data, "mafia:kill:"):
		return a.callbackMafiaNightAction(c, data, "kill")
	case strings.HasPrefix(data, "mafia:heal:"):
		return a.callbackMafiaNightAction(c, data, "heal")
	case strings.HasPrefix(data, "mafia:check:"):
		return a.callbackMafiaNightAction(c, data, "check")
	case strings.HasPrefix(data, "mafia:vote:"):
		return a.callbackMafiaVote(c, data)
	case strings.HasPrefix(data, "mafia:voteskip:"):
		return a.callbackMafiaVote(c, data)
	case strings.HasPrefix(data, "mafia:skip_phase:"):
		return a.callbackMafiaSkipPhase(c, strings.TrimPrefix(data, "mafia:skip_phase:"))
	case strings.HasPrefix(data, "mafia:leave:"):
		return a.callbackMafiaLeaveRoom(c, strings.TrimPrefix(data, "mafia:leave:"))
	case strings.HasPrefix(data, "mafia:close:"):
		return a.callbackMafiaCloseRoom(c, strings.TrimPrefix(data, "mafia:close:"))
	case strings.HasPrefix(data, "mafia:end:"):
		return a.callbackMafiaEndGame(c, strings.TrimPrefix(data, "mafia:end:"))
	case strings.HasPrefix(data, "mafia:cancel_recreate:"):
		return a.callbackMafiaRecreateCancelledRoom(c, strings.TrimPrefix(data, "mafia:cancel_recreate:"))
	case strings.HasPrefix(data, "mafia:cancel_block:"):
		return a.callbackMafiaBlockCancelledOffender(c, strings.TrimPrefix(data, "mafia:cancel_block:"))
	case strings.HasPrefix(data, "mafia:cancel_close:"):
		return a.callbackMafiaCloseCancelledRoom(c, strings.TrimPrefix(data, "mafia:cancel_close:"))
	case strings.HasPrefix(data, "mafia:leak_cancel:"):
		return a.callbackMafiaLeakCancel(c, strings.TrimPrefix(data, "mafia:leak_cancel:"))
	case strings.HasPrefix(data, "mafia:leak_kill:"):
		return a.callbackMafiaLeakKill(c, strings.TrimPrefix(data, "mafia:leak_kill:"))
	case strings.HasPrefix(data, "mafia:leak_continue:"):
		return a.callbackMafiaLeakContinue(c, strings.TrimPrefix(data, "mafia:leak_continue:"))
	case strings.HasPrefix(data, "mafia:settings_time:"):
		return a.callbackMafiaSettingsTime(c, strings.TrimPrefix(data, "mafia:settings_time:"))
	case strings.HasPrefix(data, "mafia:settings_players:"):
		return a.callbackMafiaSettingsPlayers(c, strings.TrimPrefix(data, "mafia:settings_players:"))
	case strings.HasPrefix(data, "mafia:settings_roles:"):
		return a.callbackMafiaSettingsRoles(c, strings.TrimPrefix(data, "mafia:settings_roles:"))
	case strings.HasPrefix(data, "mafia:settings_voting:"):
		return a.callbackMafiaSettingsVoting(c, strings.TrimPrefix(data, "mafia:settings_voting:"))
	case strings.HasPrefix(data, "mafia:settings_reveal:"):
		return a.callbackMafiaSettingsReveal(c, strings.TrimPrefix(data, "mafia:settings_reveal:"))
	case strings.HasPrefix(data, "mafia:settings:"):
		return a.callbackMafiaSettings(c, strings.TrimPrefix(data, "mafia:settings:"))
	case strings.HasPrefix(data, "mafia:time:"):
		return a.callbackMafiaTimeAdjust(c, data)
	case strings.HasPrefix(data, "mafia:player_limit:"):
		return a.callbackMafiaPlayerLimitAdjust(c, data)
	case strings.HasPrefix(data, "mafia:add_bot:"):
		return a.callbackMafiaAddTestBot(c, strings.TrimPrefix(data, "mafia:add_bot:"), 1)
	case strings.HasPrefix(data, "mafia:fill_bots:"):
		return a.callbackMafiaFillTestBots(c, strings.TrimPrefix(data, "mafia:fill_bots:"))
	case strings.HasPrefix(data, "mafia:clear_bots:"):
		return a.callbackMafiaClearTestBots(c, strings.TrimPrefix(data, "mafia:clear_bots:"))
	case strings.HasPrefix(data, "mafia:toggle_role:"):
		return a.callbackMafiaToggleOptionalRole(c, data)
	case strings.HasPrefix(data, "mafia:toggle:"):
		return a.callbackMafiaToggleSetting(c, data)
	case strings.HasPrefix(data, "mafia:test:"):
		return a.callbackMafiaFullTest(c, strings.TrimPrefix(data, "mafia:test:"))
	case strings.HasPrefix(data, "mafia:joinhint"):
		return c.Send("🚪 Войти можно кодом:\n/mafia_join MF-4821")
	case strings.HasPrefix(data, "mafia:voting_hint:"):
		return c.Send("🗳 Голосование ещё не началось. Лайя сама покажет кнопки, когда начнётся фаза голосования.")
	default:
		return c.Send("🎭 Эта кнопка пока недоступна в текущем контексте.")
	}
}

func mafiaCallbackNeedsManualResponse(data string) bool {
	return strings.HasPrefix(data, "mafia:vote:") || strings.HasPrefix(data, "mafia:voteskip:")
}
