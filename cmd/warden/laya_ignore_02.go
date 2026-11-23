package main

import (
	"fmt"
	"strings"
)

func layaIgnoreTargetName(target targetUser, rec LayaIgnoredUser) string {
	if strings.TrimSpace(target.Name) != "" {
		return target.Name
	}
	if strings.TrimSpace(rec.DisplayName) != "" {
		return rec.DisplayName
	}
	if strings.TrimSpace(target.Username) != "" {
		return "@" + target.Username
	}
	if strings.TrimSpace(rec.Username) != "" {
		return "@" + rec.Username
	}
	if target.ID != 0 {
		return fmt.Sprintf("user %d", target.ID)
	}
	if rec.UserID != 0 {
		return fmt.Sprintf("user %d", rec.UserID)
	}
	return "пользователь"
}
