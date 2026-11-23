package main

import (
	"strconv"
	"strings"
)

func profileIntValue(value int, a *App) string {
	if value == 0 {
		return a.t("no_data")
	}
	return strconv.Itoa(value)
}

func usernameWithAt(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return ""
	}
	if strings.HasPrefix(username, "@") {
		return username
	}
	return "@" + username
}
