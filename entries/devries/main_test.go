package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSolver(t *testing.T) {
	jsonVal := `{
  "name": "Maze #236 (10x10)",
  "mazePath": "/mazebot/mazes/ikTcNQMwKhux3bWjV3SSYKfyaVHcL0FXsvbwVGk5ns8",
  "startingPosition": [ 4, 3 ],
  "endingPosition": [ 3, 6 ],
  "message": "When you have figured out the solution, post it back to this url. See the exampleSolution for more information.",
  "exampleSolution": { "directions": "ENWNNENWNNS" },
  "map": [
    [ " ", " ", "X", " ", " ", " ", "X", " ", "X", "X" ],
    [ " ", "X", " ", " ", " ", " ", " ", " ", " ", " " ],
    [ " ", "X", " ", "X", "X", "X", "X", "X", "X", " " ],
    [ " ", "X", " ", " ", "A", " ", " ", " ", "X", " " ],
    [ " ", "X", "X", "X", "X", "X", "X", "X", " ", " " ],
    [ "X", " ", " ", " ", "X", " ", " ", " ", "X", " " ],
    [ " ", " ", "X", "B", "X", " ", "X", " ", "X", " " ],
    [ " ", " ", "X", " ", "X", " ", "X", " ", " ", " " ],
    [ "X", " ", "X", "X", "X", "X", "X", " ", "X", "X" ],
    [ "X", " ", " ", " ", " ", " ", " ", " ", "X", "X" ]
  ]
}`

	var m MazeMessage
	err := json.NewDecoder(strings.NewReader(jsonVal)).Decode(&m)
	if err != nil {
		t.Errorf("Error while parsing maze: %s", err)
	}

	maze := MapToMaze(m.Map)

	r := solve(maze, m.Start, m.End)
	expectation := "WWNNEEEEEEESSSSSSWWSSWWWWWWNNNNEES"
	if r != expectation {
		t.Errorf("Got %s, expected %s", r, expectation)
	}
}

func TestRandom(t *testing.T) {
	url := "https://api.noopschallenge.com/mazebot/random"

	m, err := GetMaze(url)
	if err != nil {
		t.Errorf("Unable to get maze: %s", err)
	}

	maze := MapToMaze(m.Map)

	r := solve(maze, m.Start, m.End)

	result, err := SendSolution(m, r)
	if err != nil {
		t.Errorf("Error sending result: %s", err)
	}

	if result.Result != "success" {
		t.Errorf("Result was not successful: %s", result.Message)
	}
}
