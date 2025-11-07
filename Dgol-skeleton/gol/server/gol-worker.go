package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// added variables to connection struct so both methods could access same variables
type Connection struct {
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

// helper function to calculate slice of alive cells
func calculateAliveCells(world [][]byte, height int, width int) []util.Cell {
	var alive []util.Cell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if world[y][x] == 255 {
				alive = append(alive, util.Cell{X: x, Y: y}) //slice containing all alive cells
			}
		}
	}
	return alive
}

func (c *Connection) SaveWorld(request stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *Connection) TerminateWorld(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *Connection) PauseWorld(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()

	if paused {
		paused = false

	} else {
		paused = true
	}

	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *Connection) TerminateClient(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()

	go func() {
		<-golFinished
		shutDown <- true
	}()

	return
}

// GolMethod executes gol for specified number of turns
func (c *Connection) GolMethod(req stubs.Request, res *stubs.Response) (err error) {

	currentWorld := req.StartWorld
	height := req.Height
	width := req.Width

	mu.Lock()
	globalWorld = currentWorld
	mu.Unlock()

	//computes alive cells for starting world
	alive := calculateAliveCells(currentWorld, height, width)
	c.mu.Lock()                                                          //updating critical section so needs to be locked
	c.aliveCells = alive                                                 //update alivecells slice (Connection struct is shared)
	fmt.Println("Start of gol method, alive cells: ", len(c.aliveCells)) //debugging print statement
	c.currentTurn = 0                                                    //resets current turn to 0
	c.mu.Unlock()                                                        //unlock lock after finishing updating alive cells

	//main loop computing GOL
loopTurns:
	for rounds := 0; rounds < req.Turns; rounds++ { //loops for each turn in p.turns

		mu.Lock()
		if terminate {
			mu.Unlock()
			break loopTurns
		}
		mu.Unlock()

		mu.Lock()
		localPaused := paused
		mu.Unlock()

		if localPaused {
		loopPaused:
			for {
				mu.Lock()
				if !paused {
					mu.Unlock()
					break loopPaused
				}
				mu.Unlock()
			}
		}

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
		currentWorld = nextWorld //advance to next world
		mu.Lock()
		globalWorld = currentWorld
		mu.Unlock()

		//update alive cells and current turn
		alive := calculateAliveCells(currentWorld, height, width)
		c.mu.Lock()          //lock during critical section
		c.aliveCells = alive //change shared section (Connection struct)
		c.currentTurn = rounds + 1
		fmt.Println("ALive cells in GOLMethod: ", len(alive)) //debugging statements
		fmt.Println("Current turn: ", c.currentTurn)
		c.mu.Unlock() //unlock after finishing editing critical sections
	}
	//send the response back to client
	c.mu.Lock()
	mu.Lock()
	res.FinalWorld = globalWorld
	res.CurrentTurn = c.currentTurn
	res.AliveCells = c.aliveCells
	terminate = false
	mu.Unlock()
	c.mu.Unlock()
	fmt.Println("returning from simulation")

	select {
	case <-golFinished:
	default:
		close(golFinished)
	}

	return
}

// returns current number of alive cells and turn for client (used for two second ticker)
func (c *Connection) AliveCellsMethod(req stubs.Request, res *stubs.Response) error {
	c.mu.Lock()                   //lock critical section
	res.AliveCells = c.aliveCells //accessing shared Connection struct
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock() //unlock
	return nil
}

// starts rpc server
func main() {
	err := rpc.Register(&Connection{}) //registers Connection struct mehtods for RPC
	if err != nil {
		fmt.Println("rpc register error:", err)
	}
	pAddr := flag.String("port", "8030", "Port to listen on") //specify a port
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
				log.Println("Accept error:", err)
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

	/*defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener close error:", err)
		}
	}(listener)
	fmt.Printf("Listening on port %s\n", *pAddr)
	rpc.Accept(listener) //blocking call constantly accepting incoming connections*/
}
