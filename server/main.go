
// Code Citations:
// strtoint: https://stackoverflow.com/questions/4278430
// foreach key in map: https://stackoverflow.com/questions/1841443/
// foreach element in array: https://yourbasic.org/golang/for-loop-range-array-slice-map-channel/
// assigning field to struct: https://stackoverflow.com/questions/42605337/


package main

import (
	"fmt"
	"encoding/json"
    "log"
    "net/http"
	"strconv"
	"math"
)

type CozmoState struct {
	Location [2]float64  // x/y location of the cozmo's location
	Waypoint [2]float64  // x/y location of the waypoint location
	HasRedFlag bool  // whether the cozmo bears the red flag
	HasBlueFlag bool // whether the cozmo bears the blue flag
	CanMove bool  // whether the cozmo can move
	AuraCount float64  // the aura influence on the robot
}

type GameStates struct {
	RedTeam map[string]CozmoState  // states of the red team cozmos
	RedTeamOculusId int  // ID number of the red player's VR headset
	RedFlagAtBase bool  // whether the red team's flag is at their base
	RedFlagBaseLocation [2]float64  // x/y location of the flag base
	RedTeamScore int  // score of the red team

	BlueTeam map[string]CozmoState  // states of the blue team cozmos
	BlueTeamOculusId int // ID number of the blue player's VR headset
	BlueFlagAtBase bool // whether the blue team's flag is at their base
	BlueFlagBaseLocation [2]float64  // x/y location of the flag base
	BlueTeamScore int  // score of the blue team
}

// distance calculation
func dist(a [2]float64, b [2]float64) float64 {
	return math.Sqrt(math.Pow(a[0] - b[0], 2) + math.Pow(a[1] - b[1], 2))
}

