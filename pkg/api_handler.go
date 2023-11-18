package webglue

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	SessionHeader = "Webglue-Session"
)

type ApiHandler struct {
	nextSessionId  int
	sessions       map[string]any
	sessionsLock   sync.Mutex
	sessionFactory SessionFactory
}

func (handler *ApiHandler) getSession(sid string) (any, string) {
	handler.sessionsLock.Lock()
	defer handler.sessionsLock.Unlock()

	session, ok := handler.sessions[sid]
	if !ok {
		sid = strconv.Itoa(handler.nextSessionId)
		handler.nextSessionId++

		session = handler.sessionFactory(sid)
		handler.sessions[sid] = session
	}
	return session, sid
}

func (handler *ApiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	_, sid := handler.getSession(request.Header.Get(SessionHeader))
	writer.Header().Set(SessionHeader, sid)

	body := make(chan []byte)
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := request.Body.Read(buffer)
			if n > 0 {
				body <- buffer[:n]
			}
			if err != nil {
				if err != io.EOF {
					println("err: " + err.Error())
				}
				break
			}
		}
		close(body)
	}()

loop:
	for {
		select {
		case request := <-body:
			if len(request) == 0 {
				println("end of body")
				break loop
			}
			println("request: " + string(request))
		case <-time.After(1 * time.Second):
			print("ok")
			writer.Write([]byte("Hello, world!\n"))
			writer.(http.Flusher).Flush()

		case <-request.Context().Done():
			println("closed by client")
			break loop
		}
	}

}

func newMessageHandler(options Options) (*ApiHandler, error) {
	return &ApiHandler{
		sessions:       make(map[string]any),
		sessionFactory: options.SessionFactory,
	}, nil
}
