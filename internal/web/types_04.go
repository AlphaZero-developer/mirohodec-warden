package web

type LayaUsersPage struct {
	ChatID int64
	Rows   []LayaUserRow
}

type LayaUserRow struct {
	ID          int64
	DisplayName string
	Username    string
	Gender      string
	Address     string
	TagsText    string
	Note        string
	Updated     string
}

type LayaUserInput struct {
	ChatID      int64
	UserID      int64
	Username    string
	DisplayName string
	Gender      string
	Address     string
	Tags        string
	Note        string
}

type AICacheStats struct {
	SystemPrompts int
}

type AILogPage struct {
	Rows          []AILogRow
	Total         int
	Info          int
	Warn          int
	Error         int
	OK            int
	Errors        int
	AverageTime   string
	LastEntry     string
	MaxEntries    int
	TotalRequests int
}

type AILogRow struct {
	ID            uint64
	At            string
	Source        string
	Stage         string
	Level         string
	Status        string
	Error         string
	OK            bool
	Elapsed       string
	Model         string
	BaseURL       string
	MaxTokens     int
	MaxReplyChars int
	UserID        int64
	User          string
	Username      string
	ChatID        int64
	Chat          string
	ChatType      string
	ThreadID      int
	MessageID     int
	UserInput     string
	SystemPrompt  string
	Messages      []AILogMessage
	RawReply      string
	FinalReply    string
	RequestChars  int
	ResponseChars int
}

type AILogMessage struct {
	Role    string
	Content string
	Chars   int
}

type FAQRow struct {
	ID           string
	Patterns     []string
	PatternsText string
	Answer       string
	Enabled      bool
	Uses         int64
	LastUsed     string
}

type FAQInput struct {
	ID       string
	Patterns string
	Answer   string
	Enabled  bool
}

type QuoteRow struct {
	ID       string
	Text     string
	Author   string
	Tags     []string
	TagsText string
	Uses     int64
	LastUsed string
}

type QuoteInput struct {
	ID     string
	Text   string
	Author string
	Tags   string
}

type PhotoRow struct {
	ID          string
	FileID      string
	UniqueID    string
	Caption     string
	Tags        []string
	TagsText    string
	AddedByName string
	AddedAt     string
	Uses        int64
	LastUsed    string
}

type PhotoInput struct {
	FileID   string
	UniqueID string
	Caption  string
	Tags     string
}
