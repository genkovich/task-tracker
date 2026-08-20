package user

import (
	"github.com/genkovich/task-tracker/api/internal/modules/user/app"
	"github.com/genkovich/task-tracker/api/internal/modules/user/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/user/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
)

func New(db *database.DB, st storage.ObjectStorage) *ports.Handler {
	repo := infra.NewPostgresUserRepository(db)
	svc := app.NewService(repo)
	return ports.NewHandler(svc, st)
}