// uses the game states object to determine the HasFlag(s), CanMove, and FlagAtBase bools
func engine(game_states *GameStates) {
	// determine auras between each robot
	aura_range := 0.15  // range at which robots connect
	
	robot_proximities := [][2]string{}  // names of robots from opposite teams in proximity to each other, so we can parse for flag transfers after calculating auras

	// for each robot on the red team...
	for red_robot_id, red_robot_state := range(game_states.RedTeam) {
		// for each other robot on the red team...
		for _, other_red_robot_state := range(game_states.RedTeam) {
			// if the robots are close (team mates), add an aura, INCLUDING self
			if dist(red_robot_state.Location, other_red_robot_state.Location) <= aura_range {
				red_robot_state.AuraCount += 1.0
			}
		}

		// for each robot on the blue team...
		robot_proximities = nil  // reset the proximity list
		for blue_robot_id, blue_robot_state := range(game_states.BlueTeam) {
			// if the robots are close (enemies), reduce aura on both robots
			if dist(red_robot_state.Location, blue_robot_state.Location) <= aura_range {
				red_robot_state.AuraCount -= 1.0
				blue_robot_state.AuraCount -= 1.0
				robot_proximities = append(robot_proximities, [2]string{red_robot_id, blue_robot_id})  // note that these robots are proximal for red
			}

			// update the blue robot state
			game_states.BlueTeam[blue_robot_id] = blue_robot_state
		}

		// if the red robot is on the blue side (y < 0.5), reduce aura by 0.5
		if red_robot_state.Location[1] < 0.5 {
			red_robot_state.AuraCount -= 0.5
		}

		// if the red robot's aura > 0, the robot can move
		red_robot_state.CanMove = red_robot_state.AuraCount > 0

		// when the red robot approaches the blue base, flag logic
		// if the blue flag is at the base AND the red robot is close to the blue flag base AND the red robot can move AND the red robot is not carrying a flag
		if game_states.BlueFlagAtBase && dist(red_robot_state.Location, game_states.BlueFlagBaseLocation) <= aura_range && red_robot_state.CanMove && !red_robot_state.HasBlueFlag && !red_robot_state.HasRedFlag {
			// give the blue flag to this robot
			red_robot_state.HasBlueFlag = true
			game_states.BlueFlagAtBase = false
		}

		// when the red robot approaches the red base with the flag, flag logic:
		// if the red robot has the flag AND the red base does not have the flag AND the red robot is at the base AND the red robot can move
		if red_robot_state.HasRedFlag && !game_states.RedFlagAtBase && dist(red_robot_state.Location, game_states.RedFlagBaseLocation) <= aura_range && red_robot_state.CanMove {
			// give the red flag to the base
			red_robot_state.HasRedFlag = false
			game_states.RedFlagAtBase = true
		}

		// when the red robot is on the red side with the blue flag, score
		if red_robot_state.HasBlueFlag && red_robot_state.Location[1] > 0.5 {
			red_robot_state.HasBlueFlag = false
			game_states.BlueFlagAtBase = true
			game_states.RedTeamScore += 1
		}


		// update the red robot state
		game_states.RedTeam[red_robot_id] = red_robot_state
	}

	// for each robot on the blue team...
	for blue_robot_id, blue_robot_state := range(game_states.BlueTeam) {
		// for each other robot on the blue team...
		for _, other_blue_robot_state := range(game_states.BlueTeam) {
			// if the robots are close (team mates), add aura, INCLUDING self
			if dist(blue_robot_state.Location, other_blue_robot_state.Location) <= aura_range {
				blue_robot_state.AuraCount += 1.0
			}
		}

		// if the blue robot is on the red side (y > 0.5), reduce aura by 0.5
		if blue_robot_state.Location[1] > 0.5 {
			blue_robot_state.AuraCount -= 0.5
		}

		// if the blue robot's aura > 0, the robot can move
		blue_robot_state.CanMove = blue_robot_state.AuraCount > 0

		// when the blue robot approaches the red base, flag logic
		// if the red flag is at the base AND the blue robot is close to the red flag base AND the blue robot can move AND the blue robot is not carrying a flag
		if game_states.RedFlagAtBase && dist(blue_robot_state.Location, game_states.RedFlagBaseLocation) <= aura_range && blue_robot_state.CanMove && !blue_robot_state.HasBlueFlag && !blue_robot_state.HasRedFlag {
			// give the blue flag to this robot
			blue_robot_state.HasRedFlag = true
			game_states.RedFlagAtBase = false
		}

		// when the blue robot approaches the blue base with the flag, flag logic:
		// if the blue robot has the flag AND the blue base does not have the flag && the blue robot is at the base AND the blue robot can move
		if blue_robot_state.HasBlueFlag && !game_states.BlueFlagAtBase && dist(blue_robot_state.Location, game_states.BlueFlagBaseLocation) <= aura_range && blue_robot_state.CanMove {
			// give the blue flag to the base
			blue_robot_state.HasBlueFlag = false
			game_states.BlueFlagAtBase = true
		}

		// when the blue robot is on the blue side with the red flag, score
		if blue_robot_state.HasRedFlag && blue_robot_state.Location[1] < 0.5 {
			blue_robot_state.HasRedFlag = false
			game_states.RedFlagAtBase = true
			game_states.BlueTeamScore += 1
		}
		
		// update the blue robot state
		game_states.BlueTeam[blue_robot_id] = blue_robot_state
	}

	// when the robot approaches an enemy robot who is carrying a red or blue flag, flag logic
	for _, robot_list := range robot_proximities {  // for each pair of proximal opposing robots
		red_robot_id := robot_list[0]  // pull the red robot
		blue_robot_id := robot_list[1]  // pull the blue robot
		// red robot takes flag from blue robot
		// if the blue robot has the (red flag OR blue flag) AND the red robot is not carrying a flag AND the blue robot cannot move AND the red robot can move
		if (game_states.BlueTeam[blue_robot_id].HasRedFlag || game_states.BlueTeam[blue_robot_id].HasBlueFlag) && !(game_states.RedTeam[red_robot_id].HasRedFlag || game_states.RedTeam[red_robot_id].HasBlueFlag) && !game_states.BlueTeam[blue_robot_id].CanMove && game_states.RedTeam[red_robot_id].CanMove {
			// transfer the flag from the blue robot to the red robot
			if game_states.BlueTeam[blue_robot_id].HasRedFlag {  // if transferring the red flag to the red robot
				if red_robot_state, ok := game_states.RedTeam[red_robot_id]; ok {  // pull the red team robot
					red_robot_state.HasRedFlag = true  // add the red flag to the blue robot
					game_states.RedTeam[red_robot_id] = red_robot_state  // put the struct back
				}
				if blue_robot_state, ok := game_states.BlueTeam[blue_robot_id]; ok {  // pull the blue team robot
					blue_robot_state.HasRedFlag = false  // remove the red flag from the blue robot
					game_states.BlueTeam[blue_robot_id] = blue_robot_state  // put the struct back
				}
			} else if game_states.BlueTeam[blue_robot_id].HasBlueFlag {  // if transferring the blue flag to the red robot
				if red_robot_state, ok := game_states.RedTeam[red_robot_id]; ok {  // pull the red team robot
					red_robot_state.HasBlueFlag = true  // remove the blue flag from the red robot
					game_states.RedTeam[red_robot_id] = red_robot_state  // put the struct back
				}
				if blue_robot_state, ok := game_states.BlueTeam[blue_robot_id]; ok {  // pull the blue team robot
					blue_robot_state.HasBlueFlag = false  // add the blue flag to the red robot
					game_states.BlueTeam[blue_robot_id] = blue_robot_state  // put the struct back
				}
			}
		}

		// blue robot takes flag from red robot
		// if the red robot has the (red flag OR blue flag) AND the blue robot is not carrying a flag AND the red robot cannot move AND the blue robot can move
		if (game_states.RedTeam[red_robot_id].HasRedFlag || game_states.RedTeam[red_robot_id].HasBlueFlag) && !(game_states.BlueTeam[blue_robot_id].HasRedFlag || game_states.BlueTeam[blue_robot_id].HasBlueFlag) && !game_states.RedTeam[red_robot_id].CanMove && game_states.BlueTeam[blue_robot_id].CanMove {
			// transfer the flag from the red robot to the blue robot
			
			if game_states.RedTeam[red_robot_id].HasRedFlag {  // if transferring the red flag to the blue robot
				if red_robot_state, ok := game_states.RedTeam[red_robot_id]; ok {  // pull the red team robot
					red_robot_state.HasRedFlag = false  // remove the red flag from the red robot
					game_states.RedTeam[red_robot_id] = red_robot_state  // put the struct back
				}
				if blue_robot_state, ok := game_states.BlueTeam[blue_robot_id]; ok {  // pull the blu3 team robot
					blue_robot_state.HasRedFlag = true  // add the red flag to the blue robot
					game_states.BlueTeam[blue_robot_id] = blue_robot_state  // put the struct back
				}
			} else if game_states.RedTeam[red_robot_id].HasBlueFlag {  // if transfering the blue flag to the blue robot
				if red_robot_state, ok := game_states.RedTeam[red_robot_id]; ok {  // pull the red team robot
					red_robot_state.HasBlueFlag = false  // remove the blue flag from the red robot
					game_states.RedTeam[red_robot_id] = red_robot_state  // put the struct back
				}
				if blue_robot_state, ok := game_states.BlueTeam[blue_robot_id]; ok {  // pull the blu3 team robot
					blue_robot_state.HasBlueFlag = true  // add the blue flag to the blue robot
					game_states.BlueTeam[blue_robot_id] = blue_robot_state  // put the struct back
				}
			}
		}
	}

	return
}

