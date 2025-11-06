package gol

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"

	//"uk.ac.bris.cs/gameoflife/util"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	ioKeyPress <-chan rune
}

var (
	aliveCount int
	turn       int
	paused     = false
	save       = false
	terminate  = false
	mu         sync.Mutex
)

func checkKeyPress(c distributorChannels, pausedChannel chan<- bool, filename string, height, width int) {
	for keyPress := range c.ioKeyPress {
		switch keyPress {
		case 'p':
			/*mu.Lock()
			if paused {
				paused = false
				pausedChannel <- false
			} else {
				paused = true
				pausedChannel <- true
			}
			mu.Unlock()*/
		case 's':
			/*if paused {
				pausedChannel <- true
			}*/
			var res stubs.Response
			clientTemp, err := rpc.Dial("tcp", "localhost:8030")
			req := stubs.Request{}
			err = clientTemp.Call(stubs.SaveWorld, req, &res)
			if err != nil {
				fmt.Println("error:", err)
			}
			filename = fmt.Sprintf("%sx%d", filename, res.CurrentTurn)
			saveWorld(res, filename, c, height, width)

		case 'q':
			/*if paused {
				pausedChannel <- true
			}
			mu.Lock()
			terminate = true
			mu.Unlock()
			ticker.Stop()*/
			var res stubs.Response
			clientTemp, err := rpc.Dial("tcp", "localhost:8030")
			req := stubs.Request{}
			err = clientTemp.Call(stubs.TerminateWorld, req, &res)
			if err != nil {
				fmt.Println("error:", err)
			}
			filename = fmt.Sprintf("%sx%d", filename, res.CurrentTurn)
			saveWorld(res, filename, c, height, width)
			c.events <- FinalTurnComplete{res.CurrentTurn, res.AliveCells}
			c.events <- StateChange{res.CurrentTurn, Quitting}

		}
	}
}

func saveWorld(res stubs.Response, filename string, c distributorChannels, height, width int) {

	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c.ioOutput <- res.FinalWorld[y][x]
		}
	}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{res.CurrentTurn, filename}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	height := p.ImageHeight
	width := p.ImageWidth

	currentWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image
	for i := range currentWorld {
		currentWorld[i] = make([]byte, width)
	}
	command := ioCommand(ioInput)
	c.ioCommand <- command

	filename := fmt.Sprintf("%dx%d", height, width)

	c.ioFilename <- filename

	//reads each byte for initial world from IOinput channel
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			b, channelOpen := <-c.ioInput
			if !channelOpen {
				fmt.Println("channel closed")
				return
			}
			currentWorld[y][x] = b
		}
	}
	pausedChannel := make(chan bool, 1)
	go checkKeyPress(c, pausedChannel, filename, height, width)

	turn := 0
	c.events <- StateChange{turn, Executing}

	// TODO: Execute all turns of the Game of Life.

	//RPC dial to RUN GoL logic in server
	client, err := rpc.Dial("tcp", "localhost:8030")
	if err != nil {
		fmt.Println(err)
	}
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println("error closing client: ", err)
		}
	}(client)

	//RPC dial to report alive cells every two seconds
	aliveCellsReport, err := rpc.Dial("tcp", "localhost:8030")
	if err != nil {
		fmt.Println("error connecting 2nd rpc call:", err)
	}
	defer func(aliveCellsReport *rpc.Client) {
		err := aliveCellsReport.Close()
		if err != nil {
			fmt.Println("error closing alive cells report:", err)
		}
	}(aliveCellsReport)

	//Create a request struct to pass to server
	req := stubs.Request{
		StartWorld: currentWorld,
		Turns:      p.Turns,
		Height:     height,
		Width:      width,
	}

	//GO function that runs constantly and sends AliveCells update to events channel every 2 seconds
	stop := make(chan struct{}) //channel to tell when ticker stops
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				//every two seconds, ask server to report alive cells
				//create a new response struct to store the alivecells and currentturn from the method
				var AliveCellsRes stubs.Response
				//Call the AliveCellsMethod in server with request and response
				err := aliveCellsReport.Call(stubs.AliveCellsMethod, req, &AliveCellsRes)
				if err != nil {
					fmt.Println("error calling AliveCellsMethod", err)
					continue
				}

				//debug print statements to see if alivecells is being updated
				fmt.Println("alive cells after rpc call: ", len(AliveCellsRes.AliveCells))
				fmt.Println("current turn: ", AliveCellsRes.CurrentTurn)

				//update c.events channel with the alivecells count
				c.events <- AliveCellsCount{
					CompletedTurns: AliveCellsRes.CurrentTurn,
					CellsCount:     len(AliveCellsRes.AliveCells),
				}

			case <-stop: //exit go routine when stop is received
				return

			}
		}
	}()

	//blocking rpc call to run all turns of GOL
	var res stubs.Response
	err = client.Call(stubs.ExecGolMethod, req, &res)
	if err != nil {
		fmt.Println("Error executing GolMethod", err)
	}

	close(stop) //stops the ticker goroutine
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//sends final state to events channel
	c.events <- FinalTurnComplete{turn, res.AliveCells}

	filename = fmt.Sprintf("%sx%d", filename, p.Turns)

	fmt.Println("turns:", p.Turns)
	fmt.Println("filename: ", filename)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c.ioOutput <- res.FinalWorld[y][x]
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	//distributor is quitting so notify events channel
	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
