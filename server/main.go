
// Code Citations:
// strtoint: https://stackoverflow.com/questions/4278430
// foreach key in map: https://stackoverflow.com/questions/1841443/
// foreach element in array: https://yourbasic.org/golang/for-loop-range-array-slice-map-channel/
// assigning field to struct: https://stackoverflow.com/questions/42605337/
// JSON remarshalling: https://stackoverflow.com/questions/51795678/
// JSON marshalling of sync.Map: https://play.golang.org/p/PdSjIcV8iJh


package main

import (
	"fmt"
	"encoding/json"
    "log"
    "net/http"
	"strconv"
	"math"
	"sync"
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
	RedTeam sync.Map  // states of the red team cozmos
	RedTeamOculusId int  // ID number of the red player's VR headset
	RedFlagAtBase bool  // whether the red team's flag is at their base
	RedFlagBaseLocation [2]float64  // x/y location of the flag base
	RedTeamScore int  // score of the red team

	BlueTeam sync.Map  // states of the blue team cozmos
	BlueTeamOculusId int // ID number of the blue player's VR headset
	BlueFlagAtBase bool // whether the blue team's flag is at their base
	BlueFlagBaseLocation [2]float64  // x/y location of the flag base
	BlueTeamScore int  // score of the blue team
}

// distance calculation
func dist(a [2]float64, b [2]float64) float64 {
	return math.Sqrt(math.Pow(a[0] - b[0], 2) + math.Pow(a[1] - b[1], 2))
}

func reset_gamestate() GameStates{
	// create the game states JSON object
	game_states := GameStates{
		// initialize the red team attributes
		RedTeam : sync.Map{},
		RedTeamOculusId : 0,
		RedFlagAtBase : true,
		RedFlagBaseLocation : [2]float64{.5, .1},
		
		// initialize the blue team attributes
		BlueTeam : sync.Map{},
		BlueTeamOculusId : 0,
		BlueFlagAtBase : true,
		BlueFlagBaseLocation : [2]float64{.5, .9},
	}

	// intialize the red team robots
	game_states.RedTeam.Store("cozmo_1", CozmoState{
		Location : [2]float64{.1, .5},
	})
	/*game_states.RedTeam.Store("cozmo_2", CozmoState{
		Location : [2]float64{.3, .4},
	})
	game_states.RedTeam.Store("cozmo_3", CozmoState{
		Location : [2]float64{.5, .6},
	})
	game_states.RedTeam.Store("cozmo_4", CozmoState{
		Location : [2]float64{.7, .8},
	})*/

	// intiialize the blue team robots
	/*game_states.BlueTeam.Store("cozmo_5", CozmoState{
		Location : [2]float64{.9, .5},
	})
	game_states.BlueTeam.Store("cozmo_6", CozmoState{
		Location : [2]float64{.7, .6},
	})
	game_states.BlueTeam.Store("cozmo_7", CozmoState{
		Location : [2]float64{.5, .4},
	})
	game_states.BlueTeam.Store("cozmo_8", CozmoState{
		Location : [2]float64{.3, .2},
	})*/

	log.Println("reset game state")
	return game_states
}

