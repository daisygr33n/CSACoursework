package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// added variables to connection struct so both methods could access same variables
type Connection struct {
	flipped     []util.Cell
	aliveCells  []util.Cell //slice of aliveCells that can be shared by GoLmethod and AliveCellsreport
	currentTurn int         //current turn of gol
	mu          sync.Mutex  //mutex lock to protect critical section (reading and writing to alivecells)
}

var (
	globalWorld [][]byte
	terminate   = false
	paused      = false
	mu          sync.Mutex
)

var shutDown = make(chan bool)
var golFinished = make(chan bool)

// helper function to calculate alive cells surrounding the current cell
func calculateAliveNeighbours(world [][]byte, height, width int, x, y int) int {
	count := 0
	for dx := -1; dx <= 1; dx++ { //calculating neighbour coordinates
		for dy := -1; dy <= 1; dy++ {

			if dx == 0 && dy == 0 {
				continue // skip the cell itself
			}
			nx := (x + dx + width) % width
			ny := (y + dy + width) % width
			// this is for wrap around at edges
			// all cells have 8 neighbours including corner cells
			if world[ny][nx] == 255 {
				count++
			}
		}
	}
	return count
}

// helper function to calculate slice of alive cells
func calculateAliveCells(world [][]byte, startY int, height int, width int) []util.Cell {
	var alive []util.Cell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if world[y][x] == 255 {
				alive = append(alive, util.Cell{X: x, Y: y + startY}) //slice containing all alive cells
			}
		}
	}
	return alive
}

func (c *Connection) TerminateWorker(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()

	select {
	case shutDown <- true:
	default:

	}
	/*go func() {
		<-golFinished
		shutDown <- true
	}()*/

	return nil
}

func (c *Connection) NextStateInit(req stubs.Request, res *stubs.Response) (err error) {
	//fmt.Println("received slice")
	res.FinalWorld = NextState(req.StartWorld, req.StartY, req.EndY, req.Height, req.Width)
	res.AliveCells = calculateAliveCells(res.FinalWorld, req.StartY, req.EndY-req.StartY, req.Width)
	res.CurrentTurn = req.CurrentTurn
	return
}

/*func (c *Connection) FlippedCells(req stubs.Request, res *stubs.Response) (err error) {

}*/

func (c *Connection) AliveCells(req stubs.Request, res *stubs.Response) (err error) {
	res.AliveCells = calculateAliveCells(req.StartWorld, req.StartY, req.Height, req.Width)
	res.CurrentTurn = req.CurrentTurn
	res.FinalWorld = req.StartWorld
	return
}

func NextState(currentWorld [][]byte, startY, endY, height, width int) [][]byte {
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
	/*select {
	case <-golFinished:
	default:
		close(golFinished)
	}*/
	return nextWorld
}

// starts rpc server
func main() {
	err := rpc.Register(&Connection{}) //registers Connection struct mehtods for RPC
	if err != nil {
		fmt.Println("rpc register error:", err)
	}
	pAddr := flag.String("port", "8040", "Port to listen on") //specify a port
	flag.Parse()
	listener, err := net.Listen("tcp", ":"+*pAddr) //listen on tcp port
	if err != nil {
		fmt.Println(err)
	}

	defer listener.Close()
	fmt.Printf("Listening on port %s\n", *pAddr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				fmt.Println("Error", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()

	<-shutDown

	fmt.Println("Shutting down...")
	err1 := listener.Close()
	if err1 != nil {
		return
	}
	fmt.Println("Goodbye!")

}
