package gol

import (
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"sync"
	"time"

	//"uk.ac.bris.cs/gameoflife/util"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
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
	paused = false
	kPress = false
	mu     sync.Mutex
)

func checkKeyPress(c distributorChannels, totalTurns int, filename string, height, width int, stopAll chan struct{}) {
	for {
		select {
		case <-stopAll: //channel listening for a shutdown signal, if any method closes stopAll then keypresses will stop running
			return

		case keyPress, channelOpen := <-c.ioKeyPress: //listening for keypresses p s q k
			if !channelOpen { //if channel closes then stop running
				return
			}
			switch keyPress {
			case 'p': //key press is p
				mu.Lock()
				if paused {
					paused = false
				} else {
					paused = true
				}
				mu.Unlock()

				var res stubs.Response
				pauseClient, err := rpc.Dial("tcp", "localhost:8030")
				req := stubs.Request{}
				err = pauseClient.Call(stubs.PauseWorld, req, &res) //call the broker and tell server to pause world
				if err != nil {
					fmt.Println("pauseworld error:", err)
				}

				if paused {
					c.events <- StateChange{res.CurrentTurn, Paused} //update sdl
				} else {
					c.events <- StateChange{res.CurrentTurn, Executing}
				}

			case 's': //key press is s
				var res stubs.Response
				saveClient, err := rpc.Dial("tcp", "localhost:8030")
				req := stubs.Request{}
				err = saveClient.Call(stubs.SaveWorld, req, &res) //call the broker and tell server we want to save output of current world
				if err != nil {
					fmt.Println("save world error:", err)
				}
				filename = fmt.Sprintf("%sx%d", filename, res.CurrentTurn)
				saveWorld(res, filename, c, height, width)

			case 'q':
				var res stubs.Response
				quitClient, err := rpc.Dial("tcp", "localhost:8030")
				req := stubs.Request{}
				err = quitClient.Call(stubs.TerminateWorld, req, &res) //call broker and tell server we want to terminate the world
				if err != nil {
					fmt.Println("quit q error:", err)
				}
				filename = fmt.Sprintf("%sx%d", filename, res.CurrentTurn)
				saveWorld(res, filename, c, height, width) //save output of current world

				select { //close stopAll if it hasn't been closed already
				case <-stopAll:
				//before i added this q wouldnt work properly as multiple channels were trying to shut down (close(stopAll))
				default:
					close(stopAll)
				}

				c.events <- FinalTurnComplete{res.CurrentTurn, res.AliveCells} //update sdl
				c.events <- StateChange{res.CurrentTurn, Quitting}

			case 'k':
				var res stubs.Response
				terminateClient, err := rpc.Dial("tcp", "localhost:8030")
				req := stubs.Request{}
				err = terminateClient.Call(stubs.TerminateClient, req, &res) //tell broker to shut down all server activity
				if err != nil {
				}
				filename = fmt.Sprintf("%sx%d", filename, res.CurrentTurn)
				mu.Lock()
				kPress = true
				mu.Unlock()

				c.events <- FinalTurnComplete{res.CurrentTurn, res.AliveCells} //update sdl
				c.events <- StateChange{res.CurrentTurn, Quitting}
				//saveWorld(res, filename, c, height, width)
			}
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

	stopAll := make(chan struct{})

	mu.Lock()
	paused = false
	kPress = false
	mu.Unlock()

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

	//reads each byte for initial world from IO input channel
	var initialCells []util.Cell
	//initialCells = make([]util.Cell, 0)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			b, channelOpen := <-c.ioInput
			if !channelOpen {
				fmt.Println("channel closed")
				return
			}
			currentWorld[y][x] = b
			if b == 255 {
				initialCells = append(initialCells, util.Cell{x, y})
			}
		}
	}
	go checkKeyPress(c, p.Turns, filename, height, width, stopAll)

	turn := 0
	c.events <- CellsFlipped{turn, initialCells}
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
		fmt.Println("error connecting alive cells rpc call:", err)
	}
	defer func(aliveCellsReport *rpc.Client) {
		err := aliveCellsReport.Close()
		if err != nil {
			fmt.Println("error closing alive cells report:", err)
		}
	}(aliveCellsReport)

	flipReport, err := rpc.Dial("tcp", "localhost:8030")
	if err != nil {
		fmt.Println("error connecting flip rpc call:", err)
	}
	defer func(flipReport *rpc.Client) {
		err := flipReport.Close()
		if err != nil {
			fmt.Println("error closing flip report:", err)
		}
	}(flipReport)

	resetClient, err := rpc.Dial("tcp", "localhost:8030")
	if err != nil {
		fmt.Println("error connecting reset rpc call:", err)
	}
	defer func(resetClient *rpc.Client) {
		err := resetClient.Close()
		if err != nil {
			fmt.Println("error closing reset client:", err)
		}
	}(resetClient)

	//Create a request struct to pass to server
	req := stubs.Request{
		StartWorld: currentWorld,
		Turns:      p.Turns,
		Height:     height,
		Width:      width,
		StartY:     0,
		EndY:       height,
		Threads:    p.Threads,
	}

	var resetRes stubs.Response
	if err := resetClient.Call(stubs.ResetMethod, req, &resetRes); err != nil {
		fmt.Println("error with calling reset method:", err)
	}

	//GO function that runs constantly and sends AliveCells update to events channel every 2 seconds
	stopAlive := make(chan struct{}) //channel to tell when ticker stops
	go func() {
		defer close(stopAlive)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopAll: //exit go routine when stop is received
				return

			case <-ticker.C:
				//every two seconds, ask server to report alive cells
				//create a new response struct to store the aliveCells and currentTurn from the method
				var AliveCellsRes stubs.Response
				//Call the AliveCellsMethod in server with request and response
				err := aliveCellsReport.Call(stubs.AliveCellsBrokers, req, &AliveCellsRes)
				if err != nil {
					fmt.Println("error calling AliveCellsMethod", err)
					return
				}

				//debug print statements to see if aliveCells is being updated
				fmt.Println("alive cells after rpc call: ", len(AliveCellsRes.AliveCells))
				fmt.Println("current turn: ", AliveCellsRes.CurrentTurn)

				//update c.events channel with the aliveCells count
				c.events <- AliveCellsCount{
					CompletedTurns: AliveCellsRes.CurrentTurn,
					CellsCount:     len(AliveCellsRes.AliveCells),
				}

			}
		}
	}()

	stopFlipped := make(chan struct{}) //signal to tell goroutine when to stop
	go func() {
		defer close(stopFlipped)
		lastTurn := 0 //turn before currentTurn
		for {
			select {
			case <-stopAll: //stop all
				return

			default:
				select {
				case <-stopAll:
					return
				default:
				}

				var flipRes stubs.Response
				//flipRes.CurrentTurn = 0
				err := flipReport.Call(stubs.CellsFlippedMethod, req, &flipRes) //rpc call to server's CellsFlippedMethod
				if err != nil {
					return
				}

				if flipRes.CurrentTurn > lastTurn && flipRes.CurrentTurn <= p.Turns { //check if the server has actually advanced to the next turn

					c.events <- CellsFlipped{ //send an event to report which cells have been flipped
						CompletedTurns: flipRes.CurrentTurn,
						Cells:          flipRes.Flipped,
					}
					c.events <- TurnComplete{ //send an event to report turn has completed with server's current turn.
						CompletedTurns: flipRes.CurrentTurn,
					}
					lastTurn = flipRes.CurrentTurn //increment last turn
				}

				if flipRes.CurrentTurn >= p.Turns { //if final turn, then exit
					return

				}

			}
		}
	}()

	//blocking rpc call to run all turns of GOL
	var res stubs.Response
	err = client.Call(stubs.ParallelGolMethod, req, &res)
	if err != nil {

		if errors.Is(err, io.EOF) || err == io.ErrUnexpectedEOF {
			fmt.Println("Client closed")
		} else {
			fmt.Println("Error executing GolMethod", err)
		}
	}

	close(stopAll) //wakes up goroutines on <-stopAll and makes them shut down
	<-stopFlipped  //wait until flipped function has stopped
	<-stopAlive    //wait until alivecells function has stopped
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//sends final state to events channel
	c.events <- FinalTurnComplete{turn, res.AliveCells}

	filename = fmt.Sprintf("%sx%d", filename, p.Turns)

	fmt.Println("turns:", p.Turns)
	fmt.Println("filename: ", filename)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	//fmt.Println(res.FinalWorld)
	mu.Lock()
	if !kPress {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c.ioOutput <- res.FinalWorld[y][x]
			}
		}
	}
	kPress = false
	mu.Unlock()

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	//distributor is quitting so notify events channel
	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
