package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoomRejectsConnectionsAtCapacity(t *testing.T) {
	r := newRoom()
	for i := 0; i < cap(r.connections); i++ {
		r.connections <- struct{}{}
	}
	req := httptest.NewRequest(http.MethodGet, "/room", nil)
	req = req.WithContext(context.WithValue(req.Context(), authPayloadContextKey{}, "verified-payload"))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(response.Body.String(), "room capacity reached") {
		t.Fatal("capacity response did not explain the rejection")
	}
}

func TestRoomRemovingClientTwiceDoesNotPanic(t *testing.T) {
	r := newRoom()
	c := &client{send: make(chan *message, 1)}
	r.clients[c] = true
	r.removeClient(c)
	r.removeClient(c)
	if len(r.clients) != 0 {
		t.Fatal("removed client remained in the room")
	}
	if _, ok := <-c.send; ok {
		t.Fatal("client send channel was not closed")
	}
}

func TestRoomHandlesBadWebSocketUpgradeWithoutExitingProcess(t *testing.T) {
	r := newRoom()
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/room", nil)
		req = req.WithContext(context.WithValue(req.Context(), authPayloadContextKey{}, "verified-payload"))
		response := httptest.NewRecorder()
		r.ServeHTTP(response, req)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d: status=%d, want %d", i+1, response.Code, http.StatusBadRequest)
		}
	}
}
