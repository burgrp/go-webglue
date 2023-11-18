package webglue

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SessionHeader       = "Webglue-Session"
	ContentTypeHeader   = "Content-Type"
	ContentTypeJson     = "application/json"
	ContentLengthHeader = "Content-Length"
)

type SessionAndTimestamp struct {
	Session   any
	Timestamp time.Time
}

type ApiHandler struct {
	nextSessionId        int
	sessionAndTimestamps map[string]SessionAndTimestamp
	sessionsLock         sync.Mutex
	sessionFactory       SessionFactory
	apiMarshaler         *ApiMarshaler
}

func (handler *ApiHandler) getSession(sid string) (any, string) {
	handler.sessionsLock.Lock()
	defer handler.sessionsLock.Unlock()

	sessionAndTimestamp, ok := handler.sessionAndTimestamps[sid]
	if !ok {
		sid = strconv.Itoa(handler.nextSessionId)
		handler.nextSessionId++

		sessionAndTimestamp = SessionAndTimestamp{
			Session: handler.sessionFactory(sid),
		}
		handler.sessionAndTimestamps[sid] = sessionAndTimestamp
	}
	sessionAndTimestamp.Timestamp = time.Now()
	return sessionAndTimestamp.Session, sid
}

func (handler *ApiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	session, sid := handler.getSession(request.Header.Get(SessionHeader))

	pathSplit := strings.Split(request.URL.Path, "/")
	function := pathSplit[len(pathSplit)-1]

	responseHeaders := writer.Header()
	responseHeaders.Set(SessionHeader, sid)
	responseHeaders.Set(ContentTypeHeader, ContentTypeJson)

	if request.Method == http.MethodHead {
		responseHeaders.Set(ContentLengthHeader, "0")
		return
	}

	if function == "" {
		handler.apiMarshaler.describe(session, writer)
	} else {
		handler.apiMarshaler.call(session, request.Context(), function, request.Body, writer)
	}

}

func newMessageHandler(options Options) (*ApiHandler, error) {

	apiMarshaler, err := newApiMarshaler(options)
	if err != nil {
		return nil, err
	}

	return &ApiHandler{
		sessionAndTimestamps: make(map[string]SessionAndTimestamp),
		sessionFactory:       options.SessionFactory,
		apiMarshaler:         apiMarshaler,
	}, nil
}
