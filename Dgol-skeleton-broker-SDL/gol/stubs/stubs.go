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
var CellsFlippedMethod = "ConnectionBroker.CellsFlippedMethod"
var ResetMethod = "ConnectionBroker.ResetMethod"

// var NewMethod = "ConnectionBroker.ExecMethod"
var AliveCellsBrokers = "ConnectionBroker.AliveCellsMethod"
var NextState = "Connection.NextStateInit"
var ParallelGolMethod = "ConnectionBroker.ParallelGolMethod"
var AliveCells = "Connection.AliveCells"
var FlippedCells = "Connection.FlippedCells"
var TerminateWorker = "Connection.TerminateWorker"

type Response struct {
	FinalWorld  [][]byte
	AliveCells  []util.Cell
	CurrentTurn int
	Flipped     []util.Cell
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
}
