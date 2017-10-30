/*
Copyright 2017 Luis Pabón luis@portworx.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

//go:generate mockgen -source=$GOPATH/src/github.com/container-storage-interface/spec/lib/go/csi/csi.pb.go -imports .=github.com/container-storage-interface/spec/lib/go/csi -package=driver -destination=driver.mock.go
import (
	"net"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type MockCSIDriverServers struct {
	Controller *MockControllerServer
	Identity   *MockIdentityServer
	Node       *MockNodeServer
}

type MockCSIDriver struct {
	listener net.Listener
	server   *grpc.Server
	conn     *grpc.ClientConn
	servers  *MockCSIDriverServers
	wg       sync.WaitGroup
}

func NewMockCSIDriver(servers *MockCSIDriverServers) *MockCSIDriver {
	return &MockCSIDriver{
		servers: servers,
	}
}

func (m *MockCSIDriver) goServe() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.server.Serve(m.listener)
	}()
}

func (m *MockCSIDriver) Address() string {
	return m.listener.Addr().String()
}
func (m *MockCSIDriver) Start() error {

	// Listen on a port assigned by the net package
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	m.listener = l

	// Create a new grpc server
	m.server = grpc.NewServer()

	// Register Mock servers
	if m.servers.Controller != nil {
		csi.RegisterControllerServer(m.server, m.servers.Controller)
	}
	if m.servers.Identity != nil {
		csi.RegisterIdentityServer(m.server, m.servers.Identity)
	}
	if m.servers.Node != nil {
		csi.RegisterNodeServer(m.server, m.servers.Node)
	}
	reflection.Register(m.server)

	// Start listening for requests
	m.goServe()
	return nil
}

func (m *MockCSIDriver) Nexus() (*grpc.ClientConn, error) {
	// Start server
	err := m.Start()
	if err != nil {
		return nil, err
	}

	// Create a client connection
	m.conn, err = grpc.Dial(m.Address(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return m.conn, nil
}

func (m *MockCSIDriver) Stop() {
	m.server.Stop()
	m.wg.Wait()
}

func (m *MockCSIDriver) Close() {
	m.conn.Close()
	m.server.Stop()
}
