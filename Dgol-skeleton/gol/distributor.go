package gol

import (
	"fmt"
	"net/rpc"

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
	if height == 16 {
		c.ioFilename <- "16x16"
	} else if height == 64 {
		c.ioFilename <- "64x64"
	} else if height == 128 {
		c.ioFilename <- "128x128"
	} else if height == 256 {
		c.ioFilename <- "256x256"
	} else {
		c.ioFilename <- "512x512"
	}
	fmt.Println("command received")

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

	turn := 0
	c.events <- StateChange{turn, Executing}

	// TODO: Execute all turns of the Game of Life.
	client, err := rpc.Dial("tcp", "3.91.52.3:8030")
	if err != nil {
		fmt.Println(err)
	}
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println("Error closing client", err)
		}
	}(client)
	req := stubs.Request{
		StartWorld: currentWorld,
		Turns:      p.Turns,
		Height:     height,
		Width:      width,
	}
	var res stubs.Response
	err = client.Call(stubs.ExecGol, req, &res)
	if err != nil {
		fmt.Println(err)
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{turn, res.AliveCells}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
