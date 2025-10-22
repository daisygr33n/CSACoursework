package gol

import (
	"fmt"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func makeChannels(disChan distributorChannels, ioChan ioChannels) (distributorChannels, ioChannels) {
	events := make(chan Event)
	command := make(chan ioCommand)
	idle := make(chan bool)
	filename := make(chan string)
	output := make(chan uint8)
	input := make(chan uint8)

	disChan = distributorChannels{
		events:     events,
		ioCommand:  command,
		ioIdle:     idle,
		ioFilename: filename,
		ioOutput:   output,
		ioInput:    input,
	}

	ioChan = ioChannels{
		command:  command,
		idle:     idle,
		filename: filename,
		output:   output,
		input:    input,
	}
	return disChan, ioChan
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	fmt.Println("distributor")

	height := p.ImageHeight
	width := p.ImageWidth

	c, ioChan := makeChannels(c, ioChannels{})

	state := ioState{
		params:   p,
		channels: ioChan,
	}
	bytes := 0
	go state.readPgmImage()
	fmt.Println("distributor finished")
	//filename := ""
	fmt.Println("distributor about to read pgm")
	if height == 16 {
		c.ioFilename <- "16x16"
		bytes = 16
	} else if height == 64 {
		c.ioFilename <- "64x64"
		bytes = 64
	} else if height == 128 {
		c.ioFilename <- "128x128"
		bytes = 128
	} else if height == 256 {
		c.ioFilename <- "256x256"
		bytes = 256
	} else {
		c.ioFilename <- "512x512"
		bytes = 512
	}
	fmt.Println("distributor finished", bytes)

	currentWorld := <-c.ioInput
	fmt.Println(currentWorld)

	nextWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image
	for i := range nextWorld {
		nextWorld[i] = make([]byte, width)
	}

	turn := 0
	c.events <- StateChange{turn, Executing}

	// TODO: Execute all turns of the Game of Life.

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			count := 0

			for surroundY := -1; surroundY < 2; surroundY++ {
				for surroundX := -1; surroundX < 2; surroundX++ {
					if surroundY == 0 && surroundX == 0 {
						continue
					}
					dy, dx := (y+surroundY+height)%height, (x+surroundX+width)%width
					if nextWorld[dy][dx] == 255 {
						count++
					}
				}
			}
		}
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
