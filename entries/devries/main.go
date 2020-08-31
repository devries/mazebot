package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	// url := "https://api.noopschallenge.com/mazebot/random?minSize=150&maxSize=200"

	mazePath, err := StartRace()
	if err != nil {
		panic(fmt.Errorf("Error starting race: %s", err))
	}

	var result SolutionResult

	for mazePath != "" {
		// Read Initial Value
		m, err := GetMaze("https://api.noopschallenge.com" + mazePath)
		if err != nil {
			panic(fmt.Errorf("Unable to get maze: %s", err))
		}

		// Solve maze
		maze := MapToMaze(m.Map)

		r := solve(maze, m.Start, m.End)
		fmt.Printf("Size: %dx%d\n", maze.XSize, maze.YSize)

		// Send solution
		result, err = SendSolution(m, r)
		if err != nil {
			panic(fmt.Errorf("Error sending result: %s", err))
		}
		fmt.Printf("Result: %s\n", result.Result)
		mazePath = result.NextMazePath
	}
	fmt.Printf("\nMessage: %s\nCertificate: %s\n", result.Message, result.Certificate)
}

func StartRace() (string, error) {
	client := http.Client{}
	loginMessage := map[string]string{
		"login": "devries",
	}

	reqBody, err := json.Marshal(loginMessage)
	if err != nil {
		return "", fmt.Errorf("Error generating JSON: %s", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.noopschallenge.com/mazebot/race/start", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("Error creating request: %s", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error making request: %s", err)
	}
	defer res.Body.Close()

	var r SolutionResult
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", fmt.Errorf("Error decoding JSON: %s", err)
	}

	return r.NextMazePath, nil
}

func GetMaze(url string) (MazeMessage, error) {
	client := http.Client{}
	var m MazeMessage

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return m, fmt.Errorf("Error creating request: %s", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return m, fmt.Errorf("Error making request: %s", err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&m)
	if err != nil {
		return m, fmt.Errorf("Unable to parse maze: %s", err)
	}

	return m, nil
}

func SendSolution(m MazeMessage, solution string) (SolutionResult, error) {
	solutionMessage := MazeSolution{solution}
	var result SolutionResult
	client := http.Client{}

	reqBody, err := json.Marshal(solutionMessage)
	if err != nil {
		return result, fmt.Errorf("Unable to generate solution JSON: %s", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.noopschallenge.com"+m.MazePath, bytes.NewBuffer(reqBody))
	if err != nil {
		return result, fmt.Errorf("Unable to generate post request: %s", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("Unable to make post request: %s", err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return result, fmt.Errorf("Unable to parse result: %s", err)
	}

	return result, nil
}

type MazeMessage struct {
	Name            string       `json:"name"`
	MazePath        string       `json:"mazePath"`
	Start           Point        `json:"startingPosition"`
	End             Point        `json:"endingPosition"`
	Message         string       `json:"message"`
	ExampleSolution MazeSolution `json:"exampleSolution",omitempty`
	Map             [][]string   `json:"map"`
}

type MazeSolution struct {
	Directions string `json:"directions"`
}

type Point struct {
	X int
	Y int
}

func (p *Point) UnmarshalJSON(b []byte) (err error) {
	tmp := []interface{}{&p.X, &p.Y}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	if l := len(tmp); l != 2 {
		return fmt.Errorf("Point arrays should have length 2, this one has length %d", l)
	}

	return nil
}

type SolutionResult struct {
	Result                 string `json:"result,omitempty"`
	Message                string `json:"message,omitempty"`
	ShortestSolutionLength int    `json:"shortestSolutionLength,omitempty"`
	SolutionLength         int    `json:"yourSolutionLength,omitempty"`
	Elapsed                int    `json:"elapsed,omitempty"`
	NextMazePath           string `json:"nextMaze,omitempty"`
	Certificate            string `json:"certificate,omitempty"`
}

var directions = map[string]Point{
	"N": Point{0, -1},
	"S": Point{0, 1},
	"E": Point{1, 0},
	"W": Point{-1, 0},
}

type Maze struct {
	XSize  int
	YSize  int
	Values map[Point]string
}

func MapToMaze(mazeMap [][]string) Maze {
	var result Maze
	result.Values = make(map[Point]string)
	result.YSize = len(mazeMap)
	result.XSize = len(mazeMap[0])

	for j, row := range mazeMap {
		for i, block := range row {
			result.Values[Point{i, j}] = block
		}
	}

	return result
}

type State struct {
	Position Point
	Path     string
}

type StateQueue []State

func NewStateQueue() *StateQueue {
	r := []State{}

	return (*StateQueue)(&r)
}

func (sq *StateQueue) Add(s State) {
	*sq = append(*sq, s)
}

func (sq *StateQueue) Pop() State {
	var r State

	if len(*sq) > 0 {
		r = (*sq)[0]
		*sq = (*sq)[1:]
	}

	return r
}

func (sq *StateQueue) Available() bool {
	if len(*sq) > 0 {
		return true
	} else {
		return false
	}
}

func solve(maze Maze, start Point, end Point) string {
	sq := NewStateQueue()
	seen := make(map[Point]bool)

	startingState := State{start, ""}
	sq.Add(startingState)
	seen[start] = true

	for sq.Available() {
		state := sq.Pop()
		for dName, d := range directions {
			p := Point{state.Position.X + d.X, state.Position.Y + d.Y}

			newState := State{p, state.Path + dName}

			if seen[p] {
				// Already seen this point
				continue
			} else if maze.Values[p] == "X" {
				// This is a wall
				continue
			} else if p.X >= maze.XSize || p.X < 0 {
				// Out of bounds in X
				continue
			} else if p.Y >= maze.YSize || p.Y < 0 {
				// Out of bounds in Y
				continue
			} else if p == end {
				// This is the goal
				return newState.Path
			} else {
				sq.Add(newState)
				seen[p] = true
			}
		}
	}

	return ""
}
