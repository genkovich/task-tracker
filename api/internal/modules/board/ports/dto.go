package ports

import "time"

// TaskResponse mirrors the Task schema (contracts/openapi.yaml) — the wire
// shape returned by create/edit/move (T7). Identical to PublicTaskResponse
// (public_handler.go); kept as its own type so the team-editor task routes
// don't couple to the public-viewer DTO's name.
type TaskResponse struct {
	ID        string    `json:"id"`
	ColumnID  string    `json:"column_id"`
	Title     string    `json:"title"`
	Assignee  *string   `json:"assignee"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskCreateRequest mirrors the TaskCreate schema (boards contract):
// board_id names the board whose leftmost column receives the task (BRD-08)
// — the column itself is still chosen by the server, never the client.
type TaskCreateRequest struct {
	BoardID  string  `json:"board_id"`
	Title    string  `json:"title"`
	Assignee *string `json:"assignee"`
}

// BoardCreateRequest mirrors the BoardCreate schema (boards contract).
type BoardCreateRequest struct {
	Name string `json:"name"`
}

// TaskUpdateRequest mirrors the TaskUpdate schema (contracts/openapi.yaml).
type TaskUpdateRequest struct {
	Title    string  `json:"title"`
	Assignee *string `json:"assignee"`
}

// TaskMoveRequest mirrors the TaskMove schema (contracts/openapi.yaml).
type TaskMoveRequest struct {
	ColumnID string `json:"column_id"`
}
