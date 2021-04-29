package dumper

import (
	"sync"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/xelabs/go-mysqlstack/driver"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
	"github.com/xelabs/go-mysqlstack/xlog"
)

func TestPool(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	// fakedbs.
	{
		fakedbs.AddQueryPattern("select .*", &sqltypes.Result{})
	}

	pool, err := NewPool(log, 8, address, "mock", "mock", "", "")
	c.Assert(err, qt.IsNil)

	var wg sync.WaitGroup
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ch1:
					return
				default:
					conn := pool.Get()
					err := conn.Execute("select 1")
					c.Assert(err, qt.IsNil)

					_, err = conn.Fetch("select 1")
					c.Assert(err, qt.IsNil)
					_, err = conn.StreamFetch("select 1")
					c.Assert(err, qt.IsNil)

					pool.Put(conn)
				}
			}
		}()
	}

	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ch2:
					return
				default:
					conn := pool.Get()
					err = conn.Execute("select 2")
					c.Assert(err, qt.IsNil)

					_, err = conn.Fetch("select 2")
					c.Assert(err, qt.IsNil)

					_, err = conn.StreamFetch("select 1")
					c.Assert(err, qt.IsNil)

					pool.Put(conn)
				}
			}
		}()
	}

	time.Sleep(time.Second)
	close(ch1)
	close(ch2)
	pool.Close()

	wg.Wait()
}
