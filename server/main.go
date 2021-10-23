package main

import (
	"fmt"
	"encoding/json"
    "log"
    "net/http"
)

type CozmoState struct {
	Location [2]float64  // x/y location of the cozmo's location
	Waypoint [2]float64  // x/y location of the waypoint location
	HasFlag bool  // whether the cozmo bears the flag
	CanMove bool  // whether the cozmo can move
}

type GameStates struct {
	RedTeam map[string]CozmoState  // states of the red team cozmos
	BlueTeam map[string]CozmoState  // states of the blue team cozmos
}

func main() {
	// create the game states JSON object
	game_states := GameStates{
		// initialize the red team
		RedTeam : map[string]CozmoState{
			"cozmo 1" : CozmoState{},
			"cozmo 2" : CozmoState{},
			"cozmo 3" : CozmoState{},
			"cozmo 4" : CozmoState{},
		},
		
		// initialize the blue team
		BlueTeam : map[string]CozmoState{
			"cozmo 5" : CozmoState{},
			"cozmo 6" : CozmoState{},
			"cozmo 7" : CozmoState{},
			"cozmo 8" : CozmoState{},
		},
	}

	// initialize the red Cozmo locations

	// initialize the blue Cozmo locations

	// initialize the flag state
	

	// handle setting robot locations (used by the Cozmo controller)
    http.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		
        fmt.Fprintf(w, "success")
    })

	// handle getting the game states (used by the VR)
    http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)		
		b, _ := json.Marshal(game_states)
		fmt.Fprintf(w, "%s", b)
    })

    log.Fatal(http.ListenAndServeTLS(":1000", "certs/server.crt", "certs/server.key", nil))
}
