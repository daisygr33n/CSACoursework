package stubs

import "uk.ac.bris.cs/gameoflife/util"

/*
var ExecGolMethod = "Connection.GolMethod"
var AliveCellsMethod = "Connection.AliveCellsMethod"
*/
var SaveWorld = "ConnectionBroker.SaveWorld"
var TerminateWorld = "ConnectionBroker.TerminateWorld"
var PauseWorld = "ConnectionBroker.PauseWorld"
var TerminateClient = "ConnectionBroker.TerminateClient"

// var NewMethod = "ConnectionBroker.ExecMethod"
var AliveCellsBrokers = "ConnectionBroker.AliveCellsMethod"
var NextState = "Connection.NextStateInit"
var ParallelGolMethod = "ConnectionBroker.ParallelGolMethod"
var AliveCells = "Connection.AliveCells"
var TerminateWorker = "Connection.TerminateWorker"
var SendTopHalo = "Connection.SendTopHalo"
var SendBottomHalo = "Connection.SendBottomHalo"
var IndependentGameOfLife = "Connection.IndependentGameOfLife"

type Response struct {
	FinalWorld  [][]byte
	AliveCells  []util.Cell
	CurrentTurn int
}

type Request struct {
	StartWorld  [][]byte
	Turns       int
	Height      int
	Width       int
	StartY      int
	EndY        int
	CurrentTurn int
	Threads     int
	WorldHeight int
	WorkerIndex int
}

type Halo struct {
	TopHalo      []byte
	BottomHalo   []byte
	FinishedTurn int
}

type HaloReq struct {
	CurrentTurn int
	WorkerIndex int
}
