// Copyright 2020 ConsenSys Software Inc.
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

// Code generated by gnark DO NOT EDIT

package witness

import (
	"errors"
	"io"
	"reflect"

	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/internal/parser"

	"github.com/consensys/gurvy/bn256/fr"

	curve "github.com/consensys/gurvy/bn256"
)

// Full extracts the full witness secret || public (including ONE_WIRE)
// and returns a slice of field elements in montgomery form
func Full(w frontend.Witness) ([]fr.Element, error) {
	nbSecret, nbPublic, err := count(w)
	if err != nil {
		return nil, err
	}
	secret := make([]fr.Element, nbSecret)
	public := make([]fr.Element, nbPublic+1) // ONE_WIRE
	public[0] = fr.One()

	var i, j int
	j++
	var collectHandler parser.LeafHandler = func(visibility backend.Visibility, name string, tInput reflect.Value) error {
		v := tInput.Interface().(frontend.Variable)

		val := frontend.GetAssignedValue(v)
		if val == nil {
			return errors.New("variable " + name + " not assigned")
		}

		if visibility == backend.Secret {
			secret[i].SetInterface(val)
			i++
		} else if visibility == backend.Public {
			public[j].SetInterface(val)
			j++
		}
		return nil
	}
	if err := parser.Visit(w, "", backend.Unset, collectHandler, reflect.TypeOf(frontend.Variable{})); err != nil {
		return nil, err
	}
	return append(secret, public...), nil
}

// Public extracts the public witness (including ONE_WIRE)
// and returns a slice of field elements in REGULAR form
func Public(w frontend.Witness) ([]fr.Element, error) {
	_, nbPublic, err := count(w)
	if err != nil {
		return nil, err
	}
	public := make([]fr.Element, nbPublic+1) // ONE_WIRE
	public[0] = fr.One()
	public[0].FromMont()

	var j int
	j++
	var collectHandler parser.LeafHandler = func(visibility backend.Visibility, name string, tInput reflect.Value) error {

		if visibility == backend.Public {
			v := tInput.Interface().(frontend.Variable)
			val := frontend.GetAssignedValue(v)
			if val == nil {
				return errors.New("variable " + name + " not assigned")
			}
			public[j].SetInterface(val).FromMont()
			j++
		}
		return nil
	}
	if err := parser.Visit(w, "", backend.Unset, collectHandler, reflect.TypeOf(frontend.Variable{})); err != nil {
		return nil, err
	}
	return public, nil
}

const frSize = fr.Limbs * 8

// WriteFull serialize full witness [secret|one_wire|public] by encoding provided values
func WriteFull(w io.Writer, witness frontend.Witness) error {

	v, err := Full(witness)
	if err != nil {
		return err
	}

	enc := curve.NewEncoder(w)
	for i := 0; i < len(v); i++ {
		if err = enc.Encode(v[i]); err != nil {
			return err
		}
	}

	return nil

}

// WritePublic serialize publicWitness [public] without one_wire by encoding provided values
func WritePublic(w io.Writer, witness frontend.Witness) error {

	v, err := Public(witness)
	if err != nil {
		return err
	}

	enc := curve.NewEncoder(w)
	for i := 1; i < len(v); i++ { // skipping one_wire at [0]
		if err = enc.Encode(v[i]); err != nil {
			return err
		}
	}

	return nil

}

// ReadFull decodes witness[]byte -> []fr.Element
// witness is [secret|one_wire|public]
// returned value is in Montgomery form
func ReadFull(witness []byte) (r []fr.Element, err error) {
	if (len(witness) % frSize) != 0 {
		return nil, errors.New("invalid input size")
	}
	r = make([]fr.Element, (len(witness) / frSize))
	offset := 0
	for i := 0; i < len(r); i++ {
		r[i].SetBytes(witness[offset : offset+frSize])
		offset += frSize
	}

	return
}

// ReadPublic decodes publicWitness[]byte -> []fr.Element
// publicWitness is [public], without one_wire
// returned value is in Regular form, and contains the one_wire at position 0
func ReadPublic(publicWitness []byte) (r []fr.Element, err error) {
	if (len(publicWitness) % frSize) != 0 {
		return nil, errors.New("invalid input size")
	}
	r = make([]fr.Element, (1 + len(publicWitness)/frSize))
	r[0].SetOne().FromMont()
	offset := 0
	for i := 1; i < len(r); i++ {
		r[i].SetBytes(publicWitness[offset : offset+frSize]).FromMont()
		offset += frSize
	}
	return
}

func count(w frontend.Witness) (nbSecret, nbPublic int, err error) {
	var collectHandler parser.LeafHandler = func(visibility backend.Visibility, name string, tInput reflect.Value) error {
		if visibility == backend.Secret {
			nbSecret++
		} else if visibility == backend.Public {
			nbPublic++
		}
		return nil
	}
	err = parser.Visit(w, "", backend.Unset, collectHandler, reflect.TypeOf(frontend.Variable{}))
	return
}
