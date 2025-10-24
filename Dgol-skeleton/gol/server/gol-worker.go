package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Connection struct{}

// helper function to calculate alive cells surrounding the current cell
func calculateAliveNeighbours(world [][]byte, dimensions int, x, y int) int {
	count := 0
	for dx := -1; dx <= 1; dx++ { //calculating neighbour coordinates
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
				alive = append(alive, util.Cell{X: x, Y: y}) //slice containing all alive cells
			}
		}
	}
	return alive
}

func (c *Connection) Gol(req stubs.Request, res *stubs.Response) (err error) {

	currentWorld := req.StartWorld
	height := req.Height
	width := req.Width

	for rounds := 0; rounds < req.Turns; rounds++ { //loops for each turn in p.turns
		nextWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image
		for i := range nextWorld {
			nextWorld[i] = make([]byte, width)
		}
		for y := 0; y < height; y++ { //loops over each row and column checking if its neighbours are alive
			for x := 0; x < width; x++ {
				neighbours := calculateAliveNeighbours(currentWorld, height, x, y)
				if currentWorld[y][x] == 255 {
					if neighbours < 2 || neighbours > 3 {
						nextWorld[y][x] = 0
					} else {
						nextWorld[y][x] = 255 //gol logic
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
	res.FinalWorld = currentWorld
	res.AliveCells = calculateAliveCells(currentWorld, height)
	return
}

func main() {
	rpc.Register(&Connection{})
	pAddr := flag.String("port", "8030", "Port to listen on") //specify a port
	flag.Parse()
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	defer listener.Close()
	fmt.Printf("Listening on port %s\n", *pAddr)
	rpc.Accept(listener)
}
