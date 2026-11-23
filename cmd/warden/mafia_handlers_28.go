package main

import (
	"strings"
)

func normalizeMafiaCallbackData(data string) string {
	switch data {
	case "mafia_open":
		return "mafia:back"
	case "mafia_register":
		return "mafia:register"
	case "create_mafia_room", "mafia_create_room":
		return "mafia:create"
	case "join_mafia_room", "enter_mafia_room_code", "mafia_join_by_code":
		return "mafia:joinhint"
	case "mafia_my_rooms":
		return "mafia:rooms"
	case "mafia_full_test":
		return "mafia:test"
	case "mafia_rules":
		return "mafia:rules"
	case "mafia_back":
		return "mafia:back"
	case "mafia_bind_group", "bind_mafia_group":
		return "mafia:bindhint"
	case "mafia_ready_queue_join":
		return "mafia:queue_join"
	case "mafia_ready_queue_leave":
		return "mafia:queue_leave"
	case "mafia_player_search":
		return "mafia:player_search"
	case "mafia_invite_disable":
		return "mafia:invite_disable"
	case "mafia_profile_open":
		return "mafia:profile"
	case "mafia_profile_edit_name":
		return "mafia:profile_edit_name"
	case "mafia_profile_edit_backstory":
		return "mafia:profile_edit_backstory"
	case "mafia_profile_generate":
		return "mafia:profile_generate"
	case "mafia_profile_generate_again":
		return "mafia:profile_generate_again"
	case "mafia_profile_use_generated":
		return "mafia:profile_use_generated"
	case "mafia_profile_manual_input":
		return "mafia:profile_manual_input"
	case "mafia_profile_back":
		return "mafia:profile"
	}
	if strings.HasPrefix(data, "mafia_vote_skip_") {
		return "mafia:voteskip:" + strings.TrimPrefix(data, "mafia_vote_skip_")
	}
	for _, prefix := range []struct {
		raw string
		out string
	}{
		{"mafia_room_players_", "mafia:players:"},
		{"mafia_invite_players_", "mafia:invite:"},
		{"mafia_invite_menu_", "mafia:invite_menu:"},
		{"mafia_recruitment_open_", "mafia:recruit_open:"},
		{"mafia_recruitment_close_", "mafia:recruit_close:"},
		{"mafia_join_request_", "mafia:join_request:"},
		{"mafia_requests_open_", "mafia:requests:"},
		{"mafia_request_accept_first_", "mafia:request_accept_first:"},
		{"mafia_request_decline_all_", "mafia:request_decline_all:"},
		{"mafia_ready_queue_auto_fill_", "mafia:queue_autofill:"},
		{"mafia_invite_active_", "mafia:invite_active:"},
		{"mafia_favorites_invite_", "mafia:invite_favorites:"},
		{"mafia_invite_decline_", "mafia:invite_decline:"},
		{"mafia_bind_group_", "mafia:bindhint:"},
		{"mafia_room_settings_", "mafia:settings:"},
		{"mafia_close_room_", "mafia:close:"},
		{"mafia_end_game_", "mafia:end:"},
		{"mafia_start_game_", "mafia:start:"},
		{"mafia_settings_time_", "mafia:settings_time:"},
		{"mafia_settings_players_", "mafia:settings_players:"},
		{"mafia_settings_roles_", "mafia:settings_roles:"},
		{"mafia_settings_reveal_policy_", "mafia:settings_reveal:"},
		{"mafia_settings_voting_", "mafia:settings_voting:"},
		{"mafia_open_shadow_chat_", "mafia:shadow:"},
		{"mafia_refresh_shadow_chat_", "mafia:shadow:"},
		{"mafia_will_open_", "mafia:will:"},
		{"mafia_player_profiles_", "mafia:profiles:"},
		{"mafia_skip_phase_", "mafia:skip_phase:"},
		{"mafia_add_test_bot_", "mafia:add_bot:"},
		{"mafia_fill_test_bots_", "mafia:fill_bots:"},
		{"mafia_clear_test_bots_", "mafia:clear_bots:"},
		{"mafia_full_test_", "mafia:test:"},
		{"start_mafia_game_", "mafia:start:"},
		{"close_mafia_room_", "mafia:close:"},
		{"end_mafia_game_", "mafia:end:"},
		{"open_mafia_settings_", "mafia:settings:"},
		{"invite_mafia_players_", "mafia:invite:"},
	} {
		if strings.HasPrefix(data, prefix.raw) {
			if prefix.out == "mafia:bindhint:" {
				return "mafia:bindhint"
			}
			return prefix.out + strings.TrimPrefix(data, prefix.raw)
		}
	}
	for _, prefix := range []struct {
		raw  string
		mode string
	}{
		{"mafia_recruitment_mode_requests_", string(MafiaRecruitmentRequests)},
		{"mafia_recruitment_mode_auto_", string(MafiaRecruitmentAutoAccept)},
	} {
		if strings.HasPrefix(data, prefix.raw) {
			return "mafia:recruit_mode:" + strings.TrimPrefix(data, prefix.raw) + ":" + prefix.mode
		}
	}
	for _, prefix := range []struct {
		raw    string
		action string
	}{
		{"mafia_request_accept_", "request_accept"},
		{"mafia_request_decline_", "request_decline"},
	} {
		if strings.HasPrefix(data, prefix.raw) {
			rest := strings.TrimPrefix(data, prefix.raw)
			idx := strings.LastIndex(rest, "_")
			if idx <= 0 || idx == len(rest)-1 {
				return data
			}
			return "mafia:" + prefix.action + ":" + rest[:idx] + ":" + rest[idx+1:]
		}
	}
	for _, prefix := range []struct {
		raw string
		key string
	}{
		{"mafia_settings_wills_", "wills"},
		{"mafia_settings_achievements_", "achievements"},
		{"mafia_settings_ai_report_", "ai_report"},
	} {
		if strings.HasPrefix(data, prefix.raw) {
			return "mafia:toggle:" + strings.TrimPrefix(data, prefix.raw) + ":" + prefix.key
		}
	}
	for _, prefix := range []struct {
		raw    string
		action string
	}{
		{"open_mafia_players_", "players"},
		{"open_mafia_shadow_chat_", "shadow"},
		{"mafia_write_shadow_message_", "shadow"},
		{"mafia_night_kill_", "kill"},
		{"mafia_doctor_heal_", "heal"},
		{"mafia_detective_check_", "check"},
		{"mafia_vote_", "vote"},
	} {
		if strings.HasPrefix(data, prefix.raw) {
			rest := strings.TrimPrefix(data, prefix.raw)
			if prefix.action == "players" || prefix.action == "shadow" {
				return "mafia:" + prefix.action + ":" + rest
			}
			idx := strings.LastIndex(rest, "_")
			if idx <= 0 || idx == len(rest)-1 {
				return data
			}
			return "mafia:" + prefix.action + ":" + rest[:idx] + ":" + rest[idx+1:]
		}
	}
	return data
}
