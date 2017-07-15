package ann

import (
	"fmt"
	"math/rand"

	"github.com/gonum/matrix/mat64"
)

type mrpt struct {
	trees []*tree
}

type tree struct {
	root *node
	r    mat64.Matrix
}

type node struct {
	split float64
	left  *node
	right *node

	xs [][]float64
}

// NewMRPTNNer creates a NN index using random projection trees
// See https://arxiv.org/pdf/1509.06957.pdf for additional details
// t -> number of trees, l -> depth of tree
func NewMRPTNNer(t int, l int, xs [][]float64) NNer {
	a := 0.5 // TODO(temporary)
	return &mrpt{trees: growTrees(xs, t, l, a)}
}

func growTrees(xs [][]float64, t int, l int, a float64) []*tree {
	// Number of vectors
	n := len(xs)

	// Infer vector dimension from xs
	d := len(xs[0])

	trees := []*tree{}

	for i := 0; i < t; i++ {
		// Create a new random projection matrix
		r := mat64.NewDense(d, l, nil)
		// Create one random vector per tree level
		for j := 0; j < (l - 1); j++ {
			vs := []float64{}

			for k := 0; k < d; k++ {
				// TODO: Use a sparse vector strategy
				vs = append(vs, rand.NormFloat64())
			}

			// Set the random vector into the matrix
			r.SetCol(j, vs)
		}

		X := mat64.NewDense(d, n, nil)
		for j := 0; j < n; j++ {
			X.SetCol(j, xs[j])
		}

		var P mat64.Dense
		P.Mul(X, r)

		// Create a new tree
		trees = append(trees, &tree{
			r:    r,
			root: growTree(xs, l, 0, r),
		})
	}

	return trees
}

// growTree is a recursive function for building a RP tree
// xs -> points
// r -> random projection matrix
func growTree(xs [][]float64, l, level int, r mat64.Matrix) *node {
	if level == l {
		return &node{xs: xs}
	}

	return nil
}

func (nn *mrpt) NN(q []float64) []float64 {
	// Keep candidates in a set
	xsSet := map[string][]float64{}
	votes := map[string]int{}

	// How many votes does a vector need to be included in the output set
	reqVotes := 1

	// Query the trees to get candidates
	for _, tree := range nn.trees {
		xs := queryTree(tree, q)
		for _, x := range xs {
			// Count vote
			k := fmt.Sprintf("%v", x)
			votes[k]++

			// If vector has enough votes, include in output set
			if votes[k] == reqVotes {
				xsSet[k] = x
			}
		}
	}

	xsCandidates := [][]float64{}
	for _, x := range xsSet {
		xsCandidates = append(xsCandidates, x)
	}

	// Perform naive k-nearest-neighbor search on candidates set
	knn := NewExhaustiveNNer(xsCandidates)
	return knn.NN(q)
}

func queryTree(tree *tree, q []float64) [][]float64 {
	// Get vector dimension and depth of tree
	d, l := tree.r.Dims()

	// Convert q to a vector so we can perform matrix math
	qv := mat64.NewVector(d, q)

	// Project query point onto tree's random matrix
	var p mat64.Vector
	p.MulVec(tree.r.T(), qv)

	// Traverse the tree until point lands in a bucket
	node := tree.root
	for i := 0; i < (l - 1); i++ {
		if p.At(i, 0) <= node.split {
			node = node.left
		} else {
			node = node.right
		}
	}

	return node.xs
}