// uses the game states object to determine the HasFlag(s), CanMove, and FlagAtBase bools
func engine(game_states *GameStates) {
	// determine auras between each robot
	aura_range := 0.1  // range at which robots connect
	
	robot_proximities := [][2]string{}  // names of robots from opposite teams in proximity to each other, so we can parse for flag transfers after calculating auras

	// reset aura counts for each robot on each team
	game_states.RedTeam.Range(func(red_robot_id, red_robot_state interface{}) bool {
		_red_robot_state := red_robot_state.(CozmoState)
		_red_robot_state.AuraCount = 0.0
		game_states.RedTeam.Store(red_robot_id, _red_robot_state)
		return true
	})

	game_states.BlueTeam.Range(func(blue_robot_id, blue_robot_state interface{}) bool {
		_blue_robot_state := blue_robot_state.(CozmoState)
		_blue_robot_state.AuraCount = 0.0
		game_states.BlueTeam.Store(blue_robot_id, _blue_robot_state)
		return true
	})


	// for each robot on the red team...
	game_states.RedTeam.Range(func(red_robot_id, red_robot_state interface{}) bool {
		_red_robot_state := red_robot_state.(CozmoState)
		_red_robot_AuraCount := 0.0
		
		// for each other robot on the red team...
		game_states.RedTeam.Range(func(_ , other_red_robot_state interface{}) bool {
			_other_red_robot_state := other_red_robot_state.(CozmoState)
			// if the robots are close (team mates), add an aura, INCLUDING self
			if dist(_red_robot_state.Location, _other_red_robot_state.Location) <= aura_range {
				_red_robot_AuraCount += 1.0
			}
			return true
		})

		// for each robot on the blue team...
		robot_proximities = nil  // reset the proximity list
		game_states.BlueTeam.Range(func(blue_robot_id, blue_robot_state interface{}) bool {
			_blue_robot_state := blue_robot_state.(CozmoState)
			_blue_robot_AuraCount := 0.0
			// if the robots are close (enemies), reduce aura on both robots
			if dist(_red_robot_state.Location, _blue_robot_state.Location) <= aura_range {
				robot_proximities = append(robot_proximities, [2]string{red_robot_id.(string), blue_robot_id.(string)})  // note that these robots are proximal for red
			}

			// update the blue robot state
			_blue_robot_state.AuraCount = _blue_robot_AuraCount
			game_states.BlueTeam.Store(blue_robot_id.(string), _blue_robot_state)
			return true
		})

		// if the red robot is on the blue side (y < 0.5), reduce aura by 0.5
		if _red_robot_state.Location[1] < 0.5 {
			_red_robot_AuraCount -= 0.5
		}

		// if the red robot's aura > 0, the robot can move
		_red_robot_state.CanMove = _red_robot_AuraCount > 0

		// when the red robot approaches the blue base, flag logic
		// if the blue flag is at the base AND the red robot is close to the blue flag base AND the red robot can move AND the red robot is not carrying a flag
		if game_states.BlueFlagAtBase && dist(_red_robot_state.Location, game_states.BlueFlagBaseLocation) <= aura_range && _red_robot_state.CanMove && !_red_robot_state.HasBlueFlag && !_red_robot_state.HasRedFlag {
			// give the blue flag to this robot
			_red_robot_state.HasBlueFlag = true
			game_states.BlueFlagAtBase = false
		}

		// when the red robot approaches the red base with the flag, flag logic:
		// if the red robot has the flag AND the red base does not have the flag AND the red robot is at the base AND the red robot can move
		if _red_robot_state.HasRedFlag && !game_states.RedFlagAtBase && dist(_red_robot_state.Location, game_states.RedFlagBaseLocation) <= aura_range && _red_robot_state.CanMove {
			// give the red flag to the base
			_red_robot_state.HasRedFlag = false
			game_states.RedFlagAtBase = true
		}

		// when the red robot is on the red side with the blue flag, score
		if _red_robot_state.HasBlueFlag && _red_robot_state.Location[1] > 0.5 {
			_red_robot_state.HasBlueFlag = false
			game_states.BlueFlagAtBase = true
			game_states.RedTeamScore += 1
		}

		// update the red robot state
		_red_robot_state.AuraCount = _red_robot_AuraCount
		game_states.RedTeam.Store(red_robot_id.(string), _red_robot_state)
		return true
	})

	// for each robot on the blue team...
	game_states.BlueTeam.Range(func(blue_robot_id, blue_robot_state interface{}) bool {
		_blue_robot_state := blue_robot_state.(CozmoState)
		// for each other robot on the blue team...
		game_states.BlueTeam.Range(func(_, other_blue_robot_state interface{}) bool {
			_other_blue_robot_state := other_blue_robot_state.(CozmoState)
			// if the robots are close (team mates), add aura, INCLUDING self
			if dist(_blue_robot_state.Location, _other_blue_robot_state.Location) <= aura_range {
				_blue_robot_state.AuraCount += 1.0
			}
			return true
		})

		// if the blue robot is on the red side (y > 0.5), reduce aura by 0.5
		if _blue_robot_state.Location[1] > 0.5 {
			_blue_robot_state.AuraCount -= 0.5
		}

		// if the blue robot's aura > 0, the robot can move
		_blue_robot_state.CanMove = _blue_robot_state.AuraCount > 0

		// when the blue robot approaches the red base, flag logic
		// if the red flag is at the base AND the blue robot is close to the red flag base AND the blue robot can move AND the blue robot is not carrying a flag
		if game_states.RedFlagAtBase && dist(_blue_robot_state.Location, game_states.RedFlagBaseLocation) <= aura_range && _blue_robot_state.CanMove && !_blue_robot_state.HasBlueFlag && !_blue_robot_state.HasRedFlag {
			// give the blue flag to this robot
			_blue_robot_state.HasRedFlag = true
			game_states.RedFlagAtBase = false
		}

		// when the blue robot approaches the blue base with the flag, flag logic:
		// if the blue robot has the flag AND the blue base does not have the flag && the blue robot is at the base AND the blue robot can move
		if _blue_robot_state.HasBlueFlag && !game_states.BlueFlagAtBase && dist(_blue_robot_state.Location, game_states.BlueFlagBaseLocation) <= aura_range && _blue_robot_state.CanMove {
			// give the blue flag to the base
			_blue_robot_state.HasBlueFlag = false
			game_states.BlueFlagAtBase = true
		}

		// when the blue robot is on the blue side with the red flag, score
		if _blue_robot_state.HasRedFlag && _blue_robot_state.Location[1] < 0.5 {
			_blue_robot_state.HasRedFlag = false
			game_states.RedFlagAtBase = true
			game_states.BlueTeamScore += 1
		}
		
		// update the blue robot state
		game_states.BlueTeam.Store(blue_robot_id.(string), _blue_robot_state)
		return true
	})

	// when the robot approaches an enemy robot who is carrying a red or blue flag, flag logic
	for _, robot_list := range robot_proximities {  // for each pair of proximal opposing robots
		red_robot_id := robot_list[0]  // pull the red robot
		red_robot_state, _ := game_states.RedTeam.Load(red_robot_id)
		_red_robot_state := red_robot_state.(CozmoState)

		blue_robot_id := robot_list[1]  // pull the blue robot
		blue_robot_state, _ := game_states.BlueTeam.Load(blue_robot_id)
		_blue_robot_state := blue_robot_state.(CozmoState)

		// red robot takes flag from blue robot
		// if the blue robot has the (red flag OR blue flag) AND the red robot is not carrying a flag AND the blue robot cannot move AND the red robot can move
		if (_blue_robot_state.HasRedFlag || _blue_robot_state.HasBlueFlag) && !(_red_robot_state.HasRedFlag || _red_robot_state.HasBlueFlag) && !_blue_robot_state.CanMove && _red_robot_state.CanMove {
			// transfer the flag from the blue robot to the red robot
			if _blue_robot_state.HasRedFlag {  // if transferring the red flag to the red robot
				_red_robot_state.HasRedFlag = true  // add the red flag to the blue robot
				_blue_robot_state.HasRedFlag = false  // remove the red flag from the blue robot
			} else if _blue_robot_state.HasBlueFlag {  // if transferring the blue flag to the red robot
				_red_robot_state.HasBlueFlag = true  // remove the blue flag from the red robot
				_blue_robot_state.HasBlueFlag = false  // add the blue flag to the red robot
			}
		}

		// blue robot takes flag from red robot
		// if the red robot has the (red flag OR blue flag) AND the blue robot is not carrying a flag AND the red robot cannot move AND the blue robot can move
		if (_red_robot_state.HasRedFlag || _red_robot_state.HasBlueFlag) && !(_blue_robot_state.HasRedFlag || _blue_robot_state.HasBlueFlag) && !_red_robot_state.CanMove && _blue_robot_state.CanMove {
			// transfer the flag from the red robot to the blue robot
			if _red_robot_state.HasRedFlag {  // if transferring the red flag to the blue robot
				_red_robot_state.HasRedFlag = false  // remove the red flag from the red robot
				_blue_robot_state.HasRedFlag = true  // add the red flag to the blue robot
			} else if _red_robot_state.HasBlueFlag {  // if transfering the blue flag to the blue robot
				_red_robot_state.HasBlueFlag = false  // remove the blue flag from the red robot
				_blue_robot_state.HasBlueFlag = true  // add the blue flag to the blue robot
			}
		}
	}

	return
}

