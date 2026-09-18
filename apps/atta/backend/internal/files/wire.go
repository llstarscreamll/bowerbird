package files

import (
	"net/http"

	httpV1 "github.com/atta/internal/files/adapters/http/v1"
	"github.com/atta/internal/files/api"
	"github.com/atta/internal/files/application"
	"github.com/atta/internal/platform/config"
	platformStorage "github.com/atta/internal/platform/storage"
)

func NewApplication(fileStore platformStorage.FileStore) *application.Application {
	if fileStore == nil {
		panic("file store is required")
	}

	return application.NewApplication(fileStore)
}

func NewTenantObjects(fileStore platformStorage.FileStore) api.TenantObjects {
	return application.NewTenantObjects(fileStore)
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}

	if app == nil {
		panic("files application is required")
	}

	controller := httpV1.NewController(
		app.Commands.RequestUploadURL,
		app.Commands.RequestDownloadURL,
	)
	router := httpV1.NewRouter(controller)
	router.Register(mux, cfg, authMiddleware)

	return router
}
