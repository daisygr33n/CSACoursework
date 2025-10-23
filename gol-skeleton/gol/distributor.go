package gol

import (
	"fmt"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event     //send only
	ioCommand  chan<- ioCommand // send only
	ioIdle     <-chan bool      // receive
	ioFilename chan<- string    // send
	ioOutput   chan<- uint8     // send
	ioInput    <-chan uint8     // receive
}

func calculateAliveNeighbours(world [][]byte, dimensions int, x, y int) int {
	count := 0
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {

			if dx == 0 && dy == 0 {
				continue // skip the cell itself
			}
			nx := (x + dx + dimensions) % dimensions
			ny := (y + dy + dimensions) % dimensions
			// this is for wrap around at edges
			// all cells have 8 neighbours including corner cells
			if world[ny][nx] == 255 {
				count++
			}
		}
	}
	return count
}

func calculateAliveCells(world [][]byte, dimensions int) []util.Cell {
	var alive []util.Cell
	for y := 0; y < dimensions; y++ {
		for x := 0; x < dimensions; x++ {
			if world[y][x] == 255 {
				alive = append(alive, util.Cell{X: x, Y: y})
			}
		}
	}
	return alive
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
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
	//c.ioCommand <- ioCommand(ioOutput)
	//c.ioFilename <- bytes

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
	//fmt.Println(currentWorld)
	turn := 0
	c.events <- StateChange{turn, Executing}

	// TODO: Execute all turns of the Game of Life.
	for rounds := 0; rounds < p.Turns; rounds++ {
		nextWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image
		for i := range nextWorld {
			nextWorld[i] = make([]byte, width)
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				neighbours := calculateAliveNeighbours(currentWorld, height, x, y)
				if currentWorld[y][x] == 255 {
					if neighbours < 2 || neighbours > 3 {
						nextWorld[y][x] = 0
					} else {
						nextWorld[y][x] = 255
					}
				} else if neighbours == 3 {
					nextWorld[y][x] = 255
				} else {
					nextWorld[y][x] = 0
				}
			}
		}
		currentWorld = nextWorld
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{turn, calculateAliveCells(currentWorld, height)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
