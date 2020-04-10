package multiplex

import (
	"github.com/cbeuw/connutil"
	"math/rand"
	"testing"
	"time"
)

func TestSwitchboard_Send(t *testing.T) {
	doTest := func(seshConfig SessionConfig) {
		sesh := MakeSession(0, seshConfig)
		hole0 := connutil.Discard()
		sesh.sb.addConn(hole0)
		connId, _, err := sesh.sb.pickRandConn()
		if err != nil {
			t.Error("failed to get a random conn", err)
			return
		}
		data := make([]byte, 1000)
		rand.Read(data)
		_, err = sesh.sb.send(data, &connId)
		if err != nil {
			t.Error(err)
			return
		}

		hole1 := connutil.Discard()
		sesh.sb.addConn(hole1)
		connId, _, err = sesh.sb.pickRandConn()
		if err != nil {
			t.Error("failed to get a random conn", err)
			return
		}
		_, err = sesh.sb.send(data, &connId)
		if err != nil {
			t.Error(err)
			return
		}

		connId, _, err = sesh.sb.pickRandConn()
		if err != nil {
			t.Error("failed to get a random conn", err)
			return
		}
		_, err = sesh.sb.send(data, &connId)
		if err != nil {
			t.Error(err)
			return
		}
	}

	t.Run("Ordered", func(t *testing.T) {
		seshConfig := SessionConfig{
			Unordered: false,
		}
		doTest(seshConfig)
	})
	t.Run("Unordered", func(t *testing.T) {
		seshConfig := SessionConfig{
			Unordered: true,
		}
		doTest(seshConfig)
	})
}

func BenchmarkSwitchboard_Send(b *testing.B) {
	hole := connutil.Discard()
	seshConfig := SessionConfig{}
	sesh := MakeSession(0, seshConfig)
	sesh.sb.addConn(hole)
	connId, _, err := sesh.sb.pickRandConn()
	if err != nil {
		b.Error("failed to get a random conn", err)
		return
	}
	data := make([]byte, 1000)
	rand.Read(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, err := sesh.sb.send(data, &connId)
		if err != nil {
			b.Error(err)
			return
		}
		b.SetBytes(int64(n))
	}
}

func TestSwitchboard_TxCredit(t *testing.T) {
	seshConfig := SessionConfig{
		Valve: MakeValve(1<<20, 1<<20),
	}
	sesh := MakeSession(0, seshConfig)
	hole := connutil.Discard()
	sesh.sb.addConn(hole)
	connId, _, err := sesh.sb.pickRandConn()
	if err != nil {
		t.Error("failed to get a random conn", err)
		return
	}
	data := make([]byte, 1000)
	rand.Read(data)

	t.Run("FIXED CONN MAPPING", func(t *testing.T) {
		*sesh.sb.valve.(*LimitedValve).tx = 0
		sesh.sb.strategy = FIXED_CONN_MAPPING
		n, err := sesh.sb.send(data[:10], &connId)
		if err != nil {
			t.Error(err)
			return
		}
		if n != 10 {
			t.Errorf("wanted to send %v, got %v", 10, n)
			return
		}
		if *sesh.sb.valve.(*LimitedValve).tx != 10 {
			t.Error("tx credit didn't increase by 10")
		}
	})
	t.Run("UNIFORM", func(t *testing.T) {
		*sesh.sb.valve.(*LimitedValve).tx = 0
		sesh.sb.strategy = UNIFORM_SPREAD
		n, err := sesh.sb.send(data[:10], &connId)
		if err != nil {
			t.Error(err)
			return
		}
		if n != 10 {
			t.Errorf("wanted to send %v, got %v", 10, n)
			return
		}
		if *sesh.sb.valve.(*LimitedValve).tx != 10 {
			t.Error("tx credit didn't increase by 10")
		}
	})
}

func TestSwitchboard_CloseOnOneDisconn(t *testing.T) {
	sesh := setupSesh(false)

	conn0client, conn0server := connutil.AsyncPipe()
	sesh.AddConnection(conn0client)

	conn1client, _ := connutil.AsyncPipe()
	sesh.AddConnection(conn1client)

	conn0server.Close()
	time.Sleep(500 * time.Millisecond)
	if !sesh.IsClosed() {
		t.Error("session not closed after one conn is disconnected")
		return
	}
	if _, err := conn1client.Write([]byte{0x00}); err == nil {
		t.Error("the other conn is still connected")
		return
	}
}