// handles the webserver
func main() {
	// create the game states JSON object
	game_states := GameStates{
		// initialize the red team
		RedTeam : map[string]CozmoState{
			"cozmo_1" : CozmoState{
				Location : [2]float64{.1, .2},
			},
			/*"cozmo_2" : CozmoState{
				Location : [2]float64{.3, .4},
			},
			"cozmo_3" : CozmoState{
				Location : [2]float64{.5, .6},
			},
			"cozmo_4" : CozmoState{
				Location : [2]float64{.7, .8},
			},*/
		},
		RedTeamOculusId : 0,
		RedFlagAtBase : true,
		RedFlagBaseLocation : [2]float64{.5, .1},
		
		// initialize the blue team
		BlueTeam : map[string]CozmoState{
			"cozmo_5" : CozmoState{
				Location : [2]float64{.9, .8},
			},
			/*"cozmo_6" : CozmoState{
				Location : [2]float64{.7, .6},
			},
			"cozmo_7" : CozmoState{
				Location : [2]float64{.5, .4},
			},
			"cozmo_8" : CozmoState{
				Location : [2]float64{.3, .2},
			},*/
		},
		BlueTeamOculusId : 0,
		BlueFlagAtBase : true,
		BlueFlagBaseLocation : [2]float64{.5, .9},
	}

	// handle registering an Oculus device to either the RedTeam or the BlueTeam
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		// run the ParseForm to pull the POST data, error if applicable
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "failure: ParseForm() err: %v", err)
			return
		}

		// create the response object
		response := map[string]int{}

		// get the Oculus' desired ID
		OculusId, err := strconv.Atoi(r.FormValue("OculusId"))
		if err != nil {
			// don't assign the user to any team and return an error
			response["Status"] = 400
			response["OculusId"] = -1
			response["Team"] = -1
		}

		fmt.Printf("BLUE ID%s", game_states.BlueTeamOculusId)
		fmt.Printf("RED ID%s", game_states.RedTeamOculusId)

		// prioritize assigning blue team first
		if game_states.BlueTeamOculusId == 0 && game_states.RedTeamOculusId != OculusId {
			// assign the user to the blue team
			game_states.BlueTeamOculusId = OculusId
			response["Status"] = 200
			response["OculusId"] = OculusId
			response["Team"] = 0
			fmt.Printf("\nRegistered Oculus ID %s to team BLUE", OculusId)
		} else if game_states.RedTeamOculusId == 0 && game_states.BlueTeamOculusId != OculusId {
			// assign the user to the blue team
			game_states.BlueTeamOculusId = OculusId
			response["Status"] = 200
			response["OculusId"] = OculusId
			response["Team"] = 1
			fmt.Printf("\nRegistered Oculus ID %s to team RED", OculusId)
		} else {
			// don't assign the user to any team and return an error
			response["Status"] = 400
			response["OculusId"] = -1
			response["Team"] = -1
			fmt.Printf("\nFailed to register Oculus ID %d, either both teams are full, ID is 0, or the ID is already used", OculusId)
		}

		// send the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)		
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	})

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
				team := r.FormValue("Team")  // team name
				robot := r.FormValue("Robot")  // robot name
				field := r.FormValue("Field")  // field name
				value := r.FormValue("Value")  // value for field
				oculus := r.FormValue("OculusId")  // value for the oculus id
				var processed_value [2]float64  // placeholder for the processed value
				team_verified := false  // whether the oculus ID given matches the team
				
				// convert the oculus ID from a string to an int
				oculusInt, err := strconv.Atoi(oculus)
				if err != nil {
					// don't assign the user to any team and return an error
					fmt.Printf("Oculus is not an int")
					return
				}

				// get the pointer to the team we are changing
				var p_team *map[string]CozmoState;  // initialize a pointer to the team we are dealing with
				if (team == "RedTeam") {
					p_team = &game_states.RedTeam;
					team_verified = oculusInt == game_states.RedTeamOculusId
				} else if (team == "BlueTeam") {
					p_team = &game_states.BlueTeam;
					team_verified = oculusInt == game_states.BlueTeamOculusId
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
				} else if field == "Waypoint" && team_verified {
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
					// send the error
					fmt.Fprintf(w, "failure: field must be Location or Waypoint")
					return
				}
				fmt.Fprintf(w, "success: team=%s, robot=%s, field=%s, value=%s", team, robot, field, value)
				
			default:
				fmt.Fprintf(w, "failure: need POST")  // notify that we only use POST (in case Glen or Jenna get it wrong)
			}
		return
    })

	// handle getting the game states (used by the VR)
    http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		engine(&game_states)
        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)		
		b, _ := json.Marshal(game_states)
		fmt.Fprintf(w, "%s", b)
		return
    })

	fmt.Printf("Running")
    log.Fatal(http.ListenAndServe(":1001", nil))
}
