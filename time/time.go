package time

import (
	"sync"
	"time"

	"github.com/beevik/ntp"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

var ntpServer = []string{"ntp.aliyun.com", "ntp1.aliyun.com", "ntp2.aliyun.com", "ntp3.aliyun.com", "ntp4.aliyun.com",
	"ntp4.aliyun.com", "ntp5.aliyun.com", "ntp6.aliyun.com", "ntp7.aliyun.com", "pool.ntp.org", "asia.pool.ntp.org",
	"europe.pool.ntp.org", "north-america.pool.ntp.org", "oceania.pool.ntp.org", "south-america.pool.ntp.org"}

var cTime ChainTime
var onceChainTime sync.Once

type ChainTime interface {
	Now() TimeStamp

	SinceSeconds(t TimeStamp) int64

	// NowAfter checks if current timestamp greater than the given one
	NowAfter(t TimeStamp) bool
}

type TimeStamp int64

func Int64MilliSecondsToTimeStamp(milliSec int64) TimeStamp {
	return TimeStamp(milliSec)
}

func TimeToTimeStamp(t time.Time) TimeStamp {
	return TimeStamp(t.UnixNano() / int64(time.Millisecond))
}

func (ts TimeStamp) toInt64() int64 {
	return int64(ts)
}

func (ts TimeStamp) Bytes() []byte {
	return tpcmm.NumberToByte(ts.toInt64())
}

func (ts TimeStamp) UTC() time.Time {
	return time.Unix(ts.Unix(), 0).UTC()
}

func (ts TimeStamp) Local() time.Time {
	return time.Unix(0, ts.toInt64()*int64(time.Millisecond)).Local()
}

func (ts TimeStamp) Unix() int64 {
	return ts.UnixMilli() / 1e3
}

func (ts TimeStamp) UnixMilli() int64 {
	return ts.toInt64()
}

func (ts TimeStamp) After(t TimeStamp) bool {
	return ts > t
}

func (ts TimeStamp) SinceSeconds(t TimeStamp) int64 {
	return int64(ts-t) / 1e3
}

func (ts TimeStamp) SinceMilliSeconds(t TimeStamp) int64 {
	return int64(ts - t)
}

func (ts TimeStamp) AddSeconds(sec int64) TimeStamp {
	return ts + TimeStamp(sec*1e3)
}

func (ts TimeStamp) AddMilliSeconds(milliSec int64) TimeStamp {
	return ts + TimeStamp(milliSec)
}

func (ts TimeStamp) String() string {
	return ts.Local().String()
}

type chainTime struct {
	log       tplog.Logger
	ntpOffset time.Duration
	timerMng  TimerManager
}

func GetChainTime(log tplog.Logger) ChainTime {
	onceChainTime.Do(func() {
		chainTime := &chainTime{
			log:       log,
			ntpOffset: 0,
			timerMng:  NewTimerManager("time_sync"),
		}

		chainTime.timerMng.RegisterPeriodicTimer("time_sync", chainTime.syncRoutine, 60)
		chainTime.timerMng.StartTimer("time_sync", false)
		chainTime.syncRoutine()

		cTime = chainTime
	})

	return cTime
}

func (t *chainTime) syncRoutine() bool {
	for _, server := range ntpServer {
		rsp, err := ntp.QueryWithOptions(server, ntp.QueryOptions{Timeout: time.Second})
		if err != nil {
			continue
		}
		t.ntpOffset = rsp.ClockOffset
		if t.ntpOffset.Seconds() > 1 {
			t.log.Warnf("time offset from %v is %v", server, t.ntpOffset.String())
		}
		return true
	}
	t.log.Info("time sync timeout")
	return false
}

func (t *chainTime) Now() TimeStamp {
	return TimeToTimeStamp(time.Now().Add(t.ntpOffset).UTC())
}

func (t *chainTime) SinceSeconds(ts TimeStamp) int64 {
	rs := t.Now().SinceSeconds(ts)
	return rs
}

func (t *chainTime) NowAfter(ts TimeStamp) bool {
	return t.Now().After(ts)
}
