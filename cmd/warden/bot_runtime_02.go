package main

import (
	tele "gopkg.in/telebot.v3"
)

func debugChatBotCommands() []tele.Command {
	commands := append([]tele.Command{}, publicBotCommands()...)
	commands = append(commands,
		tele.Command{Text: "chatid", Description: "показать id чата"},
		tele.Command{Text: "topicid", Description: "показать id темы"},
		tele.Command{Text: "myid", Description: "показать твой user id"},
	)
	return commands
}

func ownerPrivateBotCommands() []tele.Command {
	commands := []tele.Command{
		{Text: "karma", Description: "karma"},
		{Text: "topkarma", Description: "top karma"},
		{Text: "passport", Description: "паспорт участника"},
		{Text: "menu", Description: "меню Лайи"},
		{Text: "event", Description: "событие недели"},
		{Text: "register", Description: "настроить обращение Лайи"},
		{Text: "warn", Description: "warn user"},
		{Text: "warnings", Description: "list warnings"},
		{Text: "unwarn", Description: "remove warning"},
		{Text: "clearwarns", Description: "clear warnings"},
		{Text: "admin", Description: "команды админки"},
		{Text: "jail", Description: "отправить в психцентр"},
		{Text: "rehab", Description: "выпустить из психцентра"},
		{Text: "mute", Description: "замутить участника"},
		{Text: "unmute", Description: "снять мут"},
		{Text: "banforever", Description: "вечный бан"},
		{Text: "unban", Description: "убрать вечный бан"},
		{Text: "all", Description: "позвать известных участников"},
		{Text: "activity", Description: "активность участника"},
		{Text: "topactive", Description: "топ активных участников"},
		{Text: "mafia", Description: "Мафия через Лайю"},
		{Text: "mafia_find", Description: "найти игрока для Мафии"},
		{Text: "will", Description: "завещание в MafiaGame"},
		{Text: "spampardon", Description: "снять антиспам-страйки"},
		{Text: "addphoto", Description: "добавить фото в базу"},
		{Text: "photo", Description: "случайное фото Лайи"},
		{Text: "photos", Description: "список фото в базе"},
		{Text: "delphoto", Description: "удалить фото из базы"},
		{Text: "addquote", Description: "добавить цитату Лайи"},
		{Text: "quote", Description: "случайная цитата Лайи"},
		{Text: "quotes", Description: "список цитат Лайи"},
		{Text: "delquote", Description: "удалить цитату Лайи"},
		{Text: "faq", Description: "FAQ Лайи"},
		{Text: "job", Description: "заявка на работу"},
		{Text: "donate", Description: "донат владельцу"},
		{Text: "faqs", Description: "список FAQ"},
		{Text: "addfaq", Description: "добавить FAQ"},
		{Text: "delfaq", Description: "удалить FAQ"},
		{Text: "idea", Description: "сообщить идею директору"},
		{Text: "bug", Description: "сообщить об ошибке"},
		{Text: "appeal", Description: "обжаловать решение"},
		{Text: "aitest", Description: "проверить AI-ответ"},
		{Text: "cipher", Description: "шифр ответов Лайи"},
		{Text: "reload", Description: "перечитать конфиг и базы"},
		{Text: "selfmute", Description: "самомут на время"},
		{Text: "selfjail", Description: "самому уйти в психцентр"},
		{Text: "selfban", Description: "самобан на время"},
		{Text: "auth", Description: "зарегистрировать владельца"},
		{Text: "myid", Description: "показать твой user id"},
		{Text: "spamlog", Description: "лог антиспама"},
		{Text: "spamstatus", Description: "статус антиспама"},
		{Text: "chatid", Description: "показать id чата"},
		{Text: "topicid", Description: "показать id темы"},
	}
	if len(commands) > 100 {
		return commands[:100]
	}
	return commands
}
