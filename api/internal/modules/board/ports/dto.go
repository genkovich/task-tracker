package ports

import (
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// dueDateLayout is how a deadline travels on the wire: a calendar day, never
// an RFC3339 timestamp (tasks contract `due_date`, format: date). The column
// is DATE, and a time of day in a deadline would be invented precision that
// also shifts across time zones.
const dueDateLayout = "2006-01-02"

// TaskResponse mirrors the Task schema (tasks contracts/openapi.yaml) — the
// full task row, returned by create/edit and inside a task detail.
type TaskResponse struct {
	ID          string    `json:"id"`
	ColumnID    string    `json:"column_id"`
	Title       string    `json:"title"`
	Assignee    *string   `json:"assignee"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	DueDate     *string   `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskCardResponse mirrors the TaskCard schema (tasks contract) — a task as
// it appears inside a board state. It deliberately carries no description
// body, only whether there is one, plus the comment count: a board refetch
// runs on every SSE event, and dragging every description along would make
// the cheapest, most frequent response the heaviest one (tasks spec §6).
type TaskCardResponse struct {
	ID             uuid.UUID `json:"id"`
	ColumnID       uuid.UUID `json:"column_id"`
	Title          string    `json:"title"`
	Assignee       *string   `json:"assignee"`
	Priority       string    `json:"priority"`
	DueDate        *string   `json:"due_date"`
	HasDescription bool      `json:"has_description"`
	CommentCount   int       `json:"comment_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TaskDetailResponse mirrors the TaskDetail schema (tasks contract) — the
// full task plus its comments, oldest first. The same shape serves the
// team-editor and the public viewer.
type TaskDetailResponse struct {
	Task     TaskResponse      `json:"task"`
	Comments []CommentResponse `json:"comments"`
}

// CommentResponse mirrors the Comment schema (tasks contract). No updated_at:
// a comment is never edited (spec §3).
type CommentResponse struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentCreateRequest mirrors the CommentCreate schema (tasks contract).
type CommentCreateRequest struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

// TaskCreateRequest mirrors the TaskCreate schema (boards + tasks contracts):
// board_id names the board whose leftmost column receives the task (BRD-08)
// — the column itself is still chosen by the server, never the client.
type TaskCreateRequest struct {
	BoardID     string  `json:"board_id"`
	Title       string  `json:"title"`
	Assignee    *string `json:"assignee"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
}

// TaskUpdateRequest mirrors the TaskUpdate schema (tasks contract). Not a
// merge-patch: a field left out takes its zero value, exactly as assignee
// already did before this feature.
type TaskUpdateRequest struct {
	Title       string  `json:"title"`
	Assignee    *string `json:"assignee"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
}

// TaskMoveRequest mirrors the TaskMove schema (contracts/openapi.yaml).
type TaskMoveRequest struct {
	ColumnID string `json:"column_id"`
}

// BoardCreateRequest mirrors the BoardCreate schema (boards contract).
type BoardCreateRequest struct {
	Name string `json:"name"`
}

// parseDueDate turns the wire value into a domain due date. An absent, null
// or empty value means "no deadline" — the same clearing semantics assignee
// has. A value that is not a calendar day is a caller error, not a 500 from
// the driver further down.
func parseDueDate(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	due, err := time.Parse(dueDateLayout, *raw)
	if err != nil {
		return nil, err
	}
	return &due, nil
}

func formatDueDate(due *time.Time) *string {
	if due == nil {
		return nil
	}
	formatted := due.Format(dueDateLayout)
	return &formatted
}

func toTaskResponse(task *domain.Task) TaskResponse {
	return TaskResponse{
		ID:          task.ID.String(),
		ColumnID:    task.ColumnID.String(),
		Title:       task.Title,
		Assignee:    task.Assignee,
		Description: task.Description,
		Priority:    string(task.Priority),
		DueDate:     formatDueDate(task.DueDate),
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

func toCommentResponse(comment domain.Comment) CommentResponse {
	return CommentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		Author:    comment.Author,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
	}
}

func toCommentResponses(comments []domain.Comment) []CommentResponse {
	resp := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp = append(resp, toCommentResponse(c))
	}
	return resp
}

func toTaskDetailResponse(detail *TaskDetail) TaskDetailResponse {
	return TaskDetailResponse{
		Task:     toTaskResponse(&detail.Task),
		Comments: toCommentResponses(detail.Comments),
	}
}
