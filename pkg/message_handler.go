package webglue

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	SessionHeader = "Webglue-Session"
)

type Session struct {
	Id string
}

type MessageHandler struct {
	nextSessionId int
	sessions      map[string]*Session
	sessionsLock  sync.Mutex
}

func (handler *MessageHandler) getSession(sid string) *Session {
	handler.sessionsLock.Lock()
	defer handler.sessionsLock.Unlock()

	session, ok := handler.sessions[sid]
	if !ok {
		sid = strconv.Itoa(handler.nextSessionId)
		handler.nextSessionId++
		session = &Session{Id: sid}
		handler.sessions[sid] = session
	}
	return session
}

func (handler *MessageHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	session := handler.getSession(request.Header.Get(SessionHeader))
	writer.Header().Set(SessionHeader, session.Id)

	err := request.Body.Close()
	if err != nil {
		print("err: " + err.Error())
	}

	for {
		select {
		case <-time.After(1 * time.Second):
			print("ok")
			writer.Write([]byte("Hello, world!\n"))
			writer.(http.Flusher).Flush()

		case <-request.Context().Done():
			print("ko")
			return
		}
	}

}

func newMessageHandler(options Options) (*MessageHandler, error) {
	return &MessageHandler{
		sessions: make(map[string]*Session),
	}, nil
}
