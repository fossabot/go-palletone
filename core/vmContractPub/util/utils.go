/*
	This file is part of go-palletone.
	go-palletone is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
	go-palletone is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.
	You should have received a copy of the GNU General Public License
	along with go-palletone.  If not, see <http://www.gnu.org/licenses/>.
*/
/*
 * Copyright IBM Corp. All Rights Reserved.
 * @author PalletOne core developers <dev@pallet.one>
 * @date 2018
 */


package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"time"
	"hash"
	"github.com/studyzy/go-palletone/core/vmContractPub/metadata"
	"github.com/golang/protobuf/ptypes/timestamp"

	"crypto/sha256"
)

type alg struct {
	hashFun func([]byte) string
}

const defaultAlg = "sha256"

var availableIDgenAlgs = map[string]alg{
	defaultAlg: {GenerateIDfromTxSHAHash},
}

func computerHash(data []byte) (hsh []byte, err error){
	var hh hash.Hash
	hh = sha256.New()
	hh.Write(data)
	return hh.Sum(nil), nil
}

// ComputeSHA256 returns SHA2-256 on data
func ComputeSHA256(data []byte) (hsh []byte) {
	hsh, err := computerHash(data)
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA256 on [% x]", data))
	}
	return
}

// ComputeSHA3256 returns SHA3-256 on data
func ComputeSHA3256(data []byte) (hash []byte) {
	hash, err := computerHash(data)
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA3_256 on [% x]", data))
	}
	return
}

// GenerateBytesUUID returns a UUID based on RFC 4122 returning the generated bytes
func GenerateBytesUUID() []byte {
	uuid := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		panic(fmt.Sprintf("Error generating UUID: %s", err))
	}

	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80

	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40

	return uuid
}

// GenerateIntUUID returns a UUID based on RFC 4122 returning a big.Int
func GenerateIntUUID() *big.Int {
	uuid := GenerateBytesUUID()
	z := big.NewInt(0)
	return z.SetBytes(uuid)
}

// GenerateUUID returns a UUID based on RFC 4122
func GenerateUUID() string {
	uuid := GenerateBytesUUID()
	return idBytesToStr(uuid)
}

// CreateUtcTimestamp returns a google/protobuf/Timestamp in UTC
func CreateUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000))
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}

//GenerateHashFromSignature returns a hash of the combined parameters
func GenerateHashFromSignature(path string, args []byte) []byte {
	return ComputeSHA256(args)
}

// GenerateIDfromTxSHAHash generates SHA256 hash using Tx payload
func GenerateIDfromTxSHAHash(payload []byte) string {
	return fmt.Sprintf("%x", ComputeSHA256(payload))
}

// GenerateIDWithAlg generates an ID using a custom algorithm
func GenerateIDWithAlg(customIDgenAlg string, payload []byte) (string, error) {
	if customIDgenAlg == "" {
		customIDgenAlg = defaultAlg
	}
	var alg = availableIDgenAlgs[customIDgenAlg]
	if alg.hashFun != nil {
		return alg.hashFun(payload), nil
	}
	return "", fmt.Errorf("Wrong ID generation algorithm was given: %s", customIDgenAlg)
}

func idBytesToStr(id []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:])
}

// FindMissingElements identifies the elements of the first slice that are not present in the second
// The second slice is expected to be a subset of the first slice
func FindMissingElements(all []string, some []string) (delta []string) {
all:
	for _, v1 := range all {
		for _, v2 := range some {
			if strings.Compare(v1, v2) == 0 {
				continue all
			}
		}
		delta = append(delta, v1)
	}
	return
}

// ToChaincodeArgs converts string args to []byte args
func ToChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

// ArrayToChaincodeArgs converts array of string args to array of []byte args
func ArrayToChaincodeArgs(args []string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

const testchainid = "testchainid"
const testorgid = "**TEST_ORGID**"

//GetTestChainID returns the CHAINID constant in use by orderer
func GetTestChainID() string {
	return testchainid
}

//GetTestOrgID returns the ORGID constant in use by gossip join message
func GetTestOrgID() string {
	return testorgid
}

//GetSysCCVersion returns the version of all system chaincodes
//This needs to be revisited on policies around system chaincode
//"upgrades" from user and relationship with  upgrade. For
//now keep it simple and use the version stamp
func GetSysCCVersion() string {
	return metadata.Version
}

// ConcatenateBytes is useful for combining multiple arrays of bytes, especially for
// signatures or digests over multiple fields
func ConcatenateBytes(data ...[]byte) []byte {
	finalLength := 0
	for _, slice := range data {
		finalLength += len(slice)
	}
	result := make([]byte, finalLength)
	last := 0
	for _, slice := range data {
		for i := range slice {
			result[i+last] = slice[i]
		}
		last += len(slice)
	}
	return result
}

// `flatten` recursively retrieves every leaf node in a struct in depth-first fashion
// and aggregate the results into given string slice with format: "path.to.leaf = value"
// in the order of definition. Root name is ignored in the path. This helper function is
// useful to pretty-print a struct, such as configs.
// for example, given data structure:
// A{
//   B{
//     C: "foo",
//     D: 42,
//   },
//   E: nil,
// }
// it should yield a slice of string containing following items:
// [
//   "B.C = \"foo\"",
//   "B.D = 42",
//   "E =",
// ]
func Flatten(i interface{}) []string {
	var res []string
	flatten("", &res, reflect.ValueOf(i))
	return res
}

const DELIMITER = "."

func flatten(k string, m *[]string, v reflect.Value) {
	delimiter := DELIMITER
	if k == "" {
		delimiter = ""
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			*m = append(*m, fmt.Sprintf("%s =", k))
			return
		}
		flatten(k, m, v.Elem())
	case reflect.Struct:
		if x, ok := v.Interface().(fmt.Stringer); ok {
			*m = append(*m, fmt.Sprintf("%s = %v", k, x))
			return
		}

		for i := 0; i < v.NumField(); i++ {
			flatten(k+delimiter+v.Type().Field(i).Name, m, v.Field(i))
		}
	case reflect.String:
		// It is useful to quote string values
		*m = append(*m, fmt.Sprintf("%s = \"%s\"", k, v))
	default:
		*m = append(*m, fmt.Sprintf("%s = %v", k, v))
	}
}
