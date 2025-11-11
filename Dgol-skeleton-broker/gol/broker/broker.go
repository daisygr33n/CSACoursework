package main

import (
	"errors"
	"flag"
	"net"

	//"flag"
	"fmt"
	//"log"
	//"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type ConnectionBroker struct {
	aliveCells  []util.Cell //slice of aliveCells that can be shared by GoLmethod and AliveCellsreport
	currentTurn int         //current turn of gol
	mu          sync.Mutex  //mutex lock to protect critical section (reading and writing to alivecells)
}

var (
	globalWorld [][]byte
	terminate   = false
	paused      = false
	shutDown    = make(chan bool)
	golFinished = make(chan bool)
	clients     = make([]*rpc.Client, 4)
	mu          sync.Mutex
)

//var shutDown = make(chan bool)
//var golFinished = make(chan bool)

func (c *ConnectionBroker) AliveCellsMethod(req stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	return
}

func (c *ConnectionBroker) SaveWorld(request stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *ConnectionBroker) TerminateWorld(request stubs.Request, res *stubs.Response) (err error) {
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

func (c *ConnectionBroker) PauseWorld(request stubs.Request, res *stubs.Response) (err error) {
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

func (c *ConnectionBroker) TerminateClient(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()

	client, err := rpc.Dial("tcp", "localhost:8040")
	if err != nil {
		fmt.Println(err)
	}
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println("error closing client: ", err)
		}
	}(client)

	var req stubs.Request
	var resTemp stubs.Response
	clients[0].Go(stubs.TerminateWorker, req, &resTemp, nil)
	clients[1].Go(stubs.TerminateWorker, req, &resTemp, nil)
	clients[2].Go(stubs.TerminateWorker, req, &resTemp, nil)
	clients[3].Go(stubs.TerminateWorker, req, &resTemp, nil)

	go func() {
		<-golFinished
		shutDown <- true
	}()

	return
}

func (c *ConnectionBroker) ParallelGolMethod(req stubs.Request, res *stubs.Response) (err error) {
	client, err := rpc.Dial("tcp", "3.91.7.217:8040")
	if err != nil {
		fmt.Println(err)
	}
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println("error closing client: ", err)
		}
	}(client)

	c.mu.Lock()
	c.currentTurn = 0
	c.aliveCells = make([]util.Cell, 0)
	c.mu.Unlock()

	height := req.Height
	width := req.Width
	threads := 4
	rows := height / 4

	responses := make([]*stubs.Response, 4)
	done := make(chan *rpc.Call, 4)

	currentWorld := req.StartWorld
	mu.Lock()
	globalWorld = currentWorld
	mu.Unlock()

	var aliveCells []util.Cell
	client.Call(stubs.AliveCells, req, &res)
	aliveCells = res.AliveCells
	//client.Go(stubs.TerminateWorker, req, &res, nil)

	c.mu.Lock()
	c.aliveCells = aliveCells
	c.mu.Unlock()

	//clients := make([]*rpc.Client, 4)

	/*for i := 0; i < threads; i++ {
		client, err := rpc.Dial("tcp", "8040")
		if err != nil {
			fmt.Println(err)
		}
		clients[i] = client
	}*/
	mu.Lock()
	client0, err := rpc.Dial("tcp", "54.174.193.121:8050")
	if err != nil {
		fmt.Println(err)
	}
	clients[0] = client0

	client1, err := rpc.Dial("tcp", "98.80.119.172:8060")
	if err != nil {
		fmt.Println(err)
	}
	clients[1] = client1

	client2, err := rpc.Dial("tcp", "34.201.120.217:8070")
	if err != nil {
		fmt.Println(err)
	}
	clients[2] = client2

	client3, err := rpc.Dial("tcp", "54.226.151.190:8080")
	if err != nil {
		fmt.Println(err)
	}
	clients[3] = client3
	mu.Unlock()

loopTurns:
	for round := 0; round < req.Turns; round++ {

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

		calls := make([]*rpc.Call, threads)

		for i := 0; i < 4; i++ {

			startY := i * rows
			endY := (i + 1) * rows

			if i == threads-1 {
				endY = height
			}

			newReq := stubs.Request{
				StartWorld:  currentWorld,
				StartY:      startY,
				EndY:        endY,
				Height:      height,
				Width:       width,
				Turns:       req.Turns,
				CurrentTurn: round,
				Threads:     threads,
			}

			//var resTemp stubs.Response
			var resTemp = new(stubs.Response)
			responses[i] = resTemp
			calls[i] = clients[i].Go(stubs.NextState, newReq, &resTemp, done)
		}

		for i := 0; i < 4; i++ {
			<-calls[i].Done
		}

		aliveCells = nil
		var nextWorld [][]byte
		for i := 0; i < 4; i++ {
			nextWorld = append(nextWorld, responses[i].FinalWorld...)
			aliveCells = append(aliveCells, responses[i].AliveCells...)
		}
		c.mu.Lock()
		c.aliveCells = aliveCells
		c.currentTurn++
		c.mu.Unlock()

		currentWorld = nextWorld

		mu.Lock()
		globalWorld = currentWorld
		mu.Unlock()

	}
	c.mu.Lock()
	res.FinalWorld = currentWorld
	res.CurrentTurn = c.currentTurn
	res.AliveCells = aliveCells
	c.mu.Unlock()

	mu.Lock()
	terminate = false
	paused = false
	mu.Unlock()

	select {
	case <-golFinished:
	default:
		close(golFinished)
	}

	return
}

func main() {
	err := rpc.Register(&ConnectionBroker{}) //registers Connection struct methods for RPC
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
				if errors.Is(err, net.ErrClosed) {
					return
				}
				fmt.Println("Error", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()

	/*defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener close error:", err)
		}
	}(listener)
	fmt.Printf("Listening on port %s\n", *pAddr)
	rpc.Accept(listener) //blocking call constantly accepting incoming connections*/

	<-shutDown

	fmt.Println("Shutting down...")
	err1 := listener.Close()
	if err1 != nil {
		return
	}
	fmt.Println("Goodbye!")
}
