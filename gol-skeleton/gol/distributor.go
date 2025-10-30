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

func calculateAliveNeighbours(world [][]byte, height, width, x, y int) int {
	count := 0
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {

			if dx == 0 && dy == 0 {
				continue // skip the cell itself
			}
			nx := (x + dx + width) % width
			ny := (y + dy + height) % height

			if world[ny][nx] == 255 {
				count++
			}
		}
	}
	return count
}

func calculateAliveCells(world [][]byte, startY, endY, width int) []util.Cell {
	var alive []util.Cell
	for y := 0; y < endY-startY; y++ {
		for x := 0; x < width; x++ {
			if world[y][x] == 255 {
				alive = append(alive, util.Cell{X: x, Y: y})
			}
		}
	}
	return alive
}

func nextState(currentWorld [][]byte, startY, endY, height, width int, skipRow bool) [][]byte {
	nextWorld := make([][]byte, endY-startY) // Initialises 2D slice with dimensions of the image
	for i := range nextWorld {
		nextWorld[i] = make([]byte, width)
	}
	indexNextWorld := 0
	for y := startY; y < endY; y++ {
		for x := 0; x < width; x++ {
			neighbours := calculateAliveNeighbours(currentWorld, height, width, x, y)
			if currentWorld[y][x] == 255 {
				if neighbours < 2 || neighbours > 3 {
					nextWorld[indexNextWorld][x] = 0
				} else {
					nextWorld[indexNextWorld][x] = 255
				}
			} else if neighbours == 3 {
				nextWorld[indexNextWorld][x] = 255
			} else {
				nextWorld[indexNextWorld][x] = 0
			}
		}
		indexNextWorld++
	}
	return nextWorld
}

func workerNextState(currentWorld [][]byte, width, height, startY, endY int, out chan<- [][]byte, skipRow bool) {
	chunk := nextState(currentWorld, startY, endY, height, width, skipRow)
	out <- chunk
}

func workerAliveCells(world [][]byte, startY, endY, width int, out chan<- []util.Cell) {
	chunk := calculateAliveCells(world, startY, endY, width)
	out <- chunk
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	height := p.ImageHeight
	width := p.ImageWidth

	currentWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image

	threads := p.Threads
	rows := height / threads

	for i := range currentWorld {
		currentWorld[i] = make([]byte, width)
	}
	command := ioCommand(ioInput)
	c.ioCommand <- command
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	c.ioFilename <- filename

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			b, channelOpen := <-c.ioInput
			if !channelOpen {
				return
			}
			currentWorld[y][x] = b
		}
	}

	turn := 0
	c.events <- StateChange{turn, Executing}

	outChannel := make([]chan [][]byte, threads)
	for i := range outChannel {
		outChannel[i] = make(chan [][]byte)
	}

	// TODO: Execute all turns of the Game of Life.
	for rounds := 0; rounds < p.Turns; rounds++ {
		newWorld := make([][]byte, 0, height)

		for i := 0; i < threads; i++ {
			skipRow := false
			startY := i * rows
			endY := (i + 1) * rows

			if i == threads-1 {
				endY = height
			}

			if startY > 0 {
				skipRow = true
			}
			go workerNextState(currentWorld, width, height, startY, endY, outChannel[i], skipRow)
		}
		for i := 0; i < threads; i++ {
			chunk := <-outChannel[i]
			newWorld = append(newWorld, chunk...)
		}
		currentWorld = newWorld
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{turn, calculateAliveCells(currentWorld, 0, height, width)}

	filename = fmt.Sprintf("%sx%d", filename, p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c.ioOutput <- currentWorld[y][x]
		}
	}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
