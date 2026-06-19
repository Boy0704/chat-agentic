package agent

const (
	EventTypeToken       = "token"
	EventTypeSkillStart  = "skill_start"
	EventTypeSkillResult = "skill_result"
	EventTypeDone        = "done"
	EventTypeError       = "error"
)

type Event struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Skill   string `json:"skill,omitempty"`
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`

	MessageID  string   `json:"message_id,omitempty"`
	SkillsUsed []string `json:"skills_used,omitempty"`
}
