// Package tusd provides ways to accept tusd calls using HTTP.
package tusd

import (
	"net/http"

	"github.com/bmizerany/pat"
)

// Handler is a ready to use handler with routing (using pat)
type Handler struct {
	unroutedHandler *UnroutedHandler
	routeHandler    http.Handler
	CompleteUploads chan FileInfo
}

// NewHandler creates a routed tus protocol handler. This is the simplest
// way to use tusd but may not be as configurable as you require. If you are
// integrating this into an existing app you may like to use tusd.NewUnroutedHandler
// instead. Using tusd.NewUnroutedHandler allows the tus handlers to be combined into
// your existing router (aka mux) directly. It also allows the GET and DELETE
// endpoints to be customized. These are not part of the protocol so can be
// changed depending on your needs.
func NewHandler(config Config) (*Handler, error) {
	handler, err := NewUnroutedHandler(config)
	if err != nil {
		return nil, err
	}

	routedHandler := &Handler{
		unroutedHandler: handler,
		CompleteUploads: handler.CompleteUploads,
	}

	mux := pat.New()

	routedHandler.routeHandler = handler.Middleware(mux)

	mux.Post("", http.HandlerFunc(handler.PostFile))
	mux.Head(":id", http.HandlerFunc(handler.HeadFile))
	mux.Add("PATCH", ":id", http.HandlerFunc(handler.PatchFile))

	// Only attach the DELETE handler if the Terminate() method is provided
	if _, ok := config.DataStore.(TerminaterDataStore); ok {
		mux.Del(":id", http.HandlerFunc(handler.DelFile))
	}

	// GET handler requires the GetReader() method
	if _, ok := config.DataStore.(GetReaderDataStore); ok {
		mux.Get(":id", http.HandlerFunc(handler.GetFile))
	}

	return routedHandler, nil
}

// ServeHTTP Implements the http.Handler interface.
func (rHandler *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rHandler.routeHandler.ServeHTTP(w, r)
}
