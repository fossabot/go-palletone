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

package platforms

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"io/ioutil"

	"github.com/studyzy/go-palletone/core/vmContractPub/flogging"
	"github.com/studyzy/go-palletone/core/vmContractPub/metadata"
	"github.com/studyzy/go-palletone/contracts/platforms/golang"
	"github.com/studyzy/go-palletone/contracts/platforms/java"
	"github.com/studyzy/go-palletone/contracts/platforms/node"
	"github.com/studyzy/go-palletone/core/vmContractPub/config"
	cutil "github.com/studyzy/go-palletone/vm/common"
	pb "github.com/studyzy/go-palletone/core/vmContractPub/protos/peer"
	"github.com/spf13/viper"
)

// Interface for validating the specification and and writing the package for
// the given platform
type Platform interface {
	ValidateSpec(spec *pb.ChaincodeSpec) error
	ValidateDeploymentSpec(spec *pb.ChaincodeDeploymentSpec) error
	GetDeploymentPayload(spec *pb.ChaincodeSpec) ([]byte, error)
	GenerateDockerfile(spec *pb.ChaincodeDeploymentSpec) (string, error)
	GenerateDockerBuild(spec *pb.ChaincodeDeploymentSpec, tw *tar.Writer) error
}

var logger = flogging.MustGetLogger("chaincode-platform")

// Added for unit testing purposes
var _Find = Find
var _GetPath = config.GetPath
var _VGetBool = viper.GetBool
var _OSStat = os.Stat
var _IOUtilReadFile = ioutil.ReadFile
var _CUtilWriteBytesToPackage = cutil.WriteBytesToPackage
var _generateDockerfile = generateDockerfile
var _generateDockerBuild = generateDockerBuild

// Find returns the platform interface for the given platform type
func Find(chaincodeType pb.ChaincodeSpec_Type) (Platform, error) {

	switch chaincodeType {
	case pb.ChaincodeSpec_GOLANG:
		return &golang.Platform{}, nil
	case pb.ChaincodeSpec_JAVA:
		return &java.Platform{}, nil
	case pb.ChaincodeSpec_NODE:
		return &node.Platform{}, nil
	default:
		return nil, fmt.Errorf("Unknown chaincodeType: %s", chaincodeType)
	}

}

func GetDeploymentPayload(spec *pb.ChaincodeSpec) ([]byte, error) {
	platform, err := _Find(spec.Type)
	if err != nil {
		return nil, err
	}

	return platform.GetDeploymentPayload(spec)
}

func generateDockerfile(platform Platform, cds *pb.ChaincodeDeploymentSpec) ([]byte, error) {

	var buf []string

	// ----------------------------------------------------------------------------------------------------
	// Let the platform define the base Dockerfile
	// ----------------------------------------------------------------------------------------------------
	base, err := platform.GenerateDockerfile(cds)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate platform-specific Dockerfile: %s", err)
	}
	buf = append(buf, base)

	// ----------------------------------------------------------------------------------------------------
	// Add some handy labels
	// ----------------------------------------------------------------------------------------------------
	buf = append(buf, fmt.Sprintf("LABEL %s.chaincode.id.name=\"%s\" \\", metadata.BaseDockerLabel, cds.ChaincodeSpec.ChaincodeId.Name))
	buf = append(buf, fmt.Sprintf("      %s.chaincode.id.version=\"%s\" \\", metadata.BaseDockerLabel, cds.ChaincodeSpec.ChaincodeId.Version))
	buf = append(buf, fmt.Sprintf("      %s.chaincode.type=\"%s\" \\", metadata.BaseDockerLabel, cds.ChaincodeSpec.Type.String()))
	buf = append(buf, fmt.Sprintf("      %s.version=\"%s\" \\", metadata.BaseDockerLabel, metadata.Version))
	buf = append(buf, fmt.Sprintf("      %s.base.version=\"%s\"", metadata.BaseDockerLabel, metadata.BaseVersion))
	// ----------------------------------------------------------------------------------------------------
	// Then augment it with any general options
	// ----------------------------------------------------------------------------------------------------
	//append version so chaincode build version can be compared against peer build version
	buf = append(buf, fmt.Sprintf("ENV CORE_CHAINCODE_BUILDLEVEL=%s", metadata.Version))

	// ----------------------------------------------------------------------------------------------------
	// Finalize it
	// ----------------------------------------------------------------------------------------------------
	contents := strings.Join(buf, "\n")
	logger.Debugf("\n%s", contents)

	return []byte(contents), nil
}

type InputFiles map[string][]byte

func generateDockerBuild(platform Platform, cds *pb.ChaincodeDeploymentSpec, inputFiles InputFiles, tw *tar.Writer) error {

	var err error

	// ----------------------------------------------------------------------------------------------------
	// First stream out our static inputFiles
	// ----------------------------------------------------------------------------------------------------
	for name, data := range inputFiles {
		err = _CUtilWriteBytesToPackage(name, data, tw)
		if err != nil {
			return fmt.Errorf("Failed to inject \"%s\": %s", name, err)
		}
	}

	// ----------------------------------------------------------------------------------------------------
	// Now give the platform an opportunity to contribute its own context to the build
	// ----------------------------------------------------------------------------------------------------
	err = platform.GenerateDockerBuild(cds, tw)
	if err != nil {
		return fmt.Errorf("Failed to generate platform-specific docker build: %s", err)
	}

	return nil
}

func GenerateDockerBuild(cds *pb.ChaincodeDeploymentSpec) (io.Reader, error) {

	inputFiles := make(InputFiles)

	// ----------------------------------------------------------------------------------------------------
	// Determine our platform driver from the spec
	// ----------------------------------------------------------------------------------------------------
	platform, err := _Find(cds.ChaincodeSpec.Type)
	if err != nil {
		return nil, fmt.Errorf("Failed to determine platform type: %s", err)
	}

	// ----------------------------------------------------------------------------------------------------
	// Generate the Dockerfile specific to our context
	// ----------------------------------------------------------------------------------------------------
	dockerFile, err := _generateDockerfile(platform, cds)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate a Dockerfile: %s", err)
	}

	inputFiles["Dockerfile"] = dockerFile

	// ----------------------------------------------------------------------------------------------------
	// Finally, launch an asynchronous process to stream all of the above into a docker build context
	// ----------------------------------------------------------------------------------------------------
	input, output := io.Pipe()

	go func() {
		gw := gzip.NewWriter(output)
		tw := tar.NewWriter(gw)
		err := _generateDockerBuild(platform, cds, inputFiles, tw)
		if err != nil {
			logger.Error(err)
		}

		tw.Close()
		gw.Close()
		output.CloseWithError(err)
	}()

	return input, nil
}
