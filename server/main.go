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
	RedTeamId int  // ID number of the red player's VR headset

	BlueTeam map[string]CozmoState  // states of the blue team cozmos
	BlueTeamId int // ID number of the blue player's VR headset
}

func main() {
	// create the game states JSON object
	game_states := GameStates{
		// initialize the red team
		RedTeam : map[string]CozmoState{
			"cozmo_1" : CozmoState{
				Location : [2]float64{.1, .2},
			},
			"cozmo_2" : CozmoState{
				Location : [2]float64{.3, .4},
			},
			"cozmo_3" : CozmoState{
				Location : [2]float64{.5, .6},
			},
			"cozmo_4" : CozmoState{
				Location : [2]float64{.7, .8},
			},
		},
		
		// initialize the blue team
		BlueTeam : map[string]CozmoState{
			"cozmo_5" : CozmoState{
				Location : [2]float64{.9, .8},
			},
			"cozmo_6" : CozmoState{
				Location : [2]float64{.7, .6},
			},
			"cozmo_7" : CozmoState{
				Location : [2]float64{.5, .4},
			},
			"cozmo_8" : CozmoState{
				Location : [2]float64{.3, .2},
			},
		},
	}
	

	// handle setting robot locations (used by the Cozmo controller)
    http.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		// update a robot parameter
		switch r.Method {
			case "POST":
				// run the ParseForm to pull the POST data, error if applicable
				if err := r.ParseForm(); err != nil {
					fmt.Fprintf(w, "failure: ParseForm() err: %v", err)
					return
				}
				// process the robot, field, and value
				team := r.FormValue("team")  // team name
				robot := r.FormValue("robot")  // robot name
				field := r.FormValue("field")  // field name
				value := r.FormValue("value")  // value for field
				var processed_value [2]float64  // placeholder for the processed value

				// get the pointer to the team we are changing
				var p_team *map[string]CozmoState;  // initialize a pointer to the team we are dealing with
				if (team == "RedTeam") {
					p_team = &game_states.RedTeam;
				} else if (team == "BlueTeam") {
					p_team = &game_states.BlueTeam;
				} else {
					fmt.Fprintf(w, "failure: team must be RedTeam or BlueTeam")
					return
				}
				
				// check if the robot is on the team
				if _, found := (*p_team)[robot]; !found {
					fmt.Fprintf(w, "failure: robot not on the specified team")
					return
				}

				// check if the field is a valid field, and if the value is valid for that field
				if field == "Location" {
					// break the location value into an array
					err := json.Unmarshal([]byte(value), &processed_value)
					if err != nil {
						log.Fatal(err)
					}
					// update the robot location
					if robotObject, ok := (*p_team)[robot]; ok {
						robotObject.Location = processed_value
						(*p_team)[robot] = robotObject
					}
				} else if field == "Waypoint" {
					// break the waypoint value into an array
					err := json.Unmarshal([]byte(value), &processed_value)
					if err != nil {
						log.Fatal(err)
					}
					// update the robot waypoint
					if robotObject, ok := (*p_team)[robot]; ok {
						robotObject.Waypoint = processed_value
						(*p_team)[robot] = robotObject
					}
				} else {
					fmt.Fprintf(w, "failure: field must be Location or Waypoint")
					return
				}
				fmt.Fprintf(w, "success: team=%s, robot=%s, field=%s, value=%s", team, robot, field, value)
				
			default:
				fmt.Fprintf(w, "failure: need POST")  // notify that we only use POST (in case Glen or Jenna get it wrong)
			}
    })

	// handle getting the game states (used by the VR)
    http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)		
		b, _ := json.Marshal(game_states)
		fmt.Fprintf(w, "%s", b)
    })

	fmt.Printf("Running")
    log.Fatal(http.ListenAndServe(":1000", nil))
}
