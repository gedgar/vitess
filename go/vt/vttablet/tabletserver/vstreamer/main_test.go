/*
Copyright 2019 The Vitess Authors.

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

package vstreamer

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	binlogdatapb "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/stretchr/testify/require"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/vt/dbconfigs"
	"vitess.io/vitess/go/vt/vttablet/tabletserver/schema"
	"vitess.io/vitess/go/vt/vttablet/tabletserver/tabletenv"
	"vitess.io/vitess/go/vt/vttablet/tabletserver/vstreamer/testenv"
)

var (
	engine    *Engine
	env       *testenv.Env
	historian schema.Historian
)

func TestMain(m *testing.M) {
	flag.Parse() // Do not remove this comment, import into google3 depends on it

	if testing.Short() {
		os.Exit(m.Run())
	}

	exitCode := func() int {
		var err error
		env, err = testenv.Init()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return 1
		}
		defer env.Close()

		// engine cannot be initialized in testenv because it introduces
		// circular dependencies
		historian = schema.NewHistorian(env.SchemaEngine)
		historian.Open()
		engine = NewEngine(env.TabletEnv, env.SrvTopo, historian)
		engine.Open(env.KeyspaceName, env.Cells[0])
		defer engine.Close()

		return m.Run()
	}()
	os.Exit(exitCode)
}

func customEngine(t *testing.T, modifier func(mysql.ConnParams) mysql.ConnParams) *Engine {
	original, err := env.Dbcfgs.AppWithDB().MysqlParams()
	require.NoError(t, err)
	modified := modifier(*original)
	config := env.TabletEnv.Config().Clone()
	config.DB = dbconfigs.NewTestDBConfigs(modified, modified, modified.DbName)
	historian = schema.NewHistorian(env.SchemaEngine)
	historian.Open()

	engine := NewEngine(tabletenv.NewEnv(config, "VStreamerTest"), env.SrvTopo, historian)
	engine.Open(env.KeyspaceName, env.Cells[0])
	return engine
}

type mockHistorian struct {
	he schema.HistoryEngine
}

func (h *mockHistorian) SetTrackSchemaVersions(val bool) {}

func (h *mockHistorian) Open() error {
	return nil
}

func (h *mockHistorian) Close() {
}

func (h *mockHistorian) Reload(ctx context.Context) error {
	return nil
}

func newMockHistorian(he schema.HistoryEngine) *mockHistorian {
	sh := mockHistorian{he: he}
	return &sh
}

func (h *mockHistorian) GetTableForPos(tableName sqlparser.TableIdent, pos string) *binlogdatapb.MinimalTable {
	return nil
}

func (h *mockHistorian) RegisterVersionEvent() error {
	numVersionEventsReceived++
	return nil
}

var _ schema.Historian = (*mockHistorian)(nil)
