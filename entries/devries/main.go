package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	// url := "https://api.noopschallenge.com/mazebot/random?minSize=150&maxSize=200"

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFunc()
	mazePath, err := StartRace(ctx)
	if err != nil {
		panic(fmt.Errorf("Error starting race: %s", err))
	}

	var result SolutionResult

	for mazePath != "" {
		// Read Initial Value
		m, err := GetMaze(ctx, "https://api.noopschallenge.com"+mazePath)
		if err != nil {
			panic(fmt.Errorf("Unable to get maze: %s", err))
		}

		// Solve maze
		maze := MapToMaze(m.Map)

		r, err := solve(ctx, maze, m.Start, m.End)
		if err != nil {
			panic(fmt.Errorf("Error solving: %s", err))
		}
		fmt.Printf("Size: %dx%d\n", maze.XSize, maze.YSize)

		// Send solution
		result, err = SendSolution(ctx, m, r)
		if err != nil {
			panic(fmt.Errorf("Error sending result: %s", err))
		}
		fmt.Printf("Result: %s\n", result.Result)
		mazePath = result.NextMazePath
	}
	fmt.Printf("\nMessage: %s\nCertificate: %s\n", result.Message, result.Certificate)
}

// StartRace sends a starting message to the challenge server.
// The returned string is the path portion of the URL required to
// get the first maze in for the race.
func StartRace(ctx context.Context) (string, error) {
	client := http.Client{}
	loginMessage := map[string]string{
		"login": "devries",
	}

	reqBody, err := json.Marshal(loginMessage)
	if err != nil {
		return "", fmt.Errorf("Error generating JSON: %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.noopschallenge.com/mazebot/race/start", bytes.NewBuffer(reqBody))
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

// GetMaze obtains a maze from the challenge server.
// The url is the full URL of the maze endpoint. The returned
// MazeMessage object contains all the information about the maze.
func GetMaze(ctx context.Context, url string) (MazeMessage, error) {
	client := http.Client{}
	var m MazeMessage

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

// SendSolution sends the maze solution to the challenge server.
// The MazeMessage is required because it contains a reference to the
// solution endpoint, and the solution string is a series of directions N, S, E, and W
// required to solve the maze concatenated into a string. A SolutionResult object
// is returned.
func SendSolution(ctx context.Context, m MazeMessage, solution string) (SolutionResult, error) {
	solutionMessage := MazeSolution{solution}
	var result SolutionResult
	client := http.Client{}

	reqBody, err := json.Marshal(solutionMessage)
	if err != nil {
		return result, fmt.Errorf("Unable to generate solution JSON: %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.noopschallenge.com"+m.MazePath, bytes.NewBuffer(reqBody))
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

// A representation of the message returned by the GetMaze method.
type MazeMessage struct {
	Name            string       `json:"name"`                      // Name of the maze
	MazePath        string       `json:"mazePath"`                  // URL path to POST solution
	Start           Point        `json:"startingPosition"`          // Starting point
	End             Point        `json:"endingPosition"`            // Ending point
	Message         string       `json:"message"`                   // Message to accompany maze
	ExampleSolution MazeSolution `json:"exampleSolution",omitempty` // Sample of solution
	Map             [][]string   `json:"map"`                       // 2-D representation of Maze
}

// A representation of the solution sent to the maze endpoint
type MazeSolution struct {
	Directions string `json:"directions"` // Concatenated string containing maze solution
}

// A point in 2-D space
type Point struct {
	X int // Horizontal Position
	Y int // Vertical Position
}

// UnmarshalJSON unmarshals a point, which is stored as an array of integers, into a point.
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

// A representation of the response to a submitted solution.
type SolutionResult struct {
	Result                 string `json:"result,omitempty"`                 // "success" if solution is correct
	Message                string `json:"message,omitempty"`                // Message describing result
	ShortestSolutionLength int    `json:"shortestSolutionLength,omitempty"` // Shortest possible solution length
	SolutionLength         int    `json:"yourSolutionLength,omitempty"`     // Length of submitted solution
	Elapsed                int    `json:"elapsed,omitempty"`                // Seconds elapsed
	NextMazePath           string `json:"nextMaze,omitempty"`               // Next maze's URL path
	Certificate            string `json:"certificate,omitempty"`            // Certificate of completion, if race is complete
}

var directions = map[string]Point{
	"N": Point{0, -1},
	"S": Point{0, 1},
	"E": Point{1, 0},
	"W": Point{-1, 0},
}

// Representation of maze for maze solver.
type Maze struct {
	XSize  int              // Horizontal size of maze
	YSize  int              // Vertical size of maze
	Values map[Point]string // Value of each position. Space is empty, X is wall, A is start, B is end.
}

// MapToMaze creates Maze structure from 2-D representation sent by server.
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

// Representation of state of possible map step.
type State struct {
	Position Point
	Path     string
}

// Queue of states to evolve by subsequent steps.
type StateQueue []State

// Create a new state queue
func NewStateQueue() *StateQueue {
	r := []State{}

	return (*StateQueue)(&r)
}

// Add a new state to the state queue for later investigation.
func (sq *StateQueue) Add(s State) {
	*sq = append(*sq, s)
}

// Pop the first state off the state queue.
func (sq *StateQueue) Pop() State {
	var r State

	if len(*sq) > 0 {
		r = (*sq)[0]
		*sq = (*sq)[1:]
	}

	return r
}

// Check if additional states are available in queue
func (sq *StateQueue) Available() bool {
	if len(*sq) > 0 {
		return true
	} else {
		return false
	}
}

// Solve maze given Maze structure, and starting point and endpoint.
// Returns the steps required to solve the maze.
func solve(ctx context.Context, maze Maze, start Point, end Point) (string, error) {
	sq := NewStateQueue()
	seen := make(map[Point]bool)

	startingState := State{start, ""}
	sq.Add(startingState)
	seen[start] = true

	for sq.Available() {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("Premature cancellation: %s", ctx.Err())
		default:
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
					return newState.Path, nil
				} else {
					sq.Add(newState)
					seen[p] = true
				}
			}
		}
	}

	return "", fmt.Errorf("Solution not found")
}
