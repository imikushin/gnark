// Copyright 2020 ConsenSys AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by gnark/internal/generators DO NOT EDIT

package groth16_test

import (
	curve "github.com/consensys/gurvy/bn256"
	"github.com/consensys/gurvy/bn256/fr"

	bn256backend "github.com/consensys/gnark/internal/backend/bn256"

	"testing"

	bn256groth16 "github.com/consensys/gnark/internal/backend/bn256/groth16"

	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/r1cs"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/internal/backend/circuits"
	"github.com/consensys/gurvy"
)

func TestCircuits(t *testing.T) {
	for name, circuit := range circuits.Circuits {
		t.Run(name, func(t *testing.T) {
			assert := groth16.NewAssert(t)
			r1cs := circuit.R1CS.ToR1CS(curve.ID)
			assert.NotSolved(r1cs, circuit.Bad)
			assert.Solved(r1cs, circuit.Good, nil)
		})
	}
}

func TestParsePublicInput(t *testing.T) {

	expectedNames := [2]string{"data", backend.OneWire}

	inputOneWire := make(map[string]interface{})
	inputOneWire[backend.OneWire] = 3
	if _, err := bn256groth16.ParsePublicInput(expectedNames[:], inputOneWire); err == nil {
		t.Fatal("expected ErrMissingAssigment error")
	}

	missingInput := make(map[string]interface{})
	if _, err := bn256groth16.ParsePublicInput(expectedNames[:], missingInput); err == nil {
		t.Fatal("expected ErrMissingAssigment")
	}

	correctInput := make(map[string]interface{})
	correctInput["data"] = 3
	got, err := bn256groth16.ParsePublicInput(expectedNames[:], correctInput)
	if err != nil {
		t.Fatal(err)
	}

	expected := make([]fr.Element, 2)
	expected[0].SetUint64(3).FromMont()
	expected[1].SetUint64(1).FromMont()
	if len(got) != len(expected) {
		t.Fatal("Unexpected length for assignment")
	}
	for i := 0; i < len(got); i++ {
		if !got[i].Equal(&expected[i]) {
			t.Fatal("error public assignment")
		}
	}

}

//--------------------//
//     benches		  //
//--------------------//

type refCircuit struct {
	nbConstraints int
	X             frontend.Variable
	Y             frontend.Variable `gnark:",public"`
}

func (circuit *refCircuit) Define(curveID gurvy.ID, cs *frontend.ConstraintSystem) error {
	for i := 0; i < circuit.nbConstraints; i++ {
		circuit.X = cs.Mul(circuit.X, circuit.X)
	}
	cs.MustBeEqual(circuit.X, circuit.Y)
	return nil
}

func referenceCircuit() (r1cs.R1CS, map[string]interface{}) {
	const nbConstraints = 40000
	circuit := refCircuit{
		nbConstraints: nbConstraints,
	}
	r1cs, err := frontend.Compile(curve.ID, &circuit)
	if err != nil {
		panic(err)
	}

	good := make(map[string]interface{})
	good["X"] = 2

	// compute expected Y
	var expectedY fr.Element
	expectedY.SetUint64(2)

	for i := 0; i < nbConstraints; i++ {
		expectedY.Mul(&expectedY, &expectedY)
	}

	good["Y"] = expectedY

	return r1cs, good
}

func TestReferenceCircuit(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	assert := groth16.NewAssert(t)
	r1cs, solution := referenceCircuit()
	assert.Solved(r1cs, solution, nil)
}

// BenchmarkSetup is a helper to benchmark Setup on a given circuit
func BenchmarkSetup(b *testing.B) {
	r1cs, _ := referenceCircuit()

	var pk bn256groth16.ProvingKey
	var vk bn256groth16.VerifyingKey
	b.ResetTimer()

	b.Run("setup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bn256groth16.Setup(r1cs.(*bn256backend.R1CS), &pk, &vk)
		}
	})
}

// BenchmarkProver is a helper to benchmark Prove on a given circuit
// it will run the Setup, reset the benchmark timer and benchmark the prover
func BenchmarkProver(b *testing.B) {
	r1cs, solution := referenceCircuit()

	var pk bn256groth16.ProvingKey
	bn256groth16.DummySetup(r1cs.(*bn256backend.R1CS), &pk)

	b.ResetTimer()
	b.Run("prover", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = bn256groth16.Prove(r1cs.(*bn256backend.R1CS), &pk, solution)
		}
	})
}

// BenchmarkVerifier is a helper to benchmark Verify on a given circuit
// it will run the Setup, the Prover and reset the benchmark timer and benchmark the verifier
// the provided solution will be filtered to keep only public inputs
func BenchmarkVerifier(b *testing.B) {
	r1cs, solution := referenceCircuit()

	var pk bn256groth16.ProvingKey
	var vk bn256groth16.VerifyingKey
	bn256groth16.Setup(r1cs.(*bn256backend.R1CS), &pk, &vk)
	proof, err := bn256groth16.Prove(r1cs.(*bn256backend.R1CS), &pk, solution)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	b.Run("verifier", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bn256groth16.Verify(proof, &vk, solution)
		}
	})
}
