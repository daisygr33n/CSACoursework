package stubs

import "uk.ac.bris.cs/gameoflife/util"

var ExecGol = "Connection.Gol"

type Response struct {
	FinalWorld [][]byte
	AliveCells []util.Cell
}

type Request struct {
	StartWorld [][]byte
	Turns      int
	Height     int
	Width      int
}