// handles the webserver
func main() {
	log.Println("starting server")
	game_states := reset_gamestate()

	// handle registering an Oculus device to either the RedTeam or the BlueTeam
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		// run the ParseForm to pull the POST data, error if applicable
		if err := r.ParseForm(); err != nil {
			log.Println("/register failure: ParseForm() err: %v", err)
			return
		}

		// create the response object
		response := map[string]int{}

		// get the Oculus' desired ID
		OculusId, err := strconv.Atoi(r.FormValue("OculusId"))
		if err != nil {
			// don't assign the user to any team and return an error
			log.Println("/register OculusId is not an integer", OculusId)
			response["Status"] = 400
			response["OculusId"] = -1
			response["Team"] = -1
		}

		// prioritize assigning blue team first
		if game_states.BlueTeamOculusId == 0 && game_states.RedTeamOculusId != OculusId {
			// assign the user to the blue team
			game_states.BlueTeamOculusId = OculusId
			response["Status"] = 200
			response["OculusId"] = OculusId
			response["Team"] = 0
			log.Println("/register registered Oculus ID", OculusId, "to team BLUE")
			fmt.Printf("\nRegistered Oculus ID %s to team BLUE", OculusId)
		} else if game_states.RedTeamOculusId == 0 && game_states.BlueTeamOculusId != OculusId {
			// assign the user to the blue team
			game_states.BlueTeamOculusId = OculusId
			response["Status"] = 200
			response["OculusId"] = OculusId
			response["Team"] = 1
			log.Println("/register registered Oculus ID", OculusId, "to team RED")
		} else {
			// don't assign the user to any team and return an error
			response["Status"] = 400
			response["OculusId"] = -1
			response["Team"] = -1
			log.Println("/register unable to register Oculus to a team. RED:", game_states.RedTeamOculusId, ", BLUE:", game_states.BlueTeamOculusId, "OculusId:", OculusId)
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
					log.Println("/put failure: ParseForm() err: %v", err)
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
					log.Println("/put OculusId", oculus, "is not an int")
					fmt.Printf("Oculus is not an int")
					return
				}

				// get the pointer to the team we are changing
				var p_team *sync.Map;  // initialize a pointer to the team we are dealing with
				if (team == "RedTeam") {
					p_team = &game_states.RedTeam
					team_verified = oculusInt == game_states.RedTeamOculusId
				} else if (team == "BlueTeam") {
					p_team = &game_states.BlueTeam
					team_verified = oculusInt == game_states.BlueTeamOculusId
				} else {
					log.Println("/put failure: team", team, "must be RedTeam or BlueTeam")
					fmt.Fprintf(w, "failure: team must be RedTeam or BlueTeam")
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
					robotObject, _ := (*p_team).Load(robot)
					_robotObject := robotObject.(CozmoState)
					_robotObject.Location = processed_value
					(*p_team).Store(robot, _robotObject)
					log.Println("/put updated location for team", team, "and robot", robot)
				} else if field == "Waypoint" && team_verified {
					// break the waypoint value into an array
					err := json.Unmarshal([]byte(value), &processed_value)
					if err != nil {
						log.Fatal(err)
					}
					// update the robot waypoint
					robotObject, _ := (*p_team).Load(robot)
					_robotObject := robotObject.(CozmoState)
					_robotObject.Waypoint = processed_value
					(*p_team).Store(robot, _robotObject)
					log.Println("/put updated waypoint for team", team, "and robot", robot)
				} else {
					// send the error
					log.Println("/put failure: field", field, "must be Location or Waypoint", field)
					fmt.Fprintf(w, "failure: field must be Location or Waypoint")
					return
				}
				log.Println("/put team=", team, ", robot=", robot, ", field=", field, ", value=", value)
				fmt.Fprintf(w, "success: team=%s, robot=%s, field=%s, value=%s", team, robot, field, value)
				
			default:
				log.Println("/put failure: need POST")
				fmt.Fprintf(w, "failure: need POST")  // notify that we only use POST (in case Glen or Jenna get it wrong)
			}
		return
    })

	// handle getting the game states (used by the VR)
    http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		// update the engine
		engine(&game_states)
	
		// convert the sync maps to JSON
		RedTeamJSON := make(map[string]interface{})  // for red team
		game_states.RedTeam.Range(func( k interface{}, v interface{}) bool {
			RedTeamJSON[k.(string)] = v
			return true
		})

		BlueTeamJSON := make(map[string]interface{})  // for blue team
		game_states.BlueTeam.Range(func( k interface{}, v interface{}) bool {
			BlueTeamJSON[k.(string)] = v
			return true
		})

		// JSON marshall the game states
		b, _ := json.Marshal(game_states)

		// unmarshall into [string]interface
		var m map[string]interface{}
		err := json.Unmarshal(b, &m)
		if err != nil {
			log.Printf("/get Unable to unmarshall")
			fmt.Printf("Unable to unmarshall")
		}

		// add the RedTeam and BlueTeam
		m["RedTeam"] = RedTeamJSON
		m["BlueTeam"] = BlueTeamJSON

		// remarshall the JSON
		b, _ = json.Marshal(m)
	
        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)		
		
		fmt.Fprintf(w, "%s", b)
		return
    })

	// resets the game state
	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		// run the ParseForm to pull the POST data, error if applicable
		if err := r.ParseForm(); err != nil {
			log.Println("/reset failure: ParseForm() err: %v", err)
			fmt.Fprintf(w, "failure: ParseForm() err: %v", err)
			return
		}

		if r.FormValue("key") == "RAIL" {
			game_states = reset_gamestate()
			fmt.Fprintf(w, "reset game state")
		} else {
			fmt.Fprintf(w, "incorrect reset key")
		}
	})

	fmt.Printf("Running")
    log.Fatal(http.ListenAndServe(":1002", nil))
}
