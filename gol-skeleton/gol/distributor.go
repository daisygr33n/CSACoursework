package gol

import (
	"fmt"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event     //send only
	ioCommand  chan<- ioCommand // send only
	ioIdle     <-chan bool      // receive
	ioFilename chan<- string    // send
	ioOutput   chan<- uint8     // send
	ioInput    <-chan uint8     // receive
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

func nextState(c distributorChannels, currentWorld [][]byte, startY, endY, height, width, turn int) [][]byte {
	nextWorld := make([][]byte, endY-startY) // Initialises 2D slice with dimensions of the image
	for i := range nextWorld {
		nextWorld[i] = make([]byte, width)
	}
	var flipped []util.Cell
	indexNextWorld := 0
	for y := startY; y < endY; y++ {
		for x := 0; x < width; x++ {
			neighbours := calculateAliveNeighbours(currentWorld, height, width, x, y)
			if currentWorld[y][x] == 255 {
				if neighbours < 2 || neighbours > 3 {
					nextWorld[indexNextWorld][x] = 0
					flipped = append(flipped, util.Cell{X: x, Y: y})
				} else {
					nextWorld[indexNextWorld][x] = 255
				}
			} else if neighbours == 3 {
				nextWorld[indexNextWorld][x] = 255
				flipped = append(flipped, util.Cell{X: x, Y: y})
			} else {
				nextWorld[indexNextWorld][x] = 0
			}
		}
		indexNextWorld++
	}
	c.events <- CellsFlipped{turn, flipped}
	return nextWorld
}

func workerNextState(c distributorChannels, currentWorld [][]byte, width, height, startY, endY, turn int, out chan<- [][]byte) {
	chunk := nextState(c, currentWorld, startY, endY, height, width, turn)
	out <- chunk
}

func checkKeyPress(c distributorChannels, pausedChannel chan<- bool, ticker *time.Ticker) {
	for keyPress := range c.ioKeyPress {
		switch keyPress {
		case 'p':
			mu.Lock()
			if paused {
				paused = false
				pausedChannel <- false
			} else {
				paused = true
				pausedChannel <- true
			}
			mu.Unlock()
		case 's':
			if paused {
				pausedChannel <- true
			}
			mu.Lock()
			save = true
			mu.Unlock()
		case 'q':
			if paused {
				pausedChannel <- true
			}
			mu.Lock()
			terminate = true
			mu.Unlock()
			ticker.Stop()
		}
	}
}

func saveImage(c distributorChannels, filename string, turn, height, width int, currentWorld [][]byte) {
	filename = fmt.Sprintf("%sx%d", filename, turn)
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
	c.events <- ImageOutputComplete{turn, filename}
}

func saveWorld(c distributorChannels, filename string, turn, height, width int, currentWorld [][]byte) {
	go saveImage(c, filename, turn, height, width, currentWorld)
	mu.Lock()
	save = false
	mu.Unlock()
}

func terminateGame(c distributorChannels, filename string, turn, height, width int, currentWorld [][]byte) {
	saveImage(c, filename, turn, height, width, currentWorld)
	c.events <- FinalTurnComplete{turn, calculateAliveCells(currentWorld, 0, height, width)}
	c.events <- StateChange{turn, Quitting}
}

func sendFlippedCells(c distributorChannels, currentWorld [][]byte, turn, height, width int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if currentWorld[y][x] == 255 {
				c.events <- CellFlipped{turn, util.Cell{X: x, Y: y}}
			}
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	height := p.ImageHeight
	width := p.ImageWidth

	currentWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image

	threads := p.Threads
	rows := height / p.Threads

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

	mu.Lock()
	aliveCount = len(calculateAliveCells(currentWorld, 0, width, height))

	turn = 0
	sendFlippedCells(c, currentWorld, turn, height, width)

	c.events <- StateChange{turn, Executing}
	mu.Unlock()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				if turn == p.Turns || terminate {
					terminate = false
					mu.Unlock()
					return
				}
				c.events <- AliveCellsCount{turn, aliveCount}
				mu.Unlock()
			}

		}
	}()

	pausedChannel := make(chan bool, 1)
	go checkKeyPress(c, pausedChannel, ticker)

	outChannel := make([]chan [][]byte, threads)
	for i := range outChannel {
		outChannel[i] = make(chan [][]byte)
	}

	// TODO: Execute all turns of the Game of Life.
loopTurns:
	for round := 0; round < p.Turns; round++ {
		newWorld := make([][]byte, 0, height)

		mu.Lock()
		localPaused := paused
		localSave := save
		localTerminate := terminate
		mu.Unlock()

		if localPaused {
			c.events <- StateChange{round, Paused}
		loopPaused:
			for {
				select {
				case keepPaused := <-pausedChannel:
					if !keepPaused {
						c.events <- StateChange{round, Executing}
						break loopPaused
					}
					mu.Lock()
					localSave = save
					mu.Unlock()
					if localSave {
						saveWorld(c, filename, round, height, width, currentWorld)
					}
					mu.Lock()
					localTerminate = terminate
					mu.Unlock()
					if localTerminate {
						terminateGame(c, filename, round, height, width, currentWorld)
						break loopTurns
					}
				}
				mu.Lock()
				localPaused = paused
				mu.Unlock()
			}
		}
		mu.Lock()
		localPaused = paused
		localSave = save
		localTerminate = terminate
		mu.Unlock()

		if localSave {
			saveWorld(c, filename, round, height, width, currentWorld)
		}

		if localTerminate {
			terminateGame(c, filename, round, height, width, currentWorld)
			break loopTurns
		}

		for i := 0; i < threads; i++ {
			startY := i * rows
			endY := (i + 1) * rows

			if i == threads-1 {
				endY = height
			}

			go workerNextState(c, currentWorld, width, height, startY, endY, round, outChannel[i])
		}
		for i := 0; i < threads; i++ {
			chunk := <-outChannel[i]
			newWorld = append(newWorld, chunk...)
		}
		currentWorld = newWorld
		mu.Lock()
		turn++
		aliveCount = len(calculateAliveCells(currentWorld, 0, height, width))
		c.events <- TurnComplete{turn}
		mu.Unlock()
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	mu.Lock()
	ticker.Stop()
	if !terminate {
		c.events <- FinalTurnComplete{turn, calculateAliveCells(currentWorld, 0, height, width)}
		saveImage(c, filename, p.Turns, height, width, currentWorld)
		c.events <- StateChange{turn, Quitting}
	}

	terminate = false
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
	mu.Unlock()
}
