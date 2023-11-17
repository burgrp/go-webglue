package webglue

import (
	"net/http"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
)

func newWebSocketHandler() (http.Handler, error) {
	server := socketio.NewServer(&engineio.Options{
		ConnInitor: func(r *http.Request, conn engineio.Conn) {
			println("conn initor", conn.ID())
		},
	})

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		s.Emit("connected", "hello world")
		return nil
	})

	server.OnEvent("/", "message", func(s socketio.Conn, msg string) {
		s.Emit("message", msg)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		println("#error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		println("#closed", reason)
	})

	// go func() {
	// 	err := server.Serve()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()

	go func() {
		err := server.Serve()
		if err != nil {
			panic(err)
		}
	}()
	//defer server.Close()

	return server, nil
}
