package subagent

import (
	"context"
	"errors"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

// TaskHandler is the function signature for handling tasks
// It receives the task context, the full task object, and the initial message
// It returns an artifact (optional), task state, and error message (if failed)
type TaskHandler func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string)

// Skill represents a capability that the agent can perform
type Skill struct {
	Name        string
	Description string
	Handler     TaskHandler
}

// Common errors
var (
	ErrMissingAgentID      = errors.New("agent ID is required")
	ErrMissingName         = errors.New("agent name is required")
	ErrMissingDescription  = errors.New("agent description is required")
	ErrNoSkills            = errors.New("at least one skill must be registered")
	ErrDuplicateSkill      = errors.New("skill with this name already registered")
	ErrAgentNotStarted     = errors.New("agent has not been started")
	ErrAgentAlreadyRunning = errors.New("agent is already running")
)
