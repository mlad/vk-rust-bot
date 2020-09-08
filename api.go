package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const ApiAddress string = "127.0.0.1:61416"

func apiResponseProjectsTop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	players := make(map[uint32]int)
	maxPlayers := make(map[uint32]int)

	for i, j := 0, len(RustServers); i < j; i++ {
		srv := &RustServers[i]

		players[srv.Key] += srv.Players
		maxPlayers[srv.Key] += srv.MaxPlayers
	}

	// Что я сделал?
	resp := struct {
		Players    *map[uint32]int
		MaxPlayers *map[uint32]int
	}{
		Players:    &players,
		MaxPlayers: &maxPlayers,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
}

func RunApiServer() {
	http.HandleFunc("/projectsTop", apiResponseProjectsTop)

	log.Fatal(http.ListenAndServe(ApiAddress, nil))
}
