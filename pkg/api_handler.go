package webglue

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// SessionHeader       = "Webglue-Session"
	PingHeader          = "Webglue-Ping"
	ContentTypeHeader   = "Content-Type"
	ContentTypeJson     = "application/json"
	ContentLengthHeader = "Content-Length"

	SessionExpiration = 1 * time.Minute
	SessionPing       = 10 * time.Second
)

// type SessionAndTimestamp struct {
// 	Session   any
// 	Timestamp time.Time
// }

type ApiHandler struct {
	// nextSessionId        int
	// sessionAndTimestamps map[string]*SessionAndTimestamp
	// sessionsLock         sync.Mutex
	// sessionFactory       SessionFactory
	apiMarshaler *ApiMarshaler
}

// func (handler *ApiHandler) getSession(sid string) (any, string) {
// 	handler.sessionsLock.Lock()
// 	defer handler.sessionsLock.Unlock()

// 	sessionAndTimestamp, ok := handler.sessionAndTimestamps[sid]
// 	if !ok {
// 		sid = strconv.Itoa(handler.nextSessionId)
// 		handler.nextSessionId++

// 		sessionAndTimestamp = &SessionAndTimestamp{
// 			Session: handler.sessionFactory(sid),
// 		}
// 		handler.sessionAndTimestamps[sid] = sessionAndTimestamp
// 	}
// 	sessionAndTimestamp.Timestamp = time.Now()
// 	return sessionAndTimestamp.Session, sid
// }

func (handler *ApiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	//session, sid := handler.getSession(request.Header.Get(SessionHeader))

	pathSplit := strings.Split(request.URL.Path, "/")
	if len(pathSplit) < 3 {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	moduleName := pathSplit[len(pathSplit)-2]
	functionName := pathSplit[len(pathSplit)-1]

	responseHeaders := writer.Header()
	//responseHeaders.Set(SessionHeader, sid)
	responseHeaders.Set(PingHeader, strconv.Itoa(int(SessionPing.Seconds())))
	responseHeaders.Set(ContentTypeHeader, ContentTypeJson)

	if request.Method == http.MethodHead {
		responseHeaders.Set(ContentLengthHeader, "0")
		return
	}

	handler.apiMarshaler.call(request.Context(), moduleName, functionName, request.Body, writer)

}

func newApiHandler(modules []*Module) (*ApiHandler, error) {

	apiMarshaler, err := newApiMarshaler(modules)
	if err != nil {
		return nil, err
	}

	apiHandler := &ApiHandler{
		apiMarshaler: apiMarshaler,
	}

	return apiHandler, nil
}
