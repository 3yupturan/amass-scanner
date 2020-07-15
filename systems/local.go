// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package systems

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/OWASP/Amass/v3/config"
	"github.com/OWASP/Amass/v3/graph"
	"github.com/OWASP/Amass/v3/graphdb"
	"github.com/OWASP/Amass/v3/requests"
	"github.com/OWASP/Amass/v3/resolvers"
)

type memRequest struct {
	Result chan bool
}

// LocalSystem implements a System to be executed within a single process.
type LocalSystem struct {
	sync.Mutex

	cfg    *config.Config
	pool   resolvers.Resolver
	graphs []*graph.Graph

	// The various services running within the system
	dataSources []requests.Service

	// Broadcast channel that indicates no further writes to the output channel
	done              chan struct{}
	doneAlreadyClosed bool
	memReq            chan *memRequest
}

// NewLocalSystem returns an initialized LocalSystem object.
func NewLocalSystem(c *config.Config) (*LocalSystem, error) {
	if err := c.CheckSettings(); err != nil {
		return nil, err
	}

	pool := resolvers.SetupResolverPool(
		c.Resolvers,
		c.MonitorResolverRate,
		c.Log,
	)
	if pool == nil {
		return nil, errors.New("The system was unable to build the pool of resolvers")
	}

	sys := &LocalSystem{
		cfg:    c,
		pool:   pool,
		done:   make(chan struct{}, 2),
		memReq: make(chan *memRequest, 2),
	}

	// Setup the correct graph database handler
	if err := sys.setupGraphDBs(); err != nil {
		sys.Shutdown()
		return nil, err
	}

	go sys.memConsumptionMonitor()
	return sys, nil
}

// Config implements the System interface.
func (l *LocalSystem) Config() *config.Config {
	return l.cfg
}

// Pool implements the System interface.
func (l *LocalSystem) Pool() resolvers.Resolver {
	return l.pool
}

// AddSource implements the System interface.
func (l *LocalSystem) AddSource(srv requests.Service) error {
	l.Lock()
	defer l.Unlock()

	l.dataSources = append(l.dataSources, srv)
	return nil
}

// AddAndStart implements the System interface.
func (l *LocalSystem) AddAndStart(srv requests.Service) error {
	err := srv.Start()

	if err == nil {
		return l.AddSource(srv)
	}
	return err
}

// DataSources implements the System interface.
func (l *LocalSystem) DataSources() []requests.Service {
	l.Lock()
	defer l.Unlock()

	return l.dataSources
}

// SetDataSources assigns the data sources that will be used by the system.
func (l *LocalSystem) SetDataSources(sources []requests.Service) {
	// Add all the data sources that successfully start to the list
	for _, src := range sources {
		l.AddAndStart(src)
	}
}

// GraphDatabases implements the System interface.
func (l *LocalSystem) GraphDatabases() []*graph.Graph {
	return l.graphs
}

// Shutdown implements the System interface.
func (l *LocalSystem) Shutdown() error {
	if !l.doneAlreadyClosed {
		l.doneAlreadyClosed = true
		close(l.done)
	}

	for _, src := range l.DataSources() {
		src.Stop()
	}

	for _, g := range l.GraphDatabases() {
		g.Close()
	}

	l.pool.Stop()
	return nil
}

// GetAllSourceNames returns the names of all the available data sources.
func (l *LocalSystem) GetAllSourceNames() []string {
	var names []string

	for _, src := range l.DataSources() {
		names = append(names, src.String())
	}
	return names
}

// Select the graph that will store the System findings.
func (l *LocalSystem) setupGraphDBs() error {
	c := l.Config()

	if c.GremlinURL != "" {
		gremlin := graphdb.NewGremlin(c.GremlinURL, c.GremlinUser, c.GremlinPass)
		if gremlin == nil {
			return fmt.Errorf("System: Failed to create the %s graph", gremlin.String())
		}

		g := graph.NewGraph(gremlin)
		if g == nil {
			return fmt.Errorf("System: Failed to create the %s graph", g.String())
		}

		l.graphs = append(l.graphs, g)
	}

	dir := config.OutputDirectory(c.Dir)
	if c.LocalDatabase && dir != "" {
		cayley := graphdb.NewCayleyGraph(dir)
		if cayley == nil {
			return fmt.Errorf("System: Failed to create the %s graph", cayley.String())
		}

		g := graph.NewGraph(cayley)
		if g == nil {
			return fmt.Errorf("System: Failed to create the %s graph", g.String())
		}

		l.graphs = append(l.graphs, g)
	}

	return nil
}

// HighMemoryConsumption implements the System interface.
func (l *LocalSystem) HighMemoryConsumption() bool {
	if l.doneAlreadyClosed {
		return false
	}

	result := make(chan bool, 2)

	l.memReq <- &memRequest{Result: result}
	return <-result
}

func (l *LocalSystem) memConsumptionMonitor() {
	var count int
	var highConsumption bool
	var prevAlloc, curNormal uint64

	interval := 10 * time.Second
	maxCount := int((2 * time.Minute) / interval)
	curNormal = 1073741824 // one gigabyte

	t := time.NewTicker(interval)
	defer t.Stop()
loop:
	for {
		select {
		case <-l.done:
			break loop
		case req := <-l.memReq:
			req.Result <- highConsumption
		case <-t.C:
			var stats runtime.MemStats

			highConsumption = false
			runtime.ReadMemStats(&stats)
			if count >= maxCount && stats.Alloc > prevAlloc {
				curNormal += curNormal / 4
				count = 0
			}
			if stats.Alloc > curNormal {
				highConsumption = true
				count++
			} else {
				count = 0
			}
			prevAlloc = stats.Alloc
		}
	}

	// Empty the request channel
	for {
		select {
		case req := <-l.memReq:
			req.Result <- false
		default:
			close(l.memReq)
			return
		}
	}
}
